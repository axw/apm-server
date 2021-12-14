// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

package breakdownmetrics_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/breakdownmetrics"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

func TestNewAggregatorConfigInvalid(t *testing.T) {
	report := makeErrBatchProcessor(nil)

	var traceEventsReader struct {
		breakdownmetrics.TraceEventsReader
	}

	type test struct {
		config breakdownmetrics.AggregatorConfig
		err    string
	}

	for _, test := range []test{{
		config: breakdownmetrics.AggregatorConfig{},
		err:    "TraceEventsReader unspecified",
	}, {
		config: breakdownmetrics.AggregatorConfig{
			TraceEventsReader: &traceEventsReader,
		},
		err: "BatchProcessor unspecified",
	}, {
		config: breakdownmetrics.AggregatorConfig{
			TraceEventsReader: &traceEventsReader,
			BatchProcessor:    report,
		},
		err: "MaxGroups unspecified or negative",
	}, {
		config: breakdownmetrics.AggregatorConfig{
			TraceEventsReader: &traceEventsReader,
			BatchProcessor:    report,
			MaxGroups:         1,
		},
		err: "Duration unspecified or negative",
	}, {
		config: breakdownmetrics.AggregatorConfig{
			TraceEventsReader: &traceEventsReader,
			BatchProcessor:    report,
			MaxGroups:         1,
			Duration:          time.Second,
		},
		err: "Interval unspecified or negative",
	}} {
		agg, err := breakdownmetrics.NewAggregator(test.config)
		require.Error(t, err)
		require.Nil(t, agg)
		assert.EqualError(t, err, "invalid aggregator config: "+test.err)
	}
}

func TestAggregator(t *testing.T) {
	t0 := time.Unix(0, 0)
	tx := makeTransaction("trace_id", "transaction_id", "transaction_type", "transaction_name", t0, 30*time.Second)
	span1 := makeSpan("trace_id", "span1_id", "transaction_id", "app", "", t0.Add(10*time.Second), 10*time.Second)
	span2 := makeSpan("trace_id", "span2_id", "span1_id", "db", "mysql", t0.Add(15*time.Second), 10*time.Second)

	var traceEventsReader traceEventsReaderFunc = func(traceID string, out *model.Batch) error {
		*out = append(*out, tx, span1, span2)
		return nil
	}

	batches := make(chan model.Batch, 1)
	agg, err := breakdownmetrics.NewAggregator(breakdownmetrics.AggregatorConfig{
		TraceEventsReader: traceEventsReader,
		BatchProcessor:    makeChanBatchProcessor(batches),
		Duration:          time.Millisecond,
		Interval:          time.Millisecond,
		MaxGroups:         1000,
	})
	require.NoError(t, err)

	err = agg.ProcessBatch(context.Background(), &model.Batch{tx, span1, span2})
	require.NoError(t, err)

	// Start the aggregator after processing to ensure metrics are aggregated deterministically.
	//
	// Batches are processed as a unit, so there is no risk of partial metrics being published.
	go agg.Run()
	defer agg.Stop(context.Background())

	batch := expectBatch(t, batches)
	metricsets := batchMetricsets(t, batch)
	require.Len(t, metricsets, 2)
	sort.Slice(metricsets, func(i, j int) bool {
		return metricsets[i].Span.Subtype < metricsets[j].Span.Subtype
	})

	assert.Equal(t, []model.APMEvent{{
		Timestamp: t0,
		Processor: model.MetricsetProcessor,
		Metricset: &model.Metricset{Name: "span_breakdown"},
		Transaction: &model.Transaction{
			Type: "transaction_type",
			Name: "transaction_name",
		},
		Span: &model.Span{
			Type: "app",
			SelfTime: model.AggregatedDuration{
				Count: 2,
				Sum:   25 * time.Second,
			},
		},
	}, {
		Timestamp: t0,
		Processor: model.MetricsetProcessor,
		Metricset: &model.Metricset{Name: "span_breakdown"},
		Transaction: &model.Transaction{
			Type: "transaction_type",
			Name: "transaction_name",
		},
		Span: &model.Span{
			Type:    "db",
			Subtype: "mysql",
			SelfTime: model.AggregatedDuration{
				Count: 1,
				Sum:   10 * time.Second,
			},
		},
	}}, metricsets)
}

