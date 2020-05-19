// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pubsub

import "context"

// TODO(axw) implement Elasticsearch-based sampled trace ID publish-subscribe.

// Elasticsearch provides a means of publishing and subscribing to sampled trace IDs.
type Elasticsearch struct {
}

func (e *Elasticsearch) SubscribeSampledTraceIDs(ctx context.Context, traceIDs chan<- string) error {
	panic("not implemented")
}

func (e *Elasticsearch) PublishSampledTraceIDs(ctx context.Context, traceID ...string) error {
	panic("not implemented")
}
