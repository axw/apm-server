// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"context"
	"sync"
	"time"

	logs "github.com/elastic/apm-server/log"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/beats/v7/libbeat/logp"
	"golang.org/x/sync/errgroup"
)

// tailSampler periodically flushes trace group reservoirs, reading
// the events associated with sampled trace IDs from storage and sending
// them to the reporter.
type tailSampler struct {
	groups          *traceGroups
	reader          eventReader
	writer          traceSampledWriter
	subscriber      sampledTraceIDsSubscriber
	publisher       sampledTraceIDsPublisher
	reporter        publish.Reporter
	flushInterval   time.Duration
	sampledTraceIDs chan string

	stopMu   sync.Mutex
	stopping chan struct{}
	stopped  chan struct{}
}

type eventReader interface {
	// ReadEvents reads trace events from local storage.
	ReadEvents(traceID string, out *model.Batch) error
}

type traceSampledWriter interface {
	// WriteTraceSampled writes a trace sampling decision to local storage.
	WriteTraceSampled(traceID string, sampled bool) error
}

type sampledTraceIDsSubscriber interface {
	// SubscribeSampledTraceIDs subscribes to sampled trace IDs, sending them to the channel.
	//
	// SubscribeSampledTraceIDs returns when ctx is cancelled, or a fatal error occurs.
	SubscribeSampledTraceIDs(ctx context.Context, traceIDs chan<- string) error
}

type sampledTraceIDsPublisher interface {
	// PublishSampledTraceIDs publishes the IDs of one or more traces sampled locally.
	//
	// PublishSampledTraceIDs is not a guaranteed operation; it may be asynchronous,
	// e.g. bulk indexing into Elasticsearch.
	PublishSampledTraceIDs(ctx context.Context, traceIDs ...string) error
}

// newTailSampler returns a new tailSampler, starting its flush loop
// in a background goroutine. The sampler's Close method should be
// called when it is no longer needed.
func newTailSampler(
	groups *traceGroups,
	flushInterval time.Duration,
	writer traceSampledWriter,
	reader eventReader,
	subscriber sampledTraceIDsSubscriber,
	publisher sampledTraceIDsPublisher,
	reporter publish.Reporter,
) *tailSampler {
	s := &tailSampler{
		groups:          groups,
		writer:          writer,
		reader:          reader,
		subscriber:      subscriber,
		publisher:       publisher,
		reporter:        reporter,
		flushInterval:   flushInterval,
		sampledTraceIDs: make(chan string),

		stopping: make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	return s
}

// Stop stops the tail sampler if it is running, waiting for it to stop or for
// the context to be cancelled.
//
// After Stop has been called the tail sampler cannot be reused, as the Run method
// will always return immediately.
func (s *tailSampler) Stop(ctx context.Context) error {
	s.stopMu.Lock()
	select {
	case <-s.stopping:
		// already stopping
	default:
		close(s.stopping)
	}
	s.stopMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stopped:
	}
	return nil
}

// Run runs the tail sampler, receiving sampling decisions from the subscriber
// periodically finalising sampling reservoirs, and sending tail-sampled trace
// events to the output.
//
// Run returns when either a fatal error occurs, or the tail sampler's Stop method
// is invoked.
func (s *tailSampler) Run() error {
	defer func() {
		s.stopMu.Lock()
		defer s.stopMu.Unlock()
		select {
		case <-s.stopped:
		default:
			close(s.stopped)
		}
	}()

	errgroup, ctx := errgroup.WithContext(context.Background())
	errgroup.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopping:
			return context.Canceled
		}
	})
	errgroup.Go(func() error {
		defer close(s.sampledTraceIDs)
		return s.subscriber.SubscribeSampledTraceIDs(ctx, s.sampledTraceIDs)
	})
	errgroup.Go(func() error {
		return s.loop(ctx)
	})
	if err := errgroup.Wait(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

func (s *tailSampler) loop(ctx context.Context) error {
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	var events model.Batch
	var localSampledTraceIDs []string
	logger := logp.NewLogger(logs.Sampling)
	remoteSampledTraceIDs := s.sampledTraceIDs

	// TODO(axw) on errors, log and continue
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case traceID, ok := <-remoteSampledTraceIDs:
			if !ok {
				remoteSampledTraceIDs = nil
				continue
			}
			logger.Debug("received sampled trace ID")
			if err := s.writer.WriteTraceSampled(traceID, true); err != nil {
				return err
			}
			if err := s.reader.ReadEvents(traceID, &events); err != nil {
				return err
			}
		case <-ticker.C:
			logger.Debug("finalizing sampling reservoir")
			localSampledTraceIDs = s.groups.finalizeSampledTraces(localSampledTraceIDs)
			if err := s.publisher.PublishSampledTraceIDs(ctx, localSampledTraceIDs...); err != nil {
				return err
			}
			// TODO(axw) publish events incrementally, instead of all at once.
			for _, traceID := range localSampledTraceIDs {
				if err := s.writer.WriteTraceSampled(traceID, true); err != nil {
					return err
				}
				if err := s.reader.ReadEvents(traceID, &events); err != nil {
					return err
				}
			}
			localSampledTraceIDs = localSampledTraceIDs[:0]
		}

		transformables := events.Transformables()
		if len(transformables) > 0 {
			logger.Debugf("publishing %d events", len(transformables))
			if err := s.reporter(ctx, publish.PendingReq{
				Transformables: transformables,
				Trace:          true,
			}); err != nil {
				// TODO(axw) should this just log a warning?
				return err
			}
		}
		events.Reset()
	}
}
