// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package config

import "time"

// SamplingConfig holds configuration related to sampling.
type SamplingConfig struct {
	// KeepUnsampled controls whether unsampled
	// transactions should be recorded.
	KeepUnsampled bool `config:"keep_unsampled"`

	// Tail holds tail-sampling configuration.
	Tail *TailSamplingConfig `config:"tail"`
}

// TailSamplingConfig holds configuration related to tail-sampling.
type TailSamplingConfig struct {
	Enabled bool `config:"enabled"`

	DefaultSampleRate     float64       `config:"default_sample_rate" validate:"min=0, max=1"`
	Interval              time.Duration `config:"interval" validate:"min=1"`
	IngestRateCoefficient float64       `config:"ingest_rate_coefficient" validate:"min=0, max=1"`
	StorageDir            string        `config:"storage_dir"`
	StorageGCInterval     time.Duration `config:"storage_gc_interval" validate:"min=1"`
	TTL                   time.Duration `config:"ttl" validate:"min=1"`
}

func defaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		// In a future major release we will set this to
		// false, and then later remove the option.
		KeepUnsampled: true,

		Tail: &TailSamplingConfig{
			Enabled:               false,
			DefaultSampleRate:     0.5,
			Interval:              1 * time.Minute,
			IngestRateCoefficient: 0.25,
			StorageDir:            "tail_sampling",
			StorageGCInterval:     5 * time.Minute,
			TTL:                   30 * time.Minute,
		},
	}
}
