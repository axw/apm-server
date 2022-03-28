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

package middleware

import (
	"net/http"

	"github.com/elastic/beats/v7/libbeat/monitoring"

	"github.com/elastic/apm-server/beater/request"
)

// MonitoringMiddleware returns a middleware that increases monitoring counters for collecting metrics
// about request processing. As input parameter it takes a map capable of mapping a request.ResultID to a counter.
func MonitoringMiddleware(m map[request.ResultID]*monitoring.Int) Middleware {
	return func(h request.Handler) (request.Handler, error) {
		add := func(id request.ResultID, n int64) {
			if counter, ok := m[id]; ok {
				counter.Add(n)
			}
		}

		return func(c *request.Context) {
			add(request.IDRequestCount, 1)
			add(request.IDRequestInflightCount, 1)
			defer add(request.IDRequestInflightCount, -1)

			h(c)

			add(request.IDResponseCount, 1)
			if c.Result.StatusCode >= http.StatusBadRequest {
				add(request.IDResponseErrorsCount, 1)
			} else {
				add(request.IDResponseValidCount, 1)
			}

			add(c.Result.ID, 1)
		}, nil

	}
}
