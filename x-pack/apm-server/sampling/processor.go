// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-multierror"

	logs "github.com/elastic/apm-server/log"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/pubsub"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// ErrStopped is returned when calling ProcessTransformables on a stopped Processor.
var ErrStopped = errors.New("processor is stopped")

// Processor is a tail-sampling event processor.
type Processor struct {
	config Config

	groups      *traceGroups
	db          *badger.DB
	storage     *eventstorage.ShardedReadWriter
	tailSampler *tailSampler
	gc          *storageGarbageCollector

	mu      sync.RWMutex
	stopped bool
	stopErr error
}

// NewProcessor returns a new Processor, for tail-sampling trace events.
func NewProcessor(config Config) (*Processor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := logp.NewLogger(logs.Sampling)
	badgerOpts := badger.DefaultOptions(config.StorageDir)
	badgerOpts.Logger = eventstorage.LogpAdaptor{Logger: logger}
	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	pubsub, err := pubsub.New(pubsub.Config{
		Client: config.Elasticsearch,
		Logger: logger,
		Index:  ".apm-trace-ids", // TODO(axw) make configurable?
		BeatID: "xyz",            // TODO(axw) pass in via config

		// Issue pubsub subscriber search requests at the same frequency
		// as publishing, so each server observes each other's sampled
		// trace IDs soon after they are published.
		SearchInterval: config.FlushInterval,

		// NOTE(axw) this is currently not configurable. The user can
		// configure the tail-sampling flush interval, but this flush
		// interval controls how long to wait until flushing the bulk
		// indexer.
		FlushInterval: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	eventCodec := eventstorage.JSONCodec{}
	storage := eventstorage.New(db, eventCodec, config.TTL)
	readWriter := storage.NewShardedReadWriter()

	p := &Processor{
		config:  config,
		groups:  newTraceGroups(config.MaxTraceGroups, config.DefaultSampleRate, config.IngestRateCoefficient),
		db:      db,
		storage: readWriter,
		gc:      newStorageGarbageCollector(db, config.StorageGCInterval),
	}
	p.tailSampler = newTailSampler(
		p.groups,
		config.FlushInterval,
		p.storage,
		p.storage,
		pubsub,
		pubsub,
		config.Reporter,
	)
	return p, nil
}

// Run runs the tail-sampling processor.
//
// Run returns when a fatal error occurs or the Stop method is invoked.
func (p *Processor) Run() error {
	return p.tailSampler.Run()
}

// Stop stops the processor, flushing and closing the event storage.
func (p *Processor) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return p.stopErr
	}
	stopErr := p.tailSampler.Stop(ctx)
	if err := p.storage.Flush(); err != nil {
		stopErr = multierror.Append(stopErr, err)
	}
	if err := p.gc.Close(); err != nil {
		stopErr = multierror.Append(stopErr, err)
	}
	p.storage.Close()
	if err := p.db.Close(); err != nil {
		stopErr = multierror.Append(stopErr, err)
	}
	p.stopErr = stopErr
	p.stopped = true
	return p.stopErr
}

// ProcessTransformables processes events, writing head sampled transactions and
// spans to storage (except where they are part of a trace that has already been
// tail sampled), discarding events that should not be sampled, and returning
// everything else to be published immediately.
func (p *Processor) ProcessTransformables(ctx context.Context, events []transform.Transformable) ([]transform.Transformable, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.stopped {
		return nil, ErrStopped
	}
	for i := 0; i < len(events); i++ {
		var drop bool
		var err error
		switch event := events[i].(type) {
		case *model.Transaction:
			drop, err = p.processTransaction(event)
		case *model.Span:
			drop, err = p.processSpan(event)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		if drop {
			n := len(events)
			events[i], events[n-1] = events[n-1], events[i]
			events = events[:n-1]
			i--
		}
	}
	return events, nil
}

func (p *Processor) processTransaction(tx *model.Transaction) (bool, error) {
	if tx.Sampled != nil && !*tx.Sampled {
		// (Head-based) unsampled transactions are passed through
		// by the tail sampler.
		return false, nil
	}

	traceSampled, err := p.storage.IsTraceSampled(tx.TraceID)
	switch err {
	case nil:
		// Tail-sampling decision has been made, index or drop the transaction.
		drop := !traceSampled
		return drop, nil
	case eventstorage.ErrNotFound:
		// Tail-sampling decision has not yet been made.
		break
	default:
		return false, err
	}

	if tx.ParentID != "" {
		// Non-root transaction: write to local storage while we wait
		// for a sampling decision.
		return true, p.storage.WriteTransaction(tx)
	}

	// Root transaction: apply reservoir sampling.
	reservoirSampled, err := p.groups.sampleTrace(tx)
	if err == errTooManyTraceGroups {
		// Too many trace groups, drop the transaction.
		//
		// TODO(axw) log a warning with a rate limit.
		// TODO(axw) should we have an "other" bucket to capture,
		//           and capture them with the default rate?
		//           likely does not make sense to reservoir sample,
		//           except when there is a single logical trace group
		//           with high cardinality transaction names.
		return true, nil
	} else if err != nil {
		return false, err
	}

	if !reservoirSampled {
		// Write the non-sampling decision to storage to avoid further
		// writes for the trace ID, and then drop the transaction.
		//
		// This is a local optimisation only. To avoid creating network
		// traffic and load on Elasticsearch for uninteresting root
		// transactions, we do not propagate this to other APM Servers.
		return true, p.storage.WriteTraceSampled(tx.TraceID, false)
	}

	// The root transaction was admitted to the sampling reservoir, so we
	// can proceed to write the transaction to storage and then drop it;
	// we may index it later, after finalising the sampling decision.
	return true, p.storage.WriteTransaction(tx)
}

func (p *Processor) processSpan(span *model.Span) (bool, error) {
	traceSampled, err := p.storage.IsTraceSampled(span.TraceID)
	if err != nil {
		if err == eventstorage.ErrNotFound {
			// Tail-sampling decision has not yet been made, write span to local storage.
			return true, p.storage.WriteSpan(span)
		}
		return false, err
	}
	// Tail-sampling decision has been made, index or drop the event.
	drop := !traceSampled
	return drop, nil
}
