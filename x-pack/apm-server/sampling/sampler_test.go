// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceSampler(t *testing.T) {
	traceID1 := "0102030405060708090a0b0c0d0e0f10"
	traceID2 := "0102030405060708090a0b0c0d0e0f11"
	transaction1 := &model.Transaction{
		TraceID: traceID1,
		ID:      "0102030405060708",
	}
	span1 := &model.Span{
		TraceID: traceID1,
		ID:      "0102030405060709",
	}
	transaction2 := &model.Transaction{
		TraceID: traceID2,
		ID:      "0102030405060710",
	}
	span2 := &model.Span{
		TraceID: traceID2,
		ID:      "0102030405060711",
	}

	tempdir, err := ioutil.TempDir("", "samplingtest")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	badgerOpts := badger.DefaultOptions(tempdir)
	badgerOpts.Logger = nil
	db, err := badger.Open(badgerOpts)
	require.NoError(t, err)
	defer db.Close()

	reported := make(chan []transform.Transformable)
	reporter := func(ctx context.Context, req publish.PendingReq) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reported <- req.Transformables:
			return nil
		}
	}

	storage := eventstorage.New(db, eventstorage.JSONCodec{}, time.Minute)
	readWriter := storage.NewReadWriter()
	defer readWriter.Close()

	assert.NoError(t, readWriter.WriteTransaction(transaction1))
	assert.NoError(t, readWriter.WriteSpan(span1))
	assert.NoError(t, readWriter.WriteTransaction(transaction2))
	assert.NoError(t, readWriter.WriteSpan(span2))

	// Add traceID1 to the sampling reservoir.
	groups := newTraceGroups(100, 1.0, 1.0)
	groups.sampleTrace(transaction1)

	var published []string
	var publisher sampledTraceIDsPublisherFunc = func(ctx context.Context, traceIDs ...string) error {
		published = append(published, traceIDs...)
		return nil
	}

	sampler := newTailSampler(
		groups,
		time.Millisecond,
		readWriter,
		readWriter,
		nopSampledTraceIDsSubscriber,
		publisher,
		reporter,
	)
	go sampler.Run()
	defer sampler.Stop(context.Background())

	select {
	case events := <-reported:
		assert.ElementsMatch(t, []transform.Transformable{transaction1, span1}, events)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sampled trace events to be reported")
	}
	assert.NoError(t, sampler.Stop(context.Background()))

	sampled, err := readWriter.IsTraceSampled(traceID1)
	assert.NoError(t, err)
	assert.True(t, sampled)

	_, err = readWriter.IsTraceSampled(traceID2)
	assert.Equal(t, eventstorage.ErrNotFound, err)

	assert.Equal(t, []string{traceID1}, published)
}

func TestTraceSamplerSubscriber(t *testing.T) {
	traceID1 := "0102030405060708090a0b0c0d0e0f10"
	traceID2 := "0102030405060708090a0b0c0d0e0f11"
	transaction1 := &model.Transaction{
		TraceID: traceID1,
		ID:      "0102030405060708",
	}
	span1 := &model.Span{
		TraceID: traceID1,
		ID:      "0102030405060709",
	}
	transaction2 := &model.Transaction{
		TraceID: traceID2,
		ID:      "0102030405060710",
	}
	span2 := &model.Span{
		TraceID: traceID2,
		ID:      "0102030405060711",
	}

	tempdir, err := ioutil.TempDir("", "samplingtest")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)
	badgerOpts := badger.DefaultOptions(tempdir)
	badgerOpts.Logger = nil
	db, err := badger.Open(badgerOpts)
	require.NoError(t, err)
	defer db.Close()

	reported := make(chan []transform.Transformable)
	reporter := func(ctx context.Context, req publish.PendingReq) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reported <- req.Transformables:
			return nil
		}
	}

	storage := eventstorage.New(db, eventstorage.JSONCodec{}, time.Minute)
	readWriter := storage.NewReadWriter()
	defer readWriter.Close()
	assert.NoError(t, readWriter.WriteTransaction(transaction1))
	assert.NoError(t, readWriter.WriteSpan(span1))
	assert.NoError(t, readWriter.WriteTransaction(transaction2))
	assert.NoError(t, readWriter.WriteSpan(span2))

	// Simulate scenario where an external entity samples traceID1 (only).
	var subscriber sampledTraceIDsSubscriberFunc = func(ctx context.Context, traceIDs chan<- string) error {
		traceIDs <- traceID1
		return nil
	}

	var published []string
	var publisher sampledTraceIDsPublisherFunc = func(ctx context.Context, traceIDs ...string) error {
		published = append(published, traceIDs...)
		return nil
	}

	sampler := newTailSampler(
		newTraceGroups(100, 1.0, 1.0),
		time.Minute, // ~never
		readWriter,
		readWriter,
		subscriber,
		publisher,
		reporter,
	)
	go sampler.Run()
	defer sampler.Stop(context.Background())

	select {
	case events := <-reported:
		assert.ElementsMatch(t, []transform.Transformable{transaction1, span1}, events)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sampled trace events to be reported")
	}
	assert.NoError(t, sampler.Stop(context.Background()))

	sampled, err := readWriter.IsTraceSampled(traceID1)
	assert.NoError(t, err)
	assert.True(t, sampled)

	_, err = readWriter.IsTraceSampled(traceID2)
	assert.Equal(t, eventstorage.ErrNotFound, err)

	assert.Empty(t, published) // remote decisions don't get republished
}

type eventReaderFunc func(traceID string, out *model.Batch) error

func (f eventReaderFunc) ReadEvents(traceID string, out *model.Batch) error {
	return f(traceID, out)
}

type traceSampledWriterFunc func(traceID string, sampled bool) error

func (f traceSampledWriterFunc) WriteTraceSampled(traceID string, sampled bool) error {
	return f(traceID, sampled)
}

var nopSampledTraceIDsSubscriber sampledTraceIDsSubscriberFunc = func(ctx context.Context, _ chan<- string) error {
	<-ctx.Done()
	return ctx.Err()
}

var nopSampledTraceIDsPublisher sampledTraceIDsPublisherFunc = func(ctx context.Context, traceIDs ...string) error {
	return nil
}

type sampledTraceIDsSubscriberFunc func(ctx context.Context, _ chan<- string) error

func (f sampledTraceIDsSubscriberFunc) SubscribeSampledTraceIDs(ctx context.Context, traceIDs chan<- string) error {
	return f(ctx, traceIDs)
}

type sampledTraceIDsPublisherFunc func(ctx context.Context, traceIDs ...string) error

func (f sampledTraceIDsPublisherFunc) PublishSampledTraceIDs(ctx context.Context, traceIDs ...string) error {
	return f(ctx, traceIDs...)
}
