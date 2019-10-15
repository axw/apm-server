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

package otel

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/jaeger"

	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/apm-server/beater/request"
	"github.com/elastic/apm-server/processor/stream"
)

var (
	// MonitoringMap holds a mapping for request.IDs to monitoring counters
	MonitoringMap = request.MonitoringMapForRegistry(registry)
	registry      = monitoring.Default.NewRegistry("apm-server.otel", monitoring.PublishExpvar)
)

type Config struct {
	PathPrefix    string
	TraceConsumer consumer.TraceConsumer
}

// JaegerHandler returns a request.Handler for receiving Jaeger data,
// transforming them to, and reporting, Elastic APM model objects.
func JaegerHandler(config Config) request.Handler {
	// app.NewAPIHandler binds the router to a handler
	// that does not propagate request context, so the
	// only way we could trace within these requests
	// would be by creating a new router per request.
	//router := mux.NewRouter().PathPrefix(config.PathPrefix).Subrouter()
	router := mux.NewRouter()
	apiHandler := app.NewAPIHandler(&jaegerBatchSubmitter{TraceConsumer: config.TraceConsumer})
	apiHandler.RegisterRoutes(router)

	return func(c *request.Context) {
		ipRateLimiter := c.RateLimiter.ForIP(c.Request)
		ok := ipRateLimiter == nil || ipRateLimiter.Allow()
		if !ok {
			sendError(c, &stream.Error{
				Type:    stream.RateLimitErrType,
				Message: "rate limit exceeded",
			})
			return
		}
		router.ServeHTTP(c.ResponseWriter, c.Request)
	}
}

type jaegerBatchSubmitter struct {
	TraceConsumer consumer.TraceConsumer
}

func (jbs *jaegerBatchSubmitter) SubmitBatches(batches []*jaeger.Batch, options app.SubmitBatchOptions) ([]*jaeger.BatchSubmitResponse, error) {
	// TODO(axw) the API handler is
	ctx := context.Background()
	ctxWithReceiverName := observability.ContextWithReceiverName(ctx, "jaeger-collector")
	return consumeTraceData(ctxWithReceiverName, batches, jbs.TraceConsumer)
}

func consumeTraceData(ctx context.Context, batches []*jaeger.Batch, consumer consumer.TraceConsumer) ([]*jaeger.BatchSubmitResponse, error) {
	jbsr := make([]*jaeger.BatchSubmitResponse, 0, len(batches))
	var errs []error
	for _, batch := range batches {
		td, err := jaegertranslator.ThriftBatchToOCProto(batch)
		if err != nil {
			jbsr = append(jbsr, &jaeger.BatchSubmitResponse{Ok: false})
			errs = append(errs, err)
			continue
		}
		td.SourceFormat = "jaeger"
		consumer.ConsumeTraceData(ctx, td)
		// We MUST unconditionally record metrics from this reception.
		observability.RecordMetricsForTraceReceiver(ctx, len(batch.Spans), len(batch.Spans)-len(td.Spans))
		jbsr = append(jbsr, &jaeger.BatchSubmitResponse{Ok: true})
	}

	return jbsr, nil
}

func sendError(c *request.Context, err error) {
	sr := stream.Result{}
	sr.Add(err)
	sendResponse(c, &sr)
}

func sendResponse(c *request.Context, sr *stream.Result) {
	// TODO(axw)
}
