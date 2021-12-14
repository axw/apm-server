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

	logs "github.com/elastic/apm-server/log"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	metricsetName = "span_breakdown"
)

// AggregatorConfig holds configuration for creating an Aggregator.
type AggregatorConfig struct {
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

	// TODO(axw) don't store all events in memory. Can we store them
	// in Badger? e.g.
	//
	// - individual events, keyed by trace ID + span/transaction ID
	// - mapping from events to child span IDs
	//
	// Then we just need to keep the per-trace timers in memory.

	mu      sync.Mutex
	wg      sync.WaitGroup
	traces  map[string]*traceEvents
	metrics map[aggregationKey]spanMetrics
}

type traceEvents struct {
	transactions []*model.APMEvent
	spans        map[string]*model.APMEvent
	childSpans   map[string][]string
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
		traces:   make(map[string]*traceEvents),
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
	var stop bool
	for !stop {
		select {
		case <-a.stopping:
			stop = true
			a.wg.Wait()
		case <-ticker.C:
		}
		if err := a.publish(context.Background()); err != nil {
			a.config.Logger.With(logp.Error(err)).Warnf(
				"publishing span metrics failed: %s", err,
			)
		}
	}
	return nil
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
		event := event
		switch event.Processor {
		case model.SpanProcessor:
			a.processSpan(&event)
		case model.TransactionProcessor:
			a.processTransaction(&event)
		}
	}
	return nil
}

func (a *Aggregator) processSpan(event *model.APMEvent) {
	if event.Span.RepresentativeCount <= 0 {
		// RepresentativeCount is zero when the sample rate is unknown.
		// We cannot calculate accurate metrics without the sample rate,
		// so we don't calculate any at all in this case.
		return
	}
	trace := a.getTrace(event)
	trace.spans[event.Span.ID] = event
	if event.Parent.ID != "" {
		trace.childSpans[event.Parent.ID] = append(trace.childSpans[event.Parent.ID], event.Span.ID)
	} else if len(event.Child.ID) != 0 {
		trace.childSpans[event.Span.ID] = append(trace.childSpans[event.Span.ID], event.Child.ID...)
	}
}

func (a *Aggregator) processTransaction(event *model.APMEvent) {
	if event.Transaction.RepresentativeCount == 0 {
		// RepresentativeCount is zero when the sample rate is unknown.
		// We cannot calculate accurate metrics without the sample rate,
		// so we don't calculate any at all in this case.
		return
	}
	trace := a.getTrace(event)
	trace.transactions = append(trace.transactions, event)
}

func (a *Aggregator) getTrace(event *model.APMEvent) *traceEvents {
	traceID := event.Trace.ID
	trace, ok := a.traces[traceID]
	if !ok {
		trace = &traceEvents{
			spans:      make(map[string]*model.APMEvent),
			childSpans: make(map[string][]string),
		}
		a.traces[traceID] = trace

		// After config.Duration elapses, or when the aggregator is stopped,
		// process all of the accumulated trace events as a single unit.
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()

			timer := time.NewTimer(a.config.Duration)
			select {
			case <-timer.C:
			case <-a.stopping:
				timer.Stop()
			}

			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.traces, traceID)

			// Aggregate each transactions and its reachable spans.
			for _, tx := range trace.transactions {
				a.aggregateSelfTime(trace, tx, tx)
			}
		}()
	}
	return trace
}

func (a *Aggregator) aggregateSelfTime(
	trace *traceEvents,
	transaction *model.APMEvent,
	event *model.APMEvent,
) {
	// For composite spans we use the composite sum duration, which is the sum of
	// pre-aggregated spans and excludes time gaps that are counted in the reported
	// span duration. For non-composite spans we just use the reported span duration.
	spanCount := 1
	spanDuration := event.Event.Duration
	var spanID string
	var spanType, spanSubtype string
	if event.Processor == model.SpanProcessor {
		spanID = event.Span.ID
		spanType = event.Span.Type
		spanSubtype = event.Span.Subtype
		if event.Span.Composite != nil {
			spanCount = event.Span.Composite.Count
			spanDuration = time.Duration(event.Span.Composite.Sum * float64(time.Millisecond))
		}
	} else {
		spanID = event.Transaction.ID
		spanType = "app"
	}

	// Calculate self_time by subtracting time overlapping with children.
	childIDs := trace.childSpans[spanID]
	sort.Slice(childIDs, func(i, j int) bool {
		ci := trace.spans[childIDs[i]]
		cj := trace.spans[childIDs[j]]
		if ci.Timestamp.Before(cj.Timestamp) {
			return true
		}
		return ci.Timestamp.Equal(cj.Timestamp) && ci.Event.Duration < cj.Event.Duration
	})
	start := event.Timestamp
	end := event.Timestamp.Add(spanDuration)
	for _, childID := range childIDs {
		child := trace.spans[childID]
		a.aggregateSelfTime(trace, transaction, child)

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
