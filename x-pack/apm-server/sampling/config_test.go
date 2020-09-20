// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling"
	"github.com/elastic/go-elasticsearch/v7"
)

func TestNewProcessorConfigInvalid(t *testing.T) {
	var config sampling.Config
	assertInvalidConfigError := func(expectedError string) {
		t.Helper()
		agg, err := sampling.NewProcessor(config)
		require.Error(t, err)
		require.Nil(t, agg)
		assert.EqualError(t, err, "invalid tail-sampling config: "+expectedError)
	}
	assertInvalidConfigError("BeatID unspecified")
	config.BeatID = "beat"

	assertInvalidConfigError("Reporter unspecified")
	config.Reporter = func(ctx context.Context, req publish.PendingReq) error { return nil }

	assertInvalidConfigError("FlushInterval unspecified or negative")
	config.FlushInterval = 1

	assertInvalidConfigError("Elasticsearch unspecified")
	config.Elasticsearch = &elasticsearch.Client{}

	assertInvalidConfigError("SampledTracesIndex unspecified")
	config.SampledTracesIndex = "sampled-traces"

	assertInvalidConfigError("MaxTraceGroups unspecified or negative")
	config.MaxTraceGroups = 1

	for _, invalid := range []float64{-1, 1.0, 2.0} {
		config.DefaultSampleRate = invalid
		assertInvalidConfigError("DefaultSampleRate unspecified or out of range [0,1)")
	}
	config.DefaultSampleRate = 0.5

	for _, invalid := range []float64{-1, 0, 2.0} {
		config.IngestRateCoefficient = invalid
		assertInvalidConfigError("IngestRateCoefficient unspecified or out of range (0,1]")
	}
	config.IngestRateCoefficient = 0.5

	assertInvalidConfigError("StorageDir unspecified")
	config.StorageDir = "tbs"

	assertInvalidConfigError("StorageGCInterval unspecified or negative")
	config.StorageGCInterval = 1

	assertInvalidConfigError("TTL unspecified or negative")
	config.IngestRateCoefficient = 1
}
