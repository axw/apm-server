// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"errors"
	"time"

	"github.com/elastic/go-elasticsearch/v7"

	"github.com/elastic/apm-server/publish"
)

// Config holds configuration for Processor.
type Config struct {
	// General config

	BeatID        string
	Reporter      publish.Reporter
	FlushInterval time.Duration

	// Pubsub config

	Elasticsearch      *elasticsearch.Client
	SampledTracesIndex string

	// Reservoir sampling config

	MaxTraceGroups        int
	DefaultSampleRate     float64
	IngestRateCoefficient float64

	// Storage config

	StorageDir        string
	StorageGCInterval time.Duration
	TTL               time.Duration
}

func (config Config) Validate() error {
	if config.BeatID == "" {
		return errors.New("BeatID unspecified")
	}
	if config.Reporter == nil {
		return errors.New("Reporter unspecified")
	}
	if config.Elasticsearch == nil {
		return errors.New("Elasticsearch unspecified")
	}
	if config.SampledTracesIndex == "" {
		return errors.New("SampledTracesIndex unspecified")
	}
	if config.StorageDir == "" {
		return errors.New("StorageDir unspecified")
	}
	if config.StorageGCInterval <= 0 {
		return errors.New("GCInterval unspecified or negative")
	}
	if config.TTL <= 0 {
		return errors.New("TTL unspecified or negative")
	}
	if config.MaxTraceGroups <= 0 {
		return errors.New("MaxTraceGroups unspecified or negative")
	}
	if config.FlushInterval <= 0 {
		return errors.New("FlushInterval unspecified or negative")
	}
	if config.DefaultSampleRate <= 0 || config.DefaultSampleRate >= 1 {
		return errors.New("DefaultSampleRate unspecified, or out of range (0,1)")
	}
	if config.IngestRateCoefficient <= 0 || config.IngestRateCoefficient > 1 {
		return errors.New("IngestRateCoefficient unspecified, or out of range (0,1]")
	}
	return nil
}
