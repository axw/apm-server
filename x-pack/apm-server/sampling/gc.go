// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"golang.org/x/sync/errgroup"
)

// storageGarbageCollector periodically garbage collects the Badger value log.
type storageGarbageCollector struct {
	db        *badger.DB
	errgroup  errgroup.Group
	closeOnce sync.Once
	closing   chan struct{}
}

func newStorageGarbageCollector(db *badger.DB, gcInterval time.Duration) *storageGarbageCollector {
	c := &storageGarbageCollector{
		db:      db,
		closing: make(chan struct{}),
	}
	c.errgroup.Go(func() error {
		return c.loop(gcInterval)
	})
	return c
}

func (s *storageGarbageCollector) Close() error {
	s.closeOnce.Do(func() {
		close(s.closing)
	})
	return s.errgroup.Wait()
}

func (s *storageGarbageCollector) loop(gcInterval time.Duration) error {
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.closing:
			return nil
		case <-ticker.C:
		}
		const discardRatio = 0.5
		if err := s.db.RunValueLogGC(discardRatio); err != nil {
			return err
		}
	}
}
