package deadlinestorage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
)

var (
	deadlineTTL = time.Minute
)

type DeadlineStorage struct {
	db *badger.DB

	mu          sync.RWMutex
	txn         *badger.Txn
	deadlineSeq int
}

func New(db *badger.DB) *DeadlineStorage {
	return &DeadlineStorage{db: db}
}

// Flush flushes any uncommitted writes.
//
// Flush must be called when disposing of the storage to release memory.
func (s *DeadlineStorage) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.txn == nil {
		return nil
	}
	txn := s.txn
	s.txn = nil
	return txn.Commit()
}

// WriteTraceDeadline records a deadline entry for the given trace ID.
//
// Multiple deadlines may be recorded for a given trace ID; they will be
// stored in order of deadline, such that they may be iterated in order
// of deadline.
//
// The deadline is rounded half away from zero to the nearest second.
func (s *DeadlineStorage) WriteTraceDeadline(traceID string, deadline time.Time) error {
	deadline = deadline.UTC().Round(time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	// We include the trace ID and a sequence number so we can perform
	// blind writes and avoid conflicts when writing multiple deadlines
	// with the same timestamp.
	key := deadline.AppendFormat(nil, time.RFC3339)
	key = append(key, fmt.Sprintf("@%010d", s.deadlineSeq)...)
	s.deadlineSeq++

	untilDeadline := time.Until(deadline)
	entry := badger.NewEntry(key[:], []byte(traceID)).WithTTL(untilDeadline + deadlineTTL)
	if s.txn == nil {
		s.txn = s.db.NewTransaction(true)
	}
	err := s.txn.SetEntry(entry)
	if err != badger.ErrTxnTooBig {
		return err
	}
	if err := s.txn.Commit(); err != nil {
		return err
	}
	s.txn = s.db.NewTransaction(true)
	return s.txn.SetEntry(entry)
}

// ReadTraceDeadlines reads trace deadlines from storage, sending them to out.
//
// ReadTraceDeadlines returns when an error occurs in reading entries from Badger,
// or when ctx is cancelled.
func (s *DeadlineStorage) ReadTraceDeadlines(ctx context.Context, checkInterval time.Duration, out chan<- TraceDeadline) error {
	var keyBuf []byte
	var valueBuf []byte

	var iter *badger.Iterator
	defer func() {
		if iter != nil {
			iter.Close()
		}
	}()

	var txn *badger.Txn
	for {
		if iter == nil {
			if err := s.Flush(); err != nil {
				return err
			}
			txn = s.db.NewTransaction(false)
			iter = txn.NewIterator(badger.DefaultIteratorOptions)
			iter.Seek(keyBuf)
		}
		if iter != nil && !iter.Valid() {
			iter.Close()
			txn.Discard()
			iter = nil
			txn = nil

			// Increment key, so we resume iteration at the next entry.
			//
			// The keys we produce have a fixed width, so just increment
			// bytes from the back, overflowing as needed.
			for i := len(keyBuf) - 1; i > 0; i-- {
				keyBuf[i]++
				if keyBuf[i] != 0 {
					break
				}
			}
		}
		if iter != nil {
			item := iter.Item()
			traceDeadline, newKeyBuf, newValueBuf, err := readTraceDeadline(item, keyBuf, valueBuf)
			if err != nil {
				return err
			}
			keyBuf = newKeyBuf
			valueBuf = newValueBuf
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- traceDeadline:
			}
			iter.Next()
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(checkInterval):
			}
		}
	}
}

func readTraceDeadline(
	item *badger.Item, keyBuf, valueBuf []byte,
) (traceDeadline TraceDeadline, newKeyBuf []byte, newValueBuf []byte, err error) {
	keyBuf = item.KeyCopy(keyBuf[:0])
	key := string(keyBuf)
	sep := strings.IndexRune(key, '@')
	key = key[:sep]
	deadline, err := time.Parse(time.RFC3339, key)
	if err != nil {
		return TraceDeadline{}, nil, nil, err
	}
	valueBuf, err = item.ValueCopy(valueBuf[:0])
	if err != nil {
		return TraceDeadline{}, nil, nil, err
	}
	traceID := string(valueBuf)
	return TraceDeadline{TraceID: traceID, Deadline: deadline}, keyBuf, valueBuf, nil
}

type TraceDeadline struct {
	TraceID  string
	Deadline time.Time
}
