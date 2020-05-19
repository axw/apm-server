// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

func TestProcessUnsampled(t *testing.T) {
	processor, err := sampling.NewProcessor(newTempdirConfig(t))
	require.NoError(t, err)
	defer processor.Stop(context.Background())

	transaction := &model.Transaction{
		TraceID: "0102030405060708090a0b0c0d0e0f10",
		ID:      "0102030405060708",
		Sampled: newBool(false),
	}
	in := []transform.Transformable{transaction}
	out, err := processor.ProcessTransformables(context.Background(), in)
	require.NoError(t, err)

	// Unsampled transaction should be reported immediately.
	assert.Equal(t, in, out)
}

func TestProcessTailSampled(t *testing.T) {
	config := newTempdirConfig(t)

	// Seed event storage with a tail-sampling decisions, to show that
	// subsequent events in the trace will be reported immediately.
	traceID1 := "0102030405060708090a0b0c0d0e0f10"
	traceID2 := "0102030405060708090a0b0c0d0e0f11"
	withBadger(t, config.StorageDir, func(db *badger.DB) {
		storage := eventstorage.New(db, time.Minute)
		writer := storage.NewWriter()
		defer writer.Close()
		assert.NoError(t, writer.WriteTraceSampled(traceID1, true))
		assert.NoError(t, writer.Flush())

		storage = eventstorage.New(db, -1) // expire immediately
		writer = storage.NewWriter()
		defer writer.Close()
		assert.NoError(t, writer.WriteTraceSampled(traceID2, true))
		assert.NoError(t, writer.Flush())
	})

	processor, err := sampling.NewProcessor(config)
	require.NoError(t, err)
	defer processor.Stop(context.Background())

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

	in := []transform.Transformable{transaction1, transaction2, span1, span2}
	out, err := processor.ProcessTransformables(context.Background(), in)
	require.NoError(t, err)

	// Tail sampling decision already made. The first transaction and span should be
	// reported immediately, whereas the second ones should be written storage since
	// they were received after the trace sampling entry expired.
	assert.Equal(t, []transform.Transformable{transaction1, span1}, out)

	// Stop the processor so we can access the database.
	assert.NoError(t, processor.Stop(context.Background()))
	withBadger(t, config.StorageDir, func(db *badger.DB) {
		storage := eventstorage.New(db, time.Minute)
		writer := storage.NewWriter()
		defer writer.Close()

		var batch model.Batch
		err := writer.ReadEvents(traceID1, &batch)
		assert.NoError(t, err)
		assert.Zero(t, batch)

		err = writer.ReadEvents(traceID2, &batch)
		assert.NoError(t, err)
		assert.Equal(t, model.Batch{
			Spans:        []*model.Span{span2},
			Transactions: []*model.Transaction{transaction2},
		}, batch)
	})
}

func withBadger(tb testing.TB, storageDir string, f func(db *badger.DB)) {
	badgerOpts := badger.DefaultOptions(storageDir)
	badgerOpts.Logger = nil
	db, err := badger.Open(badgerOpts)
	require.NoError(tb, err)
	f(db)
	assert.NoError(tb, db.Close())
}

func newTempdirConfig(tb testing.TB) sampling.Config {
	tempdir, err := ioutil.TempDir("", "samplingtest")
	require.NoError(tb, err)
	tb.Cleanup(func() { os.RemoveAll(tempdir) })
	return sampling.Config{
		Reporter:              func(ctx context.Context, req publish.PendingReq) error { return nil },
		StorageDir:            tempdir,
		StorageGCInterval:     time.Second,
		TTL:                   30 * time.Minute,
		MaxTraceGroups:        1000,
		FlushInterval:         time.Second,
		DefaultSampleRate:     0.1,
		IngestRateCoefficient: 0.9,
	}
}

func newBool(v bool) *bool {
	return &v
}
