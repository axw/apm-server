// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/hashicorp/go-multierror"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

type shardedStorage struct {
	writers []lockedStorage
}

func newShardedStorage(storage *eventstorage.Storage) *shardedStorage {
	s := &shardedStorage{
		// Create as many writers as there are CPUs, so we can ideally
		// minimise lock contention across multiple event streams.
		writers: make([]lockedStorage, runtime.NumCPU()),
	}
	for i := range s.writers {
		s.writers[i].w = storage.NewWriter()
	}
	return s
}

func (s *shardedStorage) Flush() error {
	var result error
	for i := range s.writers {
		if err := s.writers[i].Flush(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

func (s *shardedStorage) ReadEvents(traceID string, out *model.Batch) error {
	return s.getWriter(traceID).ReadEvents(traceID, out)
}

func (s *shardedStorage) WriteTransaction(tx *model.Transaction) error {
	return s.getWriter(tx.TraceID).WriteTransaction(tx)
}

func (s *shardedStorage) WriteSpan(span *model.Span) error {
	return s.getWriter(span.TraceID).WriteSpan(span)
}

func (s *shardedStorage) WriteTraceSampled(traceID string, sampled bool) error {
	return s.getWriter(traceID).WriteTraceSampled(traceID, sampled)
}

func (s *shardedStorage) IsTraceSampled(traceID string) (bool, error) {
	return s.getWriter(traceID).IsTraceSampled(traceID)
}

// getWriter returns an event storage writer for the given trace ID.
//
// This method is idempotent, which is necessary to avoid transaction
// conflicts and ensure all events are reported once a sampling decision
// has been recorded.
func (s *shardedStorage) getWriter(traceID string) *lockedStorage {
	var h xxhash.Digest
	h.WriteString(traceID)
	return &s.writers[h.Sum64()%uint64(len(s.writers))]
}

type lockedStorage struct {
	mu sync.Mutex
	w  *eventstorage.Writer
}

func (w *lockedStorage) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Flush()
}

func (w *lockedStorage) ReadEvents(traceID string, out *model.Batch) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.ReadEvents(traceID, out)
}

func (w *lockedStorage) WriteTransaction(tx *model.Transaction) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.WriteTransaction(tx)
}

func (w *lockedStorage) WriteSpan(s *model.Span) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.WriteSpan(s)
}

func (w *lockedStorage) WriteTraceSampled(traceID string, sampled bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.WriteTraceSampled(traceID, sampled)
}

func (w *lockedStorage) IsTraceSampled(traceID string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.IsTraceSampled(traceID)
}
