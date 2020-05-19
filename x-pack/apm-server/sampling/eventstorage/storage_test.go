// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eventstorage_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

func TestWriteEvents(t *testing.T) {
	db := newBadgerDB(t, badgerOptions)
	ttl := time.Minute
	store := eventstorage.New(db, ttl)
	writer := store.NewWriter()
	defer writer.Close()

	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	transactionID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	spanID := []byte{8, 7, 6, 5, 4, 3, 2, 1}
	transaction := &model.Transaction{
		TraceID: string(traceID),
		ID:      string(transactionID),
	}
	span := &model.Span{
		TraceID: string(traceID),
		ID:      string(spanID),
	}

	before := time.Now()
	assert.NoError(t, writer.WriteTransaction(transaction))
	assert.NoError(t, writer.WriteSpan(span))
	assert.NoError(t, writer.Flush())

	var recorded []interface{}
	assert.NoError(t, db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{Prefix: traceID})
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			expiresAt := item.ExpiresAt()
			expiryTime := time.Unix(int64(expiresAt), 0)
			assert.Condition(t, func() bool {
				return !before.After(expiryTime) && !expiryTime.After(before.Add(ttl))
			})

			var value interface{}
			switch meta := item.UserMeta(); meta {
			case 't':
				value = &model.Transaction{}
			case 's':
				value = &model.Span{}
			default:
				t.Fatalf("invalid meta %q", meta)
			}
			assert.NoError(t, item.Value(func(data []byte) error {
				return json.Unmarshal(data, value)
			}))
			recorded = append(recorded, value)
		}
		return nil
	}))
	assert.Equal(t, []interface{}{transaction, span}, recorded)
}

func TestWriteTraceSampled(t *testing.T) {
	db := newBadgerDB(t, badgerOptions)
	ttl := time.Minute
	store := eventstorage.New(db, ttl)
	writer := store.NewWriter()
	defer writer.Close()

	before := time.Now()
	assert.NoError(t, writer.WriteTraceSampled("sampled_trace_id", true))
	assert.NoError(t, writer.WriteTraceSampled("unsampled_trace_id", false))
	assert.NoError(t, writer.Flush())

	sampled := make(map[string]bool)
	assert.NoError(t, db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{})
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			expiresAt := item.ExpiresAt()
			expiryTime := time.Unix(int64(expiresAt), 0)
			assert.Condition(t, func() bool {
				return !before.After(expiryTime) && !expiryTime.After(before.Add(ttl))
			})

			key := string(item.Key())
			switch meta := item.UserMeta(); meta {
			case 'S':
				sampled[key] = true
			case 'U':
				sampled[key] = false
			default:
				t.Fatalf("invalid meta %q", meta)
			}
			assert.Zero(t, item.ValueSize())
		}
		return nil
	}))
	assert.Equal(t, map[string]bool{
		"sampled_trace_id":   true,
		"unsampled_trace_id": false,
	}, sampled)
}

func TestReadEvents(t *testing.T) {
	db := newBadgerDB(t, badgerOptions)
	ttl := time.Minute
	store := eventstorage.New(db, ttl)

	traceID := [...]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	require.NoError(t, db.Update(func(txn *badger.Txn) error {
		key := append(traceID[:], ":12345678"...)
		value := []byte(`{"name":"transaction"}`)
		if err := txn.SetEntry(badger.NewEntry(key, value).WithMeta('t')); err != nil {
			return err
		}

		key = append(traceID[:], ":87654321"...)
		value = []byte(`{"name":"span"}`)
		if err := txn.SetEntry(badger.NewEntry(key, value).WithMeta('s')); err != nil {
			return err
		}

		// Write an entry with just the trace ID as the key. Because
		// there's no proceeding ":" (and transaction or span ID),
		// it will be ignored.
		key = traceID[:]
		value = []byte(`not-json`)
		if err := txn.SetEntry(badger.NewEntry(key, value).WithMeta('s')); err != nil {
			return err
		}

		// Write an entry with an unknown meta value. It will be ignored.
		key = append(traceID[:], "11111111"...)
		value = []byte(`not-json`)
		if err := txn.SetEntry(badger.NewEntry(key, value).WithMeta('?')); err != nil {
			return err
		}
		return nil
	}))

	var events model.Batch
	w := store.NewWriter()
	assert.NoError(t, w.ReadEvents(string(traceID[:]), &events))
	assert.Equal(t, []*model.Transaction{{Name: "transaction"}}, events.Transactions)
	assert.Equal(t, []*model.Span{{Name: "span"}}, events.Spans)
}

func TestIsTraceSampled(t *testing.T) {
	db := newBadgerDB(t, badgerOptions)
	ttl := time.Minute
	store := eventstorage.New(db, ttl)

	require.NoError(t, db.Update(func(txn *badger.Txn) error {
		if err := txn.SetEntry(badger.NewEntry([]byte("sampled_trace_id"), nil).WithMeta('S')); err != nil {
			return err
		}
		if err := txn.SetEntry(badger.NewEntry([]byte("unsampled_trace_id"), nil).WithMeta('U')); err != nil {
			return err
		}
		return nil
	}))

	w := store.NewWriter()

	sampled, err := w.IsTraceSampled("sampled_trace_id")
	assert.NoError(t, err)
	assert.True(t, sampled)

	sampled, err = w.IsTraceSampled("unsampled_trace_id")
	assert.NoError(t, err)
	assert.False(t, sampled)

	_, err = w.IsTraceSampled("unknown_trace_id")
	assert.Equal(t, err, eventstorage.ErrNotFound)
}

func badgerOptions() badger.Options {
	return badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
}

type badgerOptionsFunc func() badger.Options

func newBadgerDB(tb testing.TB, badgerOptions badgerOptionsFunc) *badger.DB {
	db, err := badger.Open(badgerOptions())
	if err != nil {
		panic(err)
	}
	tb.Cleanup(func() { db.Close() })
	return db
}
