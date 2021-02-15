package esoutput

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

type client struct {
	client           *elasticsearch.Client
	indexSelector    outputs.IndexSelector
	pipelineSelector *outil.Selector
	bulk             esutil.BulkIndexer
}

func newClient(
	es *elasticsearch.Client,
	indexSelector outputs.IndexSelector,
	pipelineSelector *outil.Selector,
) (*client, error) {
	bulk, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        es,
		FlushInterval: 10 * time.Second,
		OnError: func(ctx context.Context, err error) {
			fmt.Println("OnError:", err)
		},
	})
	if err != nil {
		return nil, err
	}
	c := &client{
		client:           es,
		indexSelector:    indexSelector,
		pipelineSelector: pipelineSelector,
		bulk:             bulk,
	}
	return c, nil
}

func (c *client) Connect() error {
	resp, err := c.client.Security.Authenticate()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Authenticate returned error (%s): %s", resp.Status(), body)
	}
	return nil
}

func (c *client) Close() error {
	return c.bulk.Close(context.Background())
}

func (c *client) String() string {
	// TODO(axw) include elasticsearch URL(s)
	return "go-elasticsearch"
}

func (c *client) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	for _, event := range events {
		if err := c.publishEvent(ctx, event); err != nil {
			// TODO(axw) call appropriate batch method
			return err
		}
	}
	batch.ACK()
	return nil
}

func (c *client) publishEvent(ctx context.Context, event publisher.Event) error {
	opType := events.GetOpType(event.Content)
	if opType == events.OpTypeDefault {
		opType = events.OpTypeIndex
	}
	index, err := c.indexSelector.Select(&event.Content)
	if err != nil {
		return err
	}
	// TODO(axw) pipeline
	var buf bytes.Buffer
	if err := encodeEvent(&buf, event.Content.Timestamp, event.Content.Fields); err != nil {
		return err
	}
	return c.bulk.Add(ctx, esutil.BulkIndexerItem{
		Index:  index,
		Action: opType.String(),
		Body:   &buf,
	})
}

func encodeEvent(buf *bytes.Buffer, timestamp time.Time, fields common.MapStr) error {
	// TODO(axw) use fastjson instead?
	visitor := json.NewVisitor(buf)
	visitor.SetEscapeHTML(false)
	folder, err := gotype.NewIterator(visitor, gotype.Folders(codec.MakeTimestampEncoder(), codec.MakeBCTimestampEncoder()))
	if err != nil {
		return err
	}
	type event struct {
		Timestamp time.Time     `struct:"@timestamp"`
		Fields    common.MapStr `struct:",inline"`
	}
	return folder.Fold(event{Timestamp: timestamp, Fields: fields})
}
