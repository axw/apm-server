package breakdownmetrics

import "github.com/elastic/apm-server/model"

type TraceEventsReader interface {
	ReadTraceEvents(traceID string, out *model.Batch) error
}

type TraceEventWriter interface {
	WriteTraceEvent(traceID, id string, event *model.APMEvent) error
}
