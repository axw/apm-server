// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eventstorage

import (
	stdjson "encoding/json"
	"errors"
	"time"

	"github.com/dgraph-io/badger/v2"
	jsoniter "github.com/json-iterator/go"

	"github.com/elastic/apm-server/model"
)

const (
	// NOTE(axw) these values (and their meanings) must remain stable
	// over time, to avoid misinterpreting historical data.
	entryMetaTraceSampled   = 'S'
	entryMetaTraceUnsampled = 'U'
	entryMetaTransaction    = 't'
	entryMetaSpan           = 's'
)

// ErrNotFound is returned by by the Storage.IsTraceSampled method,
// for non-existing trace IDs.
var ErrNotFound = errors.New("key not found")

var json = jsoniter.ConfigFastest

// Storage provides storage for sampled transactions and spans,
// and for recording trace sampling decisions.
type Storage struct {
	db  *badger.DB
	ttl time.Duration
}

// New returns a new Storage using db and ttl.
func New(db *badger.DB, ttl time.Duration) *Storage {
	return &Storage{db: db, ttl: ttl}
}

// NewWriter returns a new Writer for writing events to storage.
//
// The returned writer must be closed when it is no longer needed.
func (s *Storage) NewWriter() *Writer {
	return &Writer{
		s:   s,
		txn: s.db.NewTransaction(true),
	}
}

// Writer provides a means of batched writing of events to storage.
//
// Writer is not safe for concurrent access. All operations that involve
// a given trace ID should be performed with the same Writer in order to
// avoid conflicts, e.g. by using consistent hashing to distribute to
// one of a set of Writers.
type Writer struct {
	s             *Storage
	txn           *badger.Txn
	pendingWrites int
}

// Close closes the writer. Any writes that have not been flushed may be lost.
//
// This must be called when the writer is no longer needed, in order to reclaim
// resources.
func (w *Writer) Close() {
	w.txn.Discard()
}

// Flush waits for preceding writes to be committed to storage.
//
// Flush must be called to ensure writes are committed to storage.
// If Flush is not called before the writer is closed, then writes
// may be lost.
func (w *Writer) Flush() error {
	err := w.txn.Commit()
	w.txn = w.s.db.NewTransaction(true)
	w.pendingWrites = 0
	return err
}

// WriteTraceSampled records the tail-sampling decision for the given trace ID.
func (w *Writer) WriteTraceSampled(traceID string, sampled bool) error {
	key := makeTraceKey(traceID)
	var meta uint8 = entryMetaTraceUnsampled
	if sampled {
		meta = entryMetaTraceSampled
	}
	entry := badger.NewEntry(key[:], nil).WithMeta(meta)
	return w.writeEntry(entry.WithTTL(w.s.ttl))
}

// IsTraceSampled reports whether traceID belongs to a trace that is sampled
// or unsampled. If no sampling decision has been recorded, IsTraceSampled
// returns ErrNotFound.
func (w *Writer) IsTraceSampled(traceID string) (bool, error) {
	key := makeTraceKey(traceID)
	item, err := w.txn.Get(key[:])
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, ErrNotFound
		}
		return false, err
	}
	return item.UserMeta() == entryMetaTraceSampled, nil
}

// WriteTransaction writes tx to storage.
//
// WriteTransaction may return before the write is committed to storage.
// Call Flush to ensure the write is committed.
func (w *Writer) WriteTransaction(tx *model.Transaction) error {
	if tx.Sampled != nil && !*tx.Sampled {
		return errors.New("transaction is not sampled")
	}
	key := makeEventKey(tx.TraceID, tx.ID)
	return w.writeEvent(key[:], tx, entryMetaTransaction)
}

// WriteSpan writes span to storage.
//
// WriteSpan may return before the write is committed to storage.
// Call Flush to ensure the write is committed.
func (w *Writer) WriteSpan(span *model.Span) error {
	key := makeEventKey(span.TraceID, span.ID)
	return w.writeEvent(key[:], span, entryMetaSpan)
}

func (w *Writer) writeEvent(key []byte, value interface{}, meta byte) error {
	// TODO(axw) more efficient codec for the model objects, e.g. protobuf.
	// NOTE(axw) encoding/json is faster for encoding, json-iterator is faster for decoding.
	data, err := stdjson.Marshal(value)
	if err != nil {
		return err
	}
	return w.writeEntry(badger.NewEntry(key, data).WithMeta(meta).WithTTL(w.s.ttl))
}

func (w *Writer) writeEntry(e *badger.Entry) error {
	w.pendingWrites++
	err := w.txn.SetEntry(e)
	if err != badger.ErrTxnTooBig {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return w.txn.SetEntry(e)
}

// ReadEvents reads events with the given trace ID from storage into a batch.
func (w *Writer) ReadEvents(traceID string, out *model.Batch) error {
	traceKey := makeTraceKey(traceID)
	traceKeyPrefix := append(traceKey, ':')

	opts := badger.DefaultIteratorOptions
	opts.Prefix = traceKeyPrefix

	// NewIterator slows down with uncommitted writes, as it must sort
	// all keys lexicographically. If there are a significant number of
	// writes pending, flush first.
	if w.pendingWrites > 100 {
		if err := w.Flush(); err != nil {
			return err
		}
	}

	iter := w.txn.NewIterator(opts)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if item.IsDeletedOrExpired() {
			continue
		}
		switch item.UserMeta() {
		case entryMetaTransaction:
			var event model.Transaction
			if err := item.Value(func(data []byte) error {
				return json.Unmarshal(data, &event)
			}); err != nil {
				return err
			}
			out.Transactions = append(out.Transactions, &event)
		case entryMetaSpan:
			var event model.Span
			if err := item.Value(func(data []byte) error {
				return json.Unmarshal(data, &event)
			}); err != nil {
				return err
			}
			out.Spans = append(out.Spans, &event)
		default:
			// Unknown entry meta: ignore.
			continue
		}
	}
	return nil
}

func makeTraceKey(traceID string) []byte {
	return []byte(traceID)
}

func makeEventKey(traceID, spanID string) []byte {
	buf := append([]byte(traceID), ':')
	return append(buf, spanID...)
}
