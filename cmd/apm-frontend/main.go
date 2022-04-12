// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Shopify/sarama"

	"github.com/elastic/apm-server/agentcfg"
	"github.com/elastic/apm-server/beater/api"
	"github.com/elastic/apm-server/beater/auth"
	"github.com/elastic/apm-server/beater/config"
	"github.com/elastic/apm-server/beater/kafkaproducer"
	"github.com/elastic/apm-server/beater/ratelimit"
	"github.com/elastic/beats/v7/libbeat/beat"
)

var maxScannerBufSize = 300 * 1024 // APM Server default

func main() {
	var listen, broker, topic string
	flag.StringVar(&listen, "listen", ":8200", "host:port on which the server will listen for events")
	flag.StringVar(&broker, "broker", "localhost:9092", "The Kafka broker address to which events should be produced")
	flag.StringVar(&topic, "topic", "elastic-apm-events", "The Kafka topic to which events should be produced")
	flag.Parse()

	// Create a context that will be cancelled when an interrupt is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	// Create topics.
	topics := map[string]*sarama.TopicDetail{
		topic:                      {NumPartitions: 30, ReplicationFactor: 1},
		"apm-grouped-transactions": {NumPartitions: 10, ReplicationFactor: 1},
		"apm-destmetrics-events":   {NumPartitions: 10, ReplicationFactor: 1},
	}
	clusterAdmin, err := sarama.NewClusterAdmin([]string{broker}, nil)
	if err != nil {
		log.Fatal(err)
	}
	for topic, detail := range topics {
		var topicError *sarama.TopicError
		err := clusterAdmin.CreateTopic(topic, detail, false)
		if errors.As(err, &topicError) && topicError.Err == sarama.ErrTopicAlreadyExists {
			log.Printf("topic %q already exists", topic)
		} else if err != nil {
			log.Fatalf("failed to create topic %q: %s (%#v)", topic, err, err)
		} else {
			log.Printf("created topic %q", topic)
		}
	}
	if err := clusterAdmin.Close(); err != nil {
		log.Fatal(err)
	}

	producer, err := kafkaproducer.New(kafkaproducer.Config{Broker: broker, Topic: topic})
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	cfg := config.DefaultConfig()
	authenticator, _ := auth.NewAuthenticator(config.AgentAuth{})
	ratelimitStore, _ := ratelimit.NewStore(1, 1, 1) // unused, arbitrary params
	router, err := api.NewMux(
		beat.Info{},
		cfg,
		producer,
		authenticator,
		agentcfg.NewFetcher(cfg),
		ratelimitStore,
		nil,                         // no sourcemap store
		false,                       // not managed by Fleet
		func() bool { return true }, // ready for publishing
	)
	if err != nil {
		log.Fatal(err)
	}
	srv := http.Server{
		Addr:        listen,
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Println("Closing http server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Println(err)
		}
	}()

	log.Printf("Listening for requests on http://%s%s", srv.Addr, "/intake/v2/events")
	log.Printf("Sending events to Kafka broker %s", broker)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}
}