func BenchmarkAggregator(b *testing.B) {
	tempdir, err := ioutil.TempDir("", "breakdownmetrics")
	require.NoError(b, err)
	b.Cleanup(func() { os.RemoveAll(tempdir) })

	badgerDB, err := eventstorage.OpenBadger(tempdir, 0)
	require.NoError(b, err)
	b.Cleanup(func() { badgerDB.Close() })

	t0 := time.Unix(0, 0)
	makeBatch := func(traceID string) model.Batch {
		tx := makeTransaction(traceID, "transaction_id", "transaction_type", "transaction_name", t0, 30*time.Second)
		span1 := makeSpan(traceID, "span1_id", "transaction_id", "app", "", t0.Add(10*time.Second), 10*time.Second)
		span2 := makeSpan(traceID, "span2_id", "span1_id", "db", "mysql", t0.Add(15*time.Second), 10*time.Second)
		return model.Batch{tx, span1, span2}
	}

	storage := eventstorage.New(badgerDB, eventstorage.JSONCodec{}, time.Minute)
	readWriter := storage.NewReadWriter()
	for i := 0; i < b.N; i++ {
		traceID := fmt.Sprintf("trace_%d", i)
		for _, event := range makeBatch(traceID) {
			var id string
			if event.Processor == model.TransactionProcessor {
				id = event.Transaction.ID
			} else {
				id = event.Span.ID
			}
			if err := readWriter.WriteTraceEvent(traceID, id, &event); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ResetTimer()

	agg, err := breakdownmetrics.NewAggregator(breakdownmetrics.AggregatorConfig{
		TraceEventsReader: readWriter,
		BatchProcessor:    makeErrBatchProcessor(nil),
		Duration:          time.Millisecond,
		Interval:          time.Minute,
		MaxGroups:         1000,
	})
	require.NoError(b, err)

	go agg.Run()
	defer agg.Stop(context.Background())

	for i := 0; i < b.N; i++ {
		traceID := fmt.Sprintf("trace_%d", i)
		batch := makeBatch(traceID)
		if err := agg.ProcessBatch(context.Background(), &batch); err != nil {
			b.Fatal(err)
		}
	}
}

func makeTransaction(
	traceID, transactionID, transactionType, transactionName string,
	start time.Time, duration time.Duration,
) model.APMEvent {
	return model.APMEvent{
		Event:     model.Event{Duration: duration},
		Processor: model.TransactionProcessor,
		Timestamp: start,
		Trace:     model.Trace{ID: traceID},
		Transaction: &model.Transaction{
			ID:                  transactionID,
			Type:                transactionType,
			Name:                transactionName,
			RepresentativeCount: 1,
		},
	}
}

func makeSpan(
	traceID, spanID, parentID, spanType, spanSubtype string,
	start time.Time, duration time.Duration,
) model.APMEvent {
	return model.APMEvent{
		Event:     model.Event{Duration: duration},
		Processor: model.SpanProcessor,
		Timestamp: start,
		Trace:     model.Trace{ID: traceID},
		Parent:    model.Parent{ID: parentID},
		Span: &model.Span{
			ID:                  spanID,
			Type:                spanType,
			Subtype:             spanSubtype,
			RepresentativeCount: 1,
		},
	}
}

func makeErrBatchProcessor(err error) model.BatchProcessor {
	return model.ProcessBatchFunc(func(context.Context, *model.Batch) error { return err })
}

func makeChanBatchProcessor(ch chan<- model.Batch) model.BatchProcessor {
	return model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- *batch:
			return nil
		}
	})
}

func expectBatch(t *testing.T, ch <-chan model.Batch) model.Batch {
	t.Helper()
	select {
	case batch := <-ch:
		return batch
	case <-time.After(time.Second * 5):
		t.Fatal("expected publish")
	}
	panic("unreachable")
}

func batchMetricsets(t testing.TB, batch model.Batch) []model.APMEvent {
	var metricsets []model.APMEvent
	for _, event := range batch {
		if event.Metricset == nil {
			continue
		}
		metricsets = append(metricsets, event)
	}
	return metricsets
}

type traceEventsReaderFunc func(traceID string, out *model.Batch) error

func (f traceEventsReaderFunc) ReadTraceEvents(traceID string, out *model.Batch) error {
	return f(traceID, out)
}
