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

package kibana_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/beater/config"
	"github.com/elastic/apm-server/kibana"
	"github.com/elastic/apm-server/kibana/kibanatest"

	"github.com/elastic/beats/v7/libbeat/common"
	libbeatkibana "github.com/elastic/beats/v7/libbeat/kibana"
)

func TestNewConnectingClientWithAPIKey(t *testing.T) {
	requests := make(chan *http.Request)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case requests <- r:
		}
	}))
	defer srv.Close()

	// Creating a client will cause a request to be sent to query the Kibana version.
	kibana.NewConnectingClient(&config.KibanaConfig{
		Enabled: true,
		APIKey:  "foo-id:bar-apikey",
		ClientConfig: libbeatkibana.ClientConfig{
			Host:     srv.URL,
			Username: "elastic",
			Password: "secret",
		},
	})

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for request")
	case req := <-requests:
		assert.Nil(t, req.URL.User) // no username/password
		assert.Equal(t, "ApiKey Zm9vLWlkOmJhci1hcGlrZXk=", req.Header.Get("Authorization"))
	}
}

func TestConnectingClient_Send(t *testing.T) {
	t.Run("Send", func(t *testing.T) {
		c := newConnectingClient(t)
		r, err := c.Send(context.Background(), http.MethodGet, "", nil, nil, nil)
		require.NoError(t, err)
		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"response":"ok"}`, string(body))
		assert.Equal(t, http.StatusTeapot, r.StatusCode)
	})

	t.Run("SendContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		c := kibana.NewConnectingClient(mockCfg)
		r, err := c.Send(ctx, http.MethodGet, "", nil, nil, nil)
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Nil(t, r)
	})
}

func TestConnectingClient_GetVersion(t *testing.T) {
	t.Run("GetVersion", func(t *testing.T) {
		c := newConnectingClient(t)
		v, err := c.GetVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, common.MustNewVersion("7.3.0"), &v)
	})

	t.Run("GetVersionContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		c := kibana.NewConnectingClient(mockCfg)
		v, err := c.GetVersion(ctx)
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Equal(t, common.Version{}, v)
	})
}

func TestConnectingClient_SupportsVersion(t *testing.T) {
	t.Run("SupportsVersionTrue", func(t *testing.T) {
		c := newConnectingClient(t)
		s, err := c.SupportsVersion(context.Background(), common.MustNewVersion("7.3.0"))
		require.NoError(t, err)
		assert.True(t, s)
	})
	t.Run("SupportsVersionFalse", func(t *testing.T) {
		c := newConnectingClient(t)
		s, err := c.SupportsVersion(context.Background(), common.MustNewVersion("7.4.0"))
		require.NoError(t, err)
		assert.False(t, s)
	})

	t.Run("SupportsVersionContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		c := kibana.NewConnectingClient(mockCfg)
		s, err := c.SupportsVersion(ctx, common.MustNewVersion("7.3.0"))
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.False(t, s)
	})
}

var (
	mockCfg = &config.KibanaConfig{
		Enabled: true,
		ClientConfig: libbeatkibana.ClientConfig{
			Host: "non-existing",
		},
	}
)

func newConnectingClient(t testing.TB) kibana.Client {
	version := common.MustNewVersion("7.3.0")
	return kibanatest.MockKibana(t, http.StatusTeapot, map[string]interface{}{"response": "ok"}, *version)
}
