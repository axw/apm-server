// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"context"
	"math"
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
	publisher       sampledTraceIDsPublisher
	reporter        publish.Reporter
	sampledTraceIDs <-chan string

	// ingestRateCoefficient is λ, the coefficient for calculating
	// the exponentially weighted moving average ingest rate.
	ingestRateCoefficient float64

	closeOnce sync.Once
	closing   chan struct{}
	errgroup  *errgroup.Group
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
	ingestRateCoefficient float64,
) *tailSampler {
	sampledTraceIDs := make(chan string)
	s := &tailSampler{
		groups:                groups,
		writer:                writer,
		reader:                reader,
		publisher:             publisher,
		reporter:              reporter,
		sampledTraceIDs:       sampledTraceIDs,
		ingestRateCoefficient: ingestRateCoefficient,
		closing:               make(chan struct{}),
	}
	var ctx context.Context
	s.errgroup, ctx = errgroup.WithContext(context.Background())
	s.errgroup.Go(func() error {
		defer close(sampledTraceIDs)
		return subscriber.SubscribeSampledTraceIDs(ctx, sampledTraceIDs)
	})
	s.errgroup.Go(func() error {
		return s.loop(ctx, flushInterval)
	})
	return s
}

func (s *tailSampler) Close() error {
	s.closeOnce.Do(func() {
		close(s.closing)
	})
	if err := s.errgroup.Wait(); err != context.Canceled {
		return err
	}
	return nil
}

func (s *tailSampler) loop(ctx context.Context, flushInterval time.Duration) error {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		<-s.closing
	}()

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
			s.finalizeSampling(&localSampledTraceIDs)
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

// finalizeSampling locks all sampling reservoirs and appends their trace IDs
// to *out. On return the sampling reservoirs will be reset.
func (s *tailSampler) finalizeSampling(out *[]string) {
	s.groups.mu.Lock()
	defer s.groups.mu.Unlock()
	for _, groups := range s.groups.groups {
		for _, group := range groups {
			*out = append(*out, s.finalizeSamplingTraceGroup(group)...)
		}
	}
	// TODO(axw) delete groups with minimum reservoir size, so we can reuse.
	// This would require active/inactive groups, for synchronisation with
	// the updates.
}

func (s *tailSampler) finalizeSamplingTraceGroup(g *traceGroup) []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ingestRate == 0 {
		g.ingestRate = float64(g.total)
	} else {
		g.ingestRate *= 1 - s.ingestRateCoefficient
		g.ingestRate += s.ingestRateCoefficient * float64(g.total)
	}
	desiredTotal := int(math.Round(g.samplingFraction * float64(g.total)))
	g.total = 0

	for n := g.reservoir.Len(); n > desiredTotal; n-- {
		// The reservoir is larger than the desired fraction of the
		// observed total number of traces in this interval. Pop the
		// lowest weighed traces to limit to the desired total.
		g.reservoir.Pop()
	}
	traceIDs := g.reservoir.Values()

	// Resize the reservoir, so that it can hold the desired fraction of
	// the observed ingest rate.
	newReservoirSize := int(math.Round(g.samplingFraction * g.ingestRate))
	if newReservoirSize < minReservoirSize {
		newReservoirSize = minReservoirSize
	}
	g.reservoir.Reset()
	g.reservoir.Resize(newReservoirSize)
	return traceIDs
}
