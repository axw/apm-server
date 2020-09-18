// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pubsubtest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
)

// ChannelClient returns a new elasticsearch.Client that responds to publish and
// subcsribe requests by sending IDs to channel pub, and receiving from channel
// sub. If either channel is nil, then the respective operation will be a no-op.
func Client(pub chan<- string, sub <-chan string) *elasticsearch.Client {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"testing.invalid"},
		Transport: &channelClientRoundTripper{pub, sub},
	})
	if err != nil {
		panic(err)
	}
	return client
}

type channelClientRoundTripper struct {
	pub chan<- string
	sub <-chan string
}

func (c *channelClientRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	switch r.Method {
	case "GET":
		// Subscribe
		recorder.WriteString(`{"hits":{"hits":[]}}`)
	case "POST":
		// Publish
		var results []map[string]esutil.BulkIndexerResponseItem
		dec := json.NewDecoder(r.Body)
		for {
			var m map[string]interface{}
			if err := dec.Decode(&m); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			var action string
			for action = range m {
			}
			if err := dec.Decode(&m); err != nil {
				return nil, err
			}
			result := esutil.BulkIndexerResponseItem{Status: 200}
			results = append(results, map[string]esutil.BulkIndexerResponseItem{action: result})
		}
		if err := json.NewEncoder(recorder).Encode(results); err != nil {
			return nil, err
		}
	}
	return recorder.Result(), nil
}
