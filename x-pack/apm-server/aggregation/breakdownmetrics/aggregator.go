// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

package breakdownmetrics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/logp"

	logs "github.com/elastic/apm-server/log"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/breakdownmetrics/deadlinestorage"
)

const (
	metricsetName = "span_breakdown"
)

// AggregatorConfig holds configuration for creating an Aggregator.
type AggregatorConfig struct {
	TraceEventsReader TraceEventsReader
	DeadlineStorage   DeadlineStorage

	// BatchProcessor is a model.BatchProcessor for asynchronously
	// processing metrics documents.
	BatchProcessor model.BatchProcessor

	// MaxGroups is the maximum number of distinct breakdown metrics
	// to store within an aggregation period. Once this number of groups
	// is reached, a warning will be logged and no new breakdown metrics
	// will be produced for the aggregation period.
	MaxGroups int

	// Duration is the maximum aggregation duration for a transaction.
	//
	// Transaction and span events will be aggregated and discarded after
	// this timer elapses, starting from the time the first trace event
	// is received. If within that time period no transaction is received,
	// no metrics will be produced.
	Duration time.Duration

	// Interval is the time interval between publication of metrics.
	Interval time.Duration

	// Logger is the logger for logging metrics aggregation/publishing.
	//
	// If Logger is nil, a new logger will be constructed.
	Logger *logp.Logger
}

// Validate validates the aggregator config.
func (config AggregatorConfig) Validate() error {
	if config.TraceEventsReader == nil {
		return errors.New("TraceEventsReader unspecified")
	}
	if config.DeadlineStorage == nil {
		return errors.New("DeadlineStorage unspecified")
	}
	if config.BatchProcessor == nil {
		return errors.New("BatchProcessor unspecified")
	}
	if config.MaxGroups <= 0 {
		return errors.New("MaxGroups unspecified or negative")
	}
	if config.Duration <= 0 {
		return errors.New("Duration unspecified or negative")
	}
	if config.Interval <= 0 {
		return errors.New("Interval unspecified or negative")
	}
	return nil
}

// Aggregator accumulates transaction and span events, and aggregates their
// durations into breakdown metrics: the amount of time spent by span type
// and subtype, per transaction group.
type Aggregator struct {
	config AggregatorConfig

	stopMu   sync.Mutex
	stopping chan struct{}
	stopped  chan struct{}

	traces chan string
	wg     sync.WaitGroup

	mu      sync.Mutex
	metrics map[aggregationKey]spanMetrics
}

// NewAggregator returns a new Aggregator with the given config.
func NewAggregator(config AggregatorConfig) (*Aggregator, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid aggregator config")
	}
	if config.Logger == nil {
		config.Logger = logp.NewLogger(logs.SpanMetrics)
	}
	return &Aggregator{
		config:   config,
		stopping: make(chan struct{}),
		stopped:  make(chan struct{}),
		traces:   make(chan string),
		metrics:  make(map[aggregationKey]spanMetrics),
	}, nil
}

// Run runs the Aggregator, periodically publishing and clearing aggregated
// metrics. Run returns when either a fatal error occurs, or the Aggregator's
// Stop method is invoked.
func (a *Aggregator) Run() error {
	ticker := time.NewTicker(a.config.Interval)
	defer ticker.Stop()
	defer func() {
		a.stopMu.Lock()
		defer a.stopMu.Unlock()
		select {
		case <-a.stopped:
		default:
			close(a.stopped)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		flushTicker := time.NewTicker(time.Second) // TODO(axw) make configurable?
		defer flushTicker.Stop()
		defer a.config.DeadlineStorage.Flush()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-a.stopped:
				return nil
			case <-flushTicker.C:
				if err := a.config.DeadlineStorage.Flush(); err != nil {
					return err
				}
			case traceID := <-a.traces:
				deadline := time.Now().Add(a.config.Duration)
				if err := a.config.DeadlineStorage.WriteTraceDeadline(traceID, deadline); err != nil {
					return err
				}
			}
		}
	})
	deadlines := make(chan deadlinestorage.TraceDeadline)
	g.Go(func() error {
		defer close(deadlines)
		// TODO(axw) make check timeout configurable? Should be longer anyway.
		return a.config.DeadlineStorage.ReadTraceDeadlines(ctx, time.Millisecond, deadlines)
	})
	g.Go(func() error {
		timer := time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}
		for {
			var deadline deadlinestorage.TraceDeadline
			var ok bool
			select {
			case <-ctx.Done():
				return ctx.Err()
			case deadline, ok = <-deadlines:
				if !ok {
					return nil
				}
				timer.Reset(time.Until(deadline.Deadline))
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				// After config.Duration elapses, process all of the
				// accumulated trace events as a single unit.
				//
				// TODO(axw) consider starting another goroutine,
				// with a concurrency limit.
				if err := a.aggregateTrace(deadline.TraceID); err != nil {
					return err
				}
			}
		}
	})
	g.Go(func() error {
		var stop bool
		for !stop {
			select {
			case <-a.stopping:
				stop = true
			case <-ticker.C:
			}
			if err := a.publish(context.Background()); err != nil {
				a.config.Logger.With(logp.Error(err)).Warnf(
					"publishing span metrics failed: %s", err,
				)
			}
		}
		cancel() // stop other goroutines
		return nil
	})
	return g.Wait()
}

