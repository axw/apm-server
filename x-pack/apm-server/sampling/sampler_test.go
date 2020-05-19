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
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/pubsub"
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

	storage := eventstorage.New(db, time.Minute)
	writer := storage.NewWriter()
	defer writer.Close()
	assert.NoError(t, writer.WriteTransaction(transaction1))
	assert.NoError(t, writer.WriteSpan(span1))
	assert.NoError(t, writer.WriteTransaction(transaction2))
	assert.NoError(t, writer.WriteSpan(span2))

	// Add traceID1 to the sampling reservoir.
	groups := newTraceGroups(100, 1.0)
	groups.sampleTrace(transaction1)

	var published []string
	var publisher sampledTraceIDsPublisherFunc = func(ctx context.Context, traceIDs ...string) error {
		published = append(published, traceIDs...)
		return nil
	}

	sampler := newTailSampler(
		groups,
		time.Millisecond,
		writer,
		writer,
		pubsub.NopSampledTraceIDsSubscriber{},
		publisher,
		reporter,
		1.0,
	)
	defer sampler.Close()
	select {
	case events := <-reported:
		assert.ElementsMatch(t, []transform.Transformable{transaction1, span1}, events)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sampled trace events to be reported")
	}
	assert.NoError(t, sampler.Close())

	sampled, err := writer.IsTraceSampled(traceID1)
	assert.NoError(t, err)
	assert.True(t, sampled)

	_, err = writer.IsTraceSampled(traceID2)
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

	storage := eventstorage.New(db, time.Minute)
	writer := storage.NewWriter()
	defer writer.Close()
	assert.NoError(t, writer.WriteTransaction(transaction1))
	assert.NoError(t, writer.WriteSpan(span1))
	assert.NoError(t, writer.WriteTransaction(transaction2))
	assert.NoError(t, writer.WriteSpan(span2))

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
		newTraceGroups(100, 1.0),
		time.Minute, // ~never
		writer,
		writer,
		subscriber,
		publisher,
		reporter,
		1.0,
	)
	defer sampler.Close()
	select {
	case events := <-reported:
		assert.ElementsMatch(t, []transform.Transformable{transaction1, span1}, events)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sampled trace events to be reported")
	}
	assert.NoError(t, sampler.Close())

	sampled, err := writer.IsTraceSampled(traceID1)
	assert.NoError(t, err)
	assert.True(t, sampled)

	_, err = writer.IsTraceSampled(traceID2)
	assert.Equal(t, eventstorage.ErrNotFound, err)

	assert.Empty(t, published) // remote decisions don't get republished
}

func TestTraceSamplerReservoirResize(t *testing.T) {
	sampleRate := 0.2
	groups := newTraceGroups(1, sampleRate)
	sendTransactions := func(n int) {
		for i := 0; i < n; i++ {
			groups.sampleTrace(&model.Transaction{
				TraceID: "0102030405060708090a0b0c0d0e0f10",
				ID:      "0102030405060708",
			})
		}
	}

	// All groups start out with a reservoir size of 1000.
	//
	// Seed a group with 10000 transactions. After the first
	// interval, we should end up with a reservoir size of
	// 10000*0.2=2000.
	sendTransactions(10000)

	var totalSampled int
	reported := make(chan int)
	reporter := func(ctx context.Context, req publish.PendingReq) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reported <- totalSampled:
			totalSampled = 0
		}
		sendTransactions(20000)
		return nil
	}

	ingestRateCoefficient := 0.75
	sampler := newTailSampler(
		groups,
		time.Microsecond,
		traceSampledWriterFunc(func(traceID string, sampled bool) error {
			totalSampled++
			return nil
		}),
		eventReaderFunc(func(traceID string, out *model.Batch) error {
			// Add an event so the reporter is called.
			out.Transactions = append(out.Transactions, &model.Transaction{})
			return nil
		}),
		pubsub.NopSampledTraceIDsSubscriber{},
		pubsub.NopSampledTraceIDsPublisher{},
		reporter,
		ingestRateCoefficient,
	)
	defer sampler.Close()

	// We send 10000 to start with, then 20000 each time after.
	// The number of sampled trace IDs will converge on 4000 (0.2*20000).
	//
	assert.Equal(t, 1000, <-reported) // initial reservoir size
	assert.Equal(t, 2000, <-reported) // 0.2 * 10000 (initial ingest rate)
	assert.Equal(t, 3500, <-reported) // 0.2 * (0.25*10000 + 0.75*20000)
	assert.Equal(t, 3875, <-reported) // 0.2 * (0.25*17500 + 0.75*20000)
	assert.Equal(t, 3969, <-reported) // etc.
	assert.Equal(t, 3992, <-reported)
	assert.Equal(t, 3998, <-reported)
	assert.Equal(t, 4000, <-reported)
	assert.Equal(t, 4000, <-reported)
}

func TestTraceSamplerReservoirResizeMinimum(t *testing.T) {
	sampleRate := 0.1
	groups := newTraceGroups(1, sampleRate)
	sendTransactions := func(n int) {
		for i := 0; i < n; i++ {
			groups.sampleTrace(&model.Transaction{
				TraceID: "0102030405060708090a0b0c0d0e0f10",
				ID:      "0102030405060708",
			})
		}
	}
	sendTransactions(10000)

	var totalSampled int
	reported := make(chan int)
	reporter := func(ctx context.Context, req publish.PendingReq) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reported <- totalSampled:
			totalSampled = 0
		}
		sendTransactions(1000)
		return nil
	}

	ingestRateCoefficient := 1.0
	sampler := newTailSampler(
		groups,
		time.Microsecond,
		traceSampledWriterFunc(func(traceID string, sampled bool) error {
			totalSampled++
			return nil
		}),
		eventReaderFunc(func(traceID string, out *model.Batch) error {
			// Add an event so the reporter is called.
			out.Transactions = append(out.Transactions, &model.Transaction{})
			return nil
		}),
		pubsub.NopSampledTraceIDsSubscriber{},
		pubsub.NopSampledTraceIDsPublisher{},
		reporter,
		ingestRateCoefficient,
	)
	defer sampler.Close()

	assert.Equal(t, 1000, <-reported)
	assert.Equal(t, 100, <-reported)
}

type eventReaderFunc func(traceID string, out *model.Batch) error

func (f eventReaderFunc) ReadEvents(traceID string, out *model.Batch) error {
	return f(traceID, out)
}

type traceSampledWriterFunc func(traceID string, sampled bool) error

func (f traceSampledWriterFunc) WriteTraceSampled(traceID string, sampled bool) error {
	return f(traceID, sampled)
}

type sampledTraceIDsSubscriberFunc func(ctx context.Context, _ chan<- string) error

func (f sampledTraceIDsSubscriberFunc) SubscribeSampledTraceIDs(ctx context.Context, traceIDs chan<- string) error {
	return f(ctx, traceIDs)
}

type sampledTraceIDsPublisherFunc func(ctx context.Context, traceIDs ...string) error

func (f sampledTraceIDsPublisherFunc) PublishSampledTraceIDs(ctx context.Context, traceIDs ...string) error {
	return f(ctx, traceIDs...)
}
