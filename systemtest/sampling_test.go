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

package systemtest_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm"

	"github.com/elastic/apm-server/systemtest"
	"github.com/elastic/apm-server/systemtest/apmservertest"
	"github.com/elastic/apm-server/systemtest/estest"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

func TestKeepUnsampled(t *testing.T) {
	for _, keepUnsampled := range []bool{false, true} {
		t.Run(fmt.Sprint(keepUnsampled), func(t *testing.T) {
			systemtest.CleanupElasticsearch(t)
			srv := apmservertest.NewUnstartedServer(t)
			srv.Config.Sampling = &apmservertest.SamplingConfig{
				KeepUnsampled: keepUnsampled,
			}
			err := srv.Start()
			require.NoError(t, err)

			// Send one unsampled transaction, and one sampled transaction.
			transactionType := "TestKeepUnsampled"
			tracer := srv.Tracer()
			tracer.StartTransaction("sampled", transactionType).End()
			tracer.SetSampler(apm.NewRatioSampler(0))
			tracer.StartTransaction("unsampled", transactionType).End()
			tracer.Flush(nil)

			result := systemtest.Elasticsearch.ExpectDocs(t, "apm-*", estest.TermQuery{
				Field: "transaction.type",
				Value: transactionType,
			})

			expectedTransactionDocs := 1
			if keepUnsampled {
				expectedTransactionDocs++
			}
			assert.Len(t, result.Hits.Hits, expectedTransactionDocs)
		})
	}
}

func TestTailSampling(t *testing.T) {
	systemtest.CleanupElasticsearch(t)
	srv := apmservertest.NewUnstartedServer(t)
	srv.Config.Sampling = &apmservertest.SamplingConfig{
		KeepUnsampled: false,
		Tail: &apmservertest.TailSamplingConfig{
			Enabled:           true,
			DefaultSampleRate: 0.5,
			Interval:          time.Second,
		},
	}
	err := srv.Start()
	require.NoError(t, err)

	const total = 200
	const expected = 100 // 50%

	tracer := srv.Tracer()
	for i := 0; i < total; i++ {
		tx := tracer.StartTransaction("GET /", "tail_sampling")
		tx.Duration = time.Second * time.Duration(i+1)
		tx.End()
	}
	tracer.Flush(nil)

	var result estest.SearchResult
	_, err = systemtest.Elasticsearch.Search("apm-*").WithQuery(estest.BoolQuery{
		Filter: []interface{}{
			estest.TermQuery{
				Field: "transaction.type",
				Value: "tail_sampling",
			},
		},
	}).WithSize(total).Do(context.Background(), &result,
		estest.WithCondition(func(*esapi.Response) bool {
			return len(result.Hits.Hits) >= expected
		}),
	)
	require.NoError(t, err)
	assert.Len(t, result.Hits.Hits, expected)
}
