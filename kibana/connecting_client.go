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

package kibana

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/elastic/apm-server/beater/config"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/kibana"
	"github.com/elastic/beats/v7/libbeat/logp"

	logs "github.com/elastic/apm-server/log"
)

const (
	initBackoff = time.Second
	maxBackoff  = 30 * time.Second
)

// Client provides an interface for Kibana Clients
type Client interface {
	// Send tries to send request to Kibana and returns unparsed response
	Send(context.Context, string, string, url.Values, http.Header, io.Reader) (*http.Response, error)
	// GetVersion returns Kibana version or an error
	GetVersion(context.Context) (common.Version, error)
	// SupportsVersion compares given version to version of connected Kibana instance
	SupportsVersion(context.Context, *common.Version) (bool, error)
}

// ConnectingClient implements Client interface
type ConnectingClient struct {
	cfg *config.KibanaConfig

	mu      sync.Mutex
	waiters []chan *kibana.Client
	client  *kibana.Client
}

// NewConnectingClient returns instance of ConnectingClient and starts a background routine trying to connect
// to configured Kibana instance, using JitterBackoff for establishing connection.
func NewConnectingClient(cfg *config.KibanaConfig) Client {
	c := &ConnectingClient{cfg: cfg}
	go func() {
		log := logp.NewLogger(logs.Kibana)
		done := make(chan struct{})
		jitterBackoff := backoff.NewEqualJitterBackoff(done, initBackoff, maxBackoff)
		for c.client == nil {
			log.Debug("Trying to obtain connection to Kibana.")
			err := c.connect()
			if err != nil {
				log.Errorf("failed to obtain connection to Kibana: %s", err.Error())
			}
			backoff.WaitOnError(jitterBackoff, err)
		}
		log.Info("Successfully obtained connection to Kibana.")
	}()
	return c
}

// Send tries to send a request to Kibana via established connection and returns unparsed response.
func (c *ConnectingClient) Send(
	ctx context.Context,
	method, extraPath string,
	params url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	client, err := c.waitClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.SendWithContext(ctx, method, extraPath, params, headers, body)
}

// GetVersion returns Kibana version or an error.
func (c *ConnectingClient) GetVersion(ctx context.Context) (common.Version, error) {
	span, _ := apm.StartSpan(ctx, "GetVersion", "app")
	defer span.End()
	client, err := c.waitClient(ctx)
	if err != nil {
		return common.Version{}, err
	}
	return client.GetVersion(), nil
}

// SupportsVersion checks if connected Kibana instance is compatible to given version.
func (c *ConnectingClient) SupportsVersion(ctx context.Context, v *common.Version) (bool, error) {
	span, ctx := apm.StartSpan(ctx, "SupportsVersion", "app")
	defer span.End()
	log := logp.NewLogger(logs.Kibana)

	for i := 0; i < 2; i++ {
		client, err := c.waitClient(ctx)
		if err != nil {
			return false, err
		}
		if v.LessThanOrEqual(false, &client.Version) {
			return true, nil
		}
		// Reconnect in case Kibana has been upgraded since we last connected and cached its version.
		if err := c.connect(); err != nil {
			log.Errorf("failed to obtain connection to Kibana: %s", err.Error())
			return false, err
		}
	}
	return false, nil
}

func (c *ConnectingClient) waitClient(ctx context.Context) (*kibana.Client, error) {
	c.mu.Lock()
	if c.client != nil {
		c.mu.Unlock()
		return c.client, nil
	}
	ch := make(chan *kibana.Client, 1)
	c.waiters = append(c.waiters, ch)
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case client := <-ch:
		return client, nil
	}
}

func (c *ConnectingClient) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = nil

	clientConfig := kibana.DefaultClientConfig()
	if c.cfg != nil {
		clientConfig = c.cfg.ClientConfig
		if c.cfg.APIKey != "" {
			headers := make(map[string]string, len(clientConfig.Headers))
			for k, v := range clientConfig.Headers {
				headers[k] = v
			}
			headers["Authorization"] = "ApiKey " + base64.StdEncoding.EncodeToString([]byte(c.cfg.APIKey))
			clientConfig.Headers = headers
			clientConfig.Username = ""
			clientConfig.Password = ""
		}
	}

	client, err := kibana.NewClientWithConfig(&clientConfig)
	if err != nil {
		return err
	}
	client.HTTP = apmhttp.WrapClient(client.HTTP)

	c.client = client
	for _, waiter := range c.waiters {
		waiter <- client
	}
	c.waiters = c.waiters[:0]
	return nil
}