// Stop stops the Aggregator if it is running, waiting for it to flush any
// aggregated metrics and return, or for the context to be cancelled.
//
// After Stop has been called the aggregator cannot be reused, as the Run
// method will always return immediately.
func (a *Aggregator) Stop(ctx context.Context) error {
	a.stopMu.Lock()
	select {
	case <-a.stopped:
	case <-a.stopping:
		// Already stopping/stopped.
	default:
		close(a.stopping)
	}
	a.stopMu.Unlock()

	select {
	case <-a.stopped:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (a *Aggregator) publish(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.metrics) == 0 {
		return nil
	}

	batch := make(model.Batch, 0, len(a.metrics))
	for key, metrics := range a.metrics {
		delete(a.metrics, key)
		batch = append(batch, makeMetricset(key, metrics))
	}

	a.config.Logger.Debugf("publishing %d metricsets", len(batch))
	return a.config.BatchProcessor.ProcessBatch(ctx, &batch)
}

// ProcessBatch accumulates transaction and span events, grouping them by
// trace ID. When a transaction is received, it and its associated spans'
// durations will be aggregated into breakdown metrics.
//
// This method is expected to be used immediately prior to publishing the
// events.
func (a *Aggregator) ProcessBatch(ctx context.Context, b *model.Batch) error {
	// TODO(axw) don't process transactions/spans for agents that also
	// accumulate breakdown metrics. Or, we could just drop all breakdown
	// metrics sent by agents? Will that work for RUM? IIRC it produces
	// breakdowns for spans that are never reported (TLS, Connect, etc.)
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, event := range *b {
		if event.Processor != model.TransactionProcessor {
			continue
		}
		if event.Transaction.RepresentativeCount == 0 {
			// RepresentativeCount is zero when the sample rate is unknown.
			// We cannot calculate accurate metrics without the sample rate,
			// so we don't calculate any at all in this case.
			continue
		}
		a.processTransaction(event.Trace.ID, event.Transaction.ID)
	}
	return nil
}

func (a *Aggregator) processTransaction(traceID, transactionID string) {
	select {
	case <-a.stopping:
	case a.traces <- traceID:
	}
}

func (a *Aggregator) aggregateTrace(traceID string) error {
	// TODO(axw) recycle batches?
	var events model.Batch
	if err := a.config.TraceEventsReader.ReadTraceEvents(traceID, &events); err != nil {
		return err
	}

	graphNodes := make(map[string]*graphNode)
	for i := range events {
		event := &events[i]
		switch event.Processor {
		case model.TransactionProcessor:
			graphNodes[event.Transaction.ID] = &graphNode{APMEvent: event}
		case model.SpanProcessor:
			graphNodes[event.Span.ID] = &graphNode{APMEvent: event}
		}
	}
	for _, graphNode := range graphNodes {
		if graphNode.Processor != model.SpanProcessor {
			continue
		}
		if graphNode.Parent.ID != "" {
			if parent, ok := graphNodes[graphNode.Parent.ID]; ok {
				parent.children = append(parent.children, graphNode)
			}
		} else {
			for _, childID := range graphNode.Child.ID {
				if child, ok := graphNodes[childID]; ok {
					graphNode.children = append(graphNode.children, child)
				}
			}
		}
	}
	for _, graphNode := range graphNodes {
		sort.Slice(graphNode.children, func(i, j int) bool {
			ci := graphNode.children[i]
			cj := graphNode.children[j]
			if ci.Timestamp.Before(cj.Timestamp) {
				return true
			}
			return ci.Timestamp.Equal(cj.Timestamp) && ci.Event.Duration < cj.Event.Duration
		})
	}

	// Aggregate each transactions and its reachable spans.
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, graphNode := range graphNodes {
		if graphNode.Processor != model.TransactionProcessor {
			continue
		}
		a.aggregateSelfTime(graphNode.APMEvent, graphNode)
	}
	return nil
}

type graphNode struct {
	*model.APMEvent
	children []*graphNode
}

func (a *Aggregator) aggregateSelfTime(transaction *model.APMEvent, node *graphNode) {
	// For composite spans we use the composite sum duration, which is the sum of
	// pre-aggregated spans and excludes time gaps that are counted in the reported
	// span duration. For non-composite spans we just use the reported span duration.
	spanCount := 1
	spanDuration := node.Event.Duration
	var spanType, spanSubtype string
	if node.Processor == model.SpanProcessor {
		spanType = node.Span.Type
		spanSubtype = node.Span.Subtype
		if node.Span.Composite != nil {
			spanCount = node.Span.Composite.Count
			spanDuration = time.Duration(node.Span.Composite.Sum * float64(time.Millisecond))
		}
	} else {
		spanType = "app"
	}

	// Calculate self_time by subtracting time overlapping with children. Children
	// are already sorted by timestamp.
	start := node.Timestamp
	end := node.Timestamp.Add(spanDuration)
	for _, child := range node.children {
		a.aggregateSelfTime(transaction, child)

		childStart := child.Timestamp
		childDuration := child.Event.Duration
		childEnd := child.Timestamp.Add(childDuration)
		if childStart.After(end) {
			break
		}
		if !childStart.Before(start) {
			start = childStart
		}
		if childEnd.After(end) {
			childEnd = end
		}
		spanDuration -= childEnd.Sub(start)
	}

	// Update metricsets.
	key := makeAggregationKey(transaction, spanType, spanSubtype, a.config.Interval)
	metrics := a.metrics[key]
	metrics.count += float64(spanCount) * transaction.Transaction.RepresentativeCount
	metrics.sum += float64(spanDuration) * transaction.Transaction.RepresentativeCount
	a.metrics[key] = metrics
}

type aggregationKey struct {
	timestamp time.Time

	serviceName        string
	serviceEnvironment string
	transactionType    string
	transactionName    string

	spanType    string
	spanSubtype string
}

func makeAggregationKey(tx *model.APMEvent, spanType, spanSubtype string, interval time.Duration) aggregationKey {
	return aggregationKey{
		// Group metrics by time interval.
		timestamp: tx.Timestamp.Truncate(interval),

		serviceName:        tx.Service.Name,
		serviceEnvironment: tx.Service.Environment,
		transactionType:    tx.Transaction.Type,
		transactionName:    tx.Transaction.Name,

		spanType:    spanType,
		spanSubtype: spanSubtype,
	}
}

type spanMetrics struct {
	count float64
	sum   float64
}

func makeMetricset(key aggregationKey, metrics spanMetrics) model.APMEvent {
	return model.APMEvent{
		Timestamp: key.timestamp,
		Service: model.Service{
			Name:        key.serviceName,
			Environment: key.serviceEnvironment,
		},
		Processor: model.MetricsetProcessor,
		Metricset: &model.Metricset{Name: metricsetName},
		Transaction: &model.Transaction{
			Type: key.transactionType,
			Name: key.transactionName,
		},
		Span: &model.Span{
			Type:    key.spanType,
			Subtype: key.spanSubtype,
			SelfTime: model.AggregatedDuration{
				Count: int(math.Round(metrics.count)),
				Sum:   time.Duration(math.Round(metrics.sum)),
			},
		},
	}
}
