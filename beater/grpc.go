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

package beater

import (
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgrpc"
	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/apm-server/beater/config"
)

func newGRPCServer(logger *logp.Logger, cfg *config.Config, tracer *apm.Tracer) (*grpc.Server, error) {
	grpcOptions := []grpc.ServerOption{grpc.UnaryInterceptor(
		apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery(), apmgrpc.WithTracer(tracer)),
	)}
	// NOTE(axw) we must not enable TLS creds here if we're using h2mux,
	// as TLS will already have been negotiated by the net/http server.
	return grpc.NewServer(grpcOptions...), nil
}
