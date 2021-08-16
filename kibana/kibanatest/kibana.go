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

package kibanatest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/apm-server/beater/config"
	"github.com/elastic/apm-server/kibana"

	"github.com/elastic/beats/v7/libbeat/common"
	libbeatkibana "github.com/elastic/beats/v7/libbeat/kibana"
)

// MockKibana provides a kibana.Client which responds to Send requests with
// the given response code and body, and reports the given version.
func MockKibana(t testing.TB, respCode int, respBody map[string]interface{}, v common.Version) kibana.Client {
	encodedBody, err := json.Marshal(respBody)
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"version":{"number":%q}}`, v.String())))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(respCode)
		w.Write(encodedBody)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return kibana.NewConnectingClient(&config.KibanaConfig{
		Enabled:      true,
		ClientConfig: libbeatkibana.ClientConfig{Host: srv.URL},
	})
}
