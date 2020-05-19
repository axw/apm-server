// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"fmt"
	"time"

	"github.com/elastic/apm-server/publish"
)

// Config holds configuration for Processor.
type Config struct {
	Reporter              publish.Reporter
	StorageDir            string
	StorageGCInterval     time.Duration
	TTL                   time.Duration
	MaxTraceGroups        int
	FlushInterval         time.Duration
	DefaultSampleRate     float64
	IngestRateCoefficient float64
}

func (config Config) Validate() error {
	if config.Reporter == nil {
		return fmt.Errorf("Reporter unspecified")
	}
	if config.StorageDir == "" {
		return fmt.Errorf("StorageDir unspecified")
	}
	if config.StorageGCInterval <= 0 {
		return fmt.Errorf("GCInterval unspecified or negative")
	}
	if config.TTL <= 0 {
		return fmt.Errorf("TTL unspecified or negative")
	}
	if config.MaxTraceGroups <= 0 {
		return fmt.Errorf("MaxTraceGroups unspecified or negative")
	}
	if config.FlushInterval <= 0 {
		return fmt.Errorf("FlushInterval unspecified or negative")
	}
	if config.DefaultSampleRate <= 0 || config.DefaultSampleRate >= 1 {
		return fmt.Errorf("DefaultSampleRate unspecified, or out of range (0,1)")
	}
	if config.IngestRateCoefficient <= 0 || config.IngestRateCoefficient > 1 {
		return fmt.Errorf("IngestRateCoefficient unspecified, or out of range (0,1]")
	}
	return nil
}
