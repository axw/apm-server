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

// NopClient returns a new elasticsearch.Client that responds to publish and
// subcsribe requests such that they are effectively no-ops.
func NopClient() *elasticsearch.Client {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"testing.invalid"},
		Transport: nopClientRoundTripper{},
	})
	if err != nil {
		panic(err)
	}
	return client
}

type nopClientRoundTripper struct{}

func (nopClientRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
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
