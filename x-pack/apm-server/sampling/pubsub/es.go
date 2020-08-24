// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pubsub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.elastic.co/fastjson"
)

const sampledTraceIDsIndex = "apm-sampled-traces"

// Elasticsearch provides a means of publishing and subscribing to sampled trace IDs.
type Elasticsearch struct {
	client         *elasticsearch.Client
	indexer        esutil.BulkIndexer
	searchInterval time.Duration
}

// NewElasticsearch returns a new object which can publish and subscribe to sampled
// trace IDs, using Elasticsearch for storage.
func NewElasticsearch(
	client *elasticsearch.Client,
	flushInterval time.Duration,
	searchInterval time.Duration,
) (*Elasticsearch, error) {
	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        client,
		Index:         sampledTraceIDsIndex,
		FlushInterval: flushInterval,
	})
	if err != nil {
		return nil, err
	}
	return &Elasticsearch{
		client:         client,
		indexer:        indexer,
		searchInterval: searchInterval,
	}, nil
}

// PublishSampledTraceIDs bulk indexes traceIDs into Elasticsearch.
func (es *Elasticsearch) PublishSampledTraceIDs(ctx context.Context, traceID ...string) error {
	for _, id := range traceID {
		// TODO(axw) record this server's ID.
		var json fastjson.Writer
		json.RawString(`{"trace.id":`)
		json.String(id)
		json.RawString(`}`)
		if err := es.indexer.Add(ctx, esutil.BulkIndexerItem{
			Index:  sampledTraceIDsIndex,
			Action: "index",
			Body:   bytes.NewReader(json.Bytes()),
		}); err != nil {
			return err
		}
	}
	return nil
}

// SubscribeSampledTraceIDs subscribes to new sampled trace IDs, sending them to the
// traceIDs channel.
//
// TODO(axw) ignore trace IDs published by this server.
func (es *Elasticsearch) SubscribeSampledTraceIDs(ctx context.Context, traceIDs chan<- string) error {
	const interval = time.Minute
	ticker := time.NewTicker(es.searchInterval)
	defer ticker.Stop()

	var lastSeqNo, lastPrimaryTerm int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		for {
			// Keep searching until there are no more new trace IDs.
			n, err := es.searchTraceIDs(ctx, traceIDs, &lastSeqNo, &lastPrimaryTerm)
			if err != nil {
				return err
			}
			if n == 0 {
				break
			}
		}
	}
}

// searchTraceIDs searches for new sampled trace IDs (after lastPrimaryTerm and lastSeqNo),
// sending them to the out channel and returning the number of trace IDs sent.
func (es *Elasticsearch) searchTraceIDs(ctx context.Context, out chan<- string, lastSeqNo, lastPrimaryTerm *int64) (int, error) {
	searchBody := map[string]interface{}{
		"size":                1000,
		"seq_no_primary_term": true,
		"sort":                []interface{}{map[string]interface{}{"_seq_no": "asc"}},
		"search_after":        []interface{}{lastSeqNo},
	}

	req := esapi.SearchRequest{
		Index: []string{sampledTraceIDsIndex},
		Body:  esutil.NewJSONReader(searchBody),
	}
	resp, err := req.Do(ctx, es.client)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		if resp.StatusCode == http.StatusNotFound {
			return 0, nil
		}
		message, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("search request failed: %s", message)
	}

	var result struct {
		Hits struct {
			Hits []struct {
				SeqNo       int64 `json:"_seq_no,omitempty"`
				PrimaryTerm int64 `json:"_primary_term,omitempty"`
				Source      struct {
					Trace struct {
						ID string `json:"id"`
					} `json:"trace"`
				} `json:"_source"`
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.Hits.Hits) == 0 {
		return 0, nil
	}

	var n int
	maxPrimaryTerm := *lastPrimaryTerm
	for _, hit := range result.Hits.Hits {
		if hit.SeqNo < *lastSeqNo || (hit.SeqNo == *lastSeqNo && hit.PrimaryTerm <= *lastPrimaryTerm) {
			continue
		}
		select {
		case <-ctx.Done():
			return n, ctx.Err()
		case out <- hit.Source.Trace.ID:
			n++
		}
		if hit.PrimaryTerm > maxPrimaryTerm {
			maxPrimaryTerm = hit.PrimaryTerm
		}
	}
	// we sort by hit.SeqNo, but not _primary_term (you can't?)
	*lastSeqNo = result.Hits.Hits[len(result.Hits.Hits)-1].SeqNo
	*lastPrimaryTerm = maxPrimaryTerm
	return n, nil
}
