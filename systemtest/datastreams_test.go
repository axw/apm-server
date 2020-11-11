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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/elastic/apm-server/systemtest"
	"github.com/elastic/apm-server/systemtest/apmservertest"
	"github.com/elastic/apm-server/systemtest/estest"
	"github.com/stretchr/testify/require"
)

func TestDataStreamsEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			systemtest.CleanupElasticsearch(t)
			srv := apmservertest.NewUnstartedServer(t)
			if enabled {
				// Create a data stream index template.
				resp, err := systemtest.Elasticsearch.Indices.PutIndexTemplate("apm-data-streams", strings.NewReader(fmt.Sprintf(`{
				  "index_patterns": ["traces-*", "logs-*", "metrics-*"],
				  "data_stream": {},
				  "priority": 200,
				  "template": {"settings": {"number_of_shards": 1}}
				}`)))
				require.NoError(t, err)
				body, _ := ioutil.ReadAll(resp.Body)
				require.False(t, resp.IsError(), string(body))

				// Create an API Key which can write to traces-* etc.
				// The default APM Server user can only write to apm-*.
				resp, err = systemtest.Elasticsearch.Security.CreateAPIKey(strings.NewReader(fmt.Sprintf(`{
				  "name": "%s",
				  "expiration": "1h",
				  "role_descriptors": {
				    "write-apm-data": {
				      "cluster": ["monitor"],
				      "index": [
				        {
				          "names": ["traces-*", "metrics-*", "logs-*"],
					  "privileges": ["write", "create_index"]
				        }
				      ]
				    }
				  }
				}`, t.Name())))
				require.NoError(t, err)

				var apiKeyResponse struct {
					ID     string
					Name   string
					APIKey string `json:"api_key"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&apiKeyResponse))

				srv.Config.DataStreams = &apmservertest.DataStreamsConfig{Enabled: true}

				// Use an API Key to mimic running under Fleet, with limited permissions.
				srv.Config.Output.Elasticsearch.Username = ""
				srv.Config.Output.Elasticsearch.Password = ""
				srv.Config.Output.Elasticsearch.APIKey = fmt.Sprintf("%s:%s", apiKeyResponse.ID, apiKeyResponse.APIKey)

				// A pipeline cannot be registered as the API Key does not grant that permission.
				// Since it cannot be registered, we disable the pipeline altogether.
				srv.Config.RegisterPipeline = &apmservertest.RegisterPipelineConfig{Enabled: false}
				srv.Config.Output.Elasticsearch.Pipeline = "_none"

				// Disable setup of templates and ILM in both apm-server and libbeat,
				// since the server does not have privileges to setup ILM policies or templates.
				srv.Config.Setup.TemplateEnabled = false
				srv.Config.Setup.ILMEnabled = false
				srv.Config.ILM = &apmservertest.ILMConfig{SetupEnabled: false}
			}
			require.NoError(t, srv.Start())

			tracer := srv.Tracer()
			tracer.StartTransaction("name", "type").End()
			tracer.Flush(nil)

			result := systemtest.Elasticsearch.ExpectDocs(t, "apm-*,traces-*", estest.TermQuery{
				Field: "processor.event", Value: "transaction",
			})
			systemtest.ApproveEvents(
				t, t.Name(), result.Hits.Hits,
				"@timestamp", "timestamp.us",
				"trace.id", "transaction.id",
			)
		})
	}
}
