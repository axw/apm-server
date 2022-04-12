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
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/elastic/apm-server/beater/kafkaconsumer"
	"github.com/elastic/apm-server/beater/kafkaproducer"
	"github.com/elastic/apm-server/elasticsearch"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/modelindexer"
	"github.com/elastic/apm-server/model/modelprocessor"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/txmetrics"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func main() {
	var broker, topic string
	flag.StringVar(&broker, "broker", "localhost:9092", "The Kafka broker address from which events should be consumed")
	flag.StringVar(&topic, "topic", "elastic-apm-events", "The Kafka topic from which events should be consumed")
	flag.Parse()

	logp.DevelopmentSetup(logp.WithSelectors("*"))

	host, _ := os.Hostname()
	log.SetPrefix(fmt.Sprintf("[%s] ", host))

	// Create a context that will be cancelled when an interrupt is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	consumer, err := kafkaconsumer.New(kafkaconsumer.Config{
		Brokers:         []string{broker},
		Topics:          []string{topic},
		ConsumerGroupID: "apm-backend",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	transactionGroupTopicProducer, err := kafkaproducer.New(kafkaproducer.Config{
		Broker: broker,
		Topic:  "apm-grouped-transactions",
		Key: func(event model.APMEvent) []byte {
			var buf bytes.Buffer
			k := txmetrics.MakeTransactionAggregationKey(event, 30*time.Second)
			k.WriteTo(&buf)
			return buf.Bytes()
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer transactionGroupTopicProducer.Close()

	destmetricsEventsTopicProducer, err := kafkaproducer.New(kafkaproducer.Config{
		Broker: broker,
		Topic:  "apm-destmetrics-events",
		Key: func(event model.APMEvent) []byte {
			// Partition by instrumented service.
			var data []byte
			data = append(data, event.Agent.Name...)
			data = append(data, event.Service.Name...)
			data = append(data, event.Service.Environment...)
			return data
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer destmetricsEventsTopicProducer.Close()

	escfg := elasticsearch.DefaultConfig()
	escfg.Hosts = elasticsearch.Hosts{os.Getenv("ELASTICSEARCH_URL")}
	escfg.Username = os.Getenv("ELASTICSEARCH_USER")
	escfg.Password = os.Getenv("ELASTICSEARCH_PASSWORD")
	escfg.TLS = &tlscommon.Config{
		VerificationMode: tlscommon.VerifyNone, // TODO(axw) configure CA cert
	}
	esclient, err := elasticsearch.NewClient(escfg)
	if err != nil {
		log.Fatal(err)
	}
	indexer, err := modelindexer.New(esclient, modelindexer.Config{
		FlushBytes:    1024 * 1024,
		FlushInterval: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	var eventCounter int64
	go func() {
		for range time.Tick(10 * time.Second) {
			log.Printf("consumed %d events", atomic.LoadInt64(&eventCounter))
		}
	}()

	processor := modelprocessor.Chained{
		modelprocessor.SetHostHostname{},
		modelprocessor.SetServiceNodeName{},
		modelprocessor.SetMetricsetName{},
		modelprocessor.SetGroupingKey{},
		modelprocessor.SetErrorMessage{},

		model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
			atomic.AddInt64(&eventCounter, int64(len(*batch)))
			return nil
		}),

		// TODO observer info here? in the frontend? both?
		model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
			for i := range *batch {
				obs := &(*batch)[i].Observer
				obs.Type = "apm-server"
				obs.Version = "8.1.0"
			}
			return nil
		}),

		// TODO set ecs.version depending on the output version
		&modelprocessor.SetDataStream{Namespace: "default"},
		modelprocessor.SetUnknownSpanType{},
		// TODO default service.environment

		// Send transactions keyed by transaction group to another topic for aggregation.
		model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
			for i, event := range *batch {
				if event.Processor != model.TransactionProcessor ||
					event.Transaction == nil || event.Transaction.RepresentativeCount <= 0 {
					continue
				}
				single := (*batch)[i : i+1]
				if err := transactionGroupTopicProducer.ProcessBatch(ctx, &single); err != nil {
					return err
				}
			}
			return nil
		}),

		// Send transactions and spans keyed by service to another topic for service-destination metrics aggregation.
		model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
			for i, event := range *batch {
				if event.Processor != model.TransactionProcessor && event.Processor != model.SpanProcessor {
					continue
				}
				single := (*batch)[i : i+1]
				if err := destmetricsEventsTopicProducer.ProcessBatch(ctx, &single); err != nil {
					return err
				}
			}
			return nil
		}),

		// Send events to Elasticsearch.
		indexer,
	}

	log.Printf("consuming events")
	if err := consumer.Consume(ctx, processor); err != nil {
		log.Fatal(err)
	}
}
