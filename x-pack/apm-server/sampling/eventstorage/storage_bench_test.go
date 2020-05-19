// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eventstorage_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

func BenchmarkWriteTransaction(b *testing.B) {
	db := newBadgerDB(b, badgerOptions)
	ttl := time.Minute
	store := eventstorage.New(db, ttl)
	writer := store.NewWriter()
	defer writer.Close()

	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	transactionID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	transaction := &model.Transaction{
		TraceID: hex.EncodeToString(traceID),
		ID:      hex.EncodeToString(transactionID),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := writer.WriteTransaction(transaction); err != nil {
			b.Fatal(err)
		}
	}
	assert.NoError(b, writer.Flush())
}

func BenchmarkReadEvents(b *testing.B) {
	traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	// Test with varying numbers of events in the trace.
	counts := []int{0, 1, 10, 100, 1000}
	for _, count := range counts {
		b.Run(fmt.Sprintf("%d events", count), func(b *testing.B) {
			db := newBadgerDB(b, badgerOptions)
			ttl := time.Minute
			store := eventstorage.New(db, ttl)
			writer := store.NewWriter()
			defer writer.Close()

			for i := 0; i < count; i++ {
				var transactionID [8]byte
				binary.LittleEndian.PutUint64(transactionID[:], uint64(i+1))
				transaction := &model.Transaction{
					TraceID: hex.EncodeToString(traceID),
					ID:      hex.EncodeToString(transactionID[:]),
				}
				if err := writer.WriteTransaction(transaction); err != nil {
					b.Fatal(err)
				}
			}
			require.NoError(b, writer.Flush())

			b.ResetTimer()
			var batch model.Batch
			for i := 0; i < b.N; i++ {
				batch.Reset()
				if err := writer.ReadEvents(hex.EncodeToString(traceID), &batch); err != nil {
					b.Fatal(err)
				}
				if batch.Len() != count {
					panic(fmt.Errorf(
						"event count mismatch: expected %d, got %d",
						count, batch.Len(),
					))
				}
			}
		})
	}
}
