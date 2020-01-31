package transformer

import (
	"regexp"
	"time"

	apmerror "github.com/elastic/apm-server/model/error"
	"github.com/elastic/apm-server/model/metadata"
	"github.com/elastic/apm-server/model/metricset"
	"github.com/elastic/apm-server/model/span"
	"github.com/elastic/apm-server/model/transaction"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/sourcemap"
	"github.com/elastic/beats/libbeat/beat"
)

// Transformer encapsulates configuration for decoding events into model
// objects, which can later be transformed into beat.Events.
//
// The Decode methods adhere to the processor/stream.DecodeFunc signature,
// and wrap the model objects as necessary to adhere to publish.Transformable.
type Transformer struct {
	Experimental        bool
	LibraryPattern      *regexp.Regexp
	ExcludeFromGrouping *regexp.Regexp
	SourcemapStore      *sourcemap.Store
}

func (t *Transformer) DecodeTransaction(input interface{}, requestTime time.Time, metadata metadata.Metadata) (publish.Transformable, error) {
	return transaction.Decode(input, requestTime, metadata, t.Experimental)
}

func (t *Transformer) DecodeSpan(input interface{}, requestTime time.Time, metadata metadata.Metadata) (publish.Transformable, error) {
	event, err := span.Decode(input, requestTime, metadata, t.Experimental)
	if err != nil {
		return nil, err
	}
	return &transformableSpan{t, event}, nil
}

func (t *Transformer) DecodeError(input interface{}, requestTime time.Time, metadata metadata.Metadata) (publish.Transformable, error) {
	event, err := apmerror.Decode(input, requestTime, metadata, t.Experimental)
	if err != nil {
		return nil, err
	}
	return &transformableError{t, event}, nil
}

func (t *Transformer) DecodeMetricset(input interface{}, requestTime time.Time, metadata metadata.Metadata) (publish.Transformable, error) {
	return metricset.Decode(input, requestTime, metadata)
}

type transformableSpan struct {
	*Transformer
	event *span.Event
}

func (ts *transformableSpan) Transform() []beat.Event {
	return ts.event.Transform(ts.LibraryPattern, ts.ExcludeFromGrouping, ts.SourcemapStore)
}

type transformableError struct {
	*Transformer
	event *apmerror.Event
}

func (te *transformableError) Transform() []beat.Event {
	return te.event.Transform(te.LibraryPattern, te.ExcludeFromGrouping, te.SourcemapStore)
}
