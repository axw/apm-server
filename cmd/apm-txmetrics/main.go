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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/elastic/apm-server/beater/kafkaconsumer"
	"github.com/elastic/apm-server/beater/kafkaproducer"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/modelprocessor"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/txmetrics"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func main() {
	var broker string
	flag.StringVar(&broker, "broker", "localhost:9092", "The Kafka broker address from which events should be consumed")
	flag.Parse()

	logp.DevelopmentSetup(logp.WithSelectors("*"))

	host, _ := os.Hostname()
	log.SetPrefix(fmt.Sprintf("[%s] ", host))

	// Create a context that will be cancelled when an interrupt is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	groupedTransactionsConsumer, err := kafkaconsumer.New(kafkaconsumer.Config{
		Brokers:         []string{broker},
		Topics:          []string{"apm-grouped-transactions"},
		ConsumerGroupID: "apm-txmetrics",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer groupedTransactionsConsumer.Close()

	logEventsProcessor := model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
		for _, event := range *batch {
			log.Printf("%+v", event)
			if event.Transaction != nil {
				log.Printf("\ttransaction: %+v", event.Transaction)
			}
		}
		return nil
	})

	producer, err := kafkaproducer.New(kafkaproducer.Config{
		Topic:  "elastic-apm-events",
		Broker: broker,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	agg, err := txmetrics.NewAggregator(txmetrics.AggregatorConfig{
		BatchProcessor: modelprocessor.Chained{
			logEventsProcessor,
			producer,
		},
		MaxTransactionGroups:           10000,
		MetricsInterval:                10 * time.Second,
		HDRHistogramSignificantFigures: 2,
		Logger:                         logp.NewLogger("txmetrics"),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("consuming events")
	go agg.Run()

	// NOTE(axw) single-event metrics will not be published
	// to Kafka, as we don't process the batch after it has
	// been aggregated. (Single-event metrics are appended
	// to the input batch.)
	if err := groupedTransactionsConsumer.Consume(ctx, agg); err != nil {
		log.Fatal(err)
	}
}
