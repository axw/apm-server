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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/jaegertracing/jaeger/thrift-gen/sampling"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/receiver/zipkinreceiver"
	jaegertranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace/jaeger"

	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/apm-server/agentcfg"
	"github.com/elastic/apm-server/beater/request"
	"github.com/elastic/apm-server/processor/stream"
)

var (
	// MonitoringMap holds a mapping for request.IDs to monitoring counters
	MonitoringMap = request.MonitoringMapForRegistry(registry)
	registry      = monitoring.Default.NewRegistry("apm-server.otel", monitoring.PublishExpvar)
)

type Config struct {
	PathPrefix         string
	TraceConsumer      consumer.TraceConsumer
	AgentConfigFetcher *agentcfg.Fetcher
}

/*
// OpenCensusHandler returns a request.Handler for receiving OpenCensus data,
// transforming them to, and reporting, Elastic APM model objects.
func OpenCensusHandler(config Config) request.Handler {
	// TODO(axw) ensure server can negotiate HTTP/2
	// TODO(axw) also register grpc-gateway paths?
	//
	// TODO(axw) the octrace.Receiver needs to be stopped with the server.
	tr, err := octrace.New(config.TraceConsumer)
	if err != nil {
		// TODO(axw) don't panic, return an error.
		panic(err)
	}
	agenttracepb.RegisterTraceServiceServer(server, tr)
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
		// NOTE(axw) using grpc.Server with net/http.Server is considered
		// experimental. Should we have a separate endpoint that uses
		// grpc-go's internal http/2 implementation?
		server.ServeHTTP(c.ResponseWriter, c.Request)
	}
}
*/

// ZipkinHandler returns a request.Handler for receiving Zipkin data,
// transforming them to, and reporting, Elastic APM model objects.
func ZipkinHandler(config Config) request.Handler {
	zr, err := zipkinreceiver.New("", config.TraceConsumer)
	if err != nil {
		panic(err)
	}
	handler := http.StripPrefix(strings.TrimSuffix(config.PathPrefix, "/"), zr)

	return func(c *request.Context) {
		ok := c.RateLimiter == nil || c.RateLimiter.Allow()
		if !ok {
			sendError(c, &stream.Error{
				Type:    stream.RateLimitErrType,
				Message: "rate limit exceeded",
			})
			return
		}
		handler.ServeHTTP(c.ResponseWriter, c.Request)
	}
}

// JaegerHandler returns a request.Handler for receiving Jaeger data,
// transforming them to, and reporting, Elastic APM model objects.
func JaegerHandler(config Config) request.Handler {
	// app.NewAPIHandler binds the router to a handler
	// that does not propagate request context, so the
	// only way we could trace within these requests
	// would be by creating a new router per request.
	router := mux.NewRouter().PathPrefix(config.PathPrefix).Subrouter()
	apiHandler := app.NewAPIHandler(&jaegerBatchSubmitter{TraceConsumer: config.TraceConsumer})
	apiHandler.RegisterRoutes(router)

	router.HandleFunc("/sampling", func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.URL.Query().Get("service")
		if serviceName == "" {
			http.Error(w, "'service' query parameter missing", http.StatusBadRequest)
			return
		}
		// TODO(axw) if Kibana is disabled/unavailable, should we return an error or a default strategy?
		if config.AgentConfigFetcher == nil {
			http.Error(w, "kibana connection is disabled", http.StatusServiceUnavailable)
			return
		}

		query := agentcfg.NewQuery(serviceName, "" /* environment */)
		result, err := config.AgentConfigFetcher.Fetch(query)
		if err != nil {
			// TODO(axw) convert error
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if result.Source.Etag == "" {
			http.Error(w, fmt.Sprintf("config for service %q not found", serviceName), http.StatusNotFound)
			return
		}

		response := sampling.NewSamplingStrategyResponse()
		if v, ok := result.Source.Settings["transaction_sample_rate"]; ok {
			rate, err := strconv.ParseFloat(v, 64)
			if err == nil {
				response.StrategyType = sampling.SamplingStrategyType_PROBABILISTIC
				response.ProbabilisticSampling = sampling.NewProbabilisticSamplingStrategy()
				response.ProbabilisticSampling.SamplingRate = rate
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// TODO(axw) this isn't working. Why?
		// Fetch again to mark the config as applied.
		//query.Etag = result.Source.Etag
		//pretty.Println(config.AgentConfigFetcher.Fetch(query))
	})

	return func(c *request.Context) {
		ok := c.RateLimiter == nil || c.RateLimiter.Allow()
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
	ctx := context.Background()
	ctxWithReceiverName := observability.ContextWithReceiverName(ctx, "jaeger-collector")
	return consumeJaegerTraceData(ctxWithReceiverName, batches, jbs.TraceConsumer)
}

func consumeJaegerTraceData(ctx context.Context, batches []*jaeger.Batch, consumer consumer.TraceConsumer) ([]*jaeger.BatchSubmitResponse, error) {
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
