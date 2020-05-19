// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pubsub

import "context"

// TODO(axw) remove these when we implement the Elasticsearch publisher/subscriber.

// NopSampledTraceIDsSubscriber provides a no-op SubscribeSampledTraceIDs method.
type NopSampledTraceIDsSubscriber struct{}

// SubscribeSampledTraceIDs waits for ctx.Done(), and returns ctx.Err().
func (NopSampledTraceIDsSubscriber) SubscribeSampledTraceIDs(ctx context.Context, _ chan<- string) error {
	<-ctx.Done()
	return ctx.Err()
}

// NopSampledTraceIDsPublisher provides a no-op PublishSampledTraceIDs method.
type NopSampledTraceIDsPublisher struct{}

// PublishSampledTraceIDs returns ctx.Err().
func (NopSampledTraceIDsPublisher) PublishSampledTraceIDs(ctx context.Context, _ ...string) error {
	return ctx.Err()
}
