package breakdownmetrics

import (
	"context"
	"time"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/breakdownmetrics/deadlinestorage"
)

type TraceEventsReader interface {
	ReadTraceEvents(traceID string, out *model.Batch) error
}

type TraceEventWriter interface {
	WriteTraceEvent(traceID, id string, event *model.APMEvent) error
}

type DeadlineStorage interface {
	Flush() error
	WriteTraceDeadline(traceID string, deadline time.Time) error
	ReadTraceDeadlines(ctx context.Context, checkInterval time.Duration, out chan<- deadlinestorage.TraceDeadline) error
}
