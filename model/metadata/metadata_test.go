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

package metadata

import (
	"net"
	"testing"

	"github.com/elastic/apm-server/tests"
	"github.com/elastic/beats/v7/libbeat/common"
)

func BenchmarkMetadataSet(b *testing.B) {
	test := func(b *testing.B, name string, input Metadata) {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			out := make(common.MapStr)
			for i := 0; i < b.N; i++ {
				input.Set(out)
				for k := range out {
					delete(out, k)
				}
			}
		})
	}

	test(b, "minimal", Metadata{
		Service: &Service{
			Name:    tests.StringPtr("foo"),
			Version: tests.StringPtr("1.0"),
		},
	})
	test(b, "full", Metadata{
		Service: &Service{
			Name:        tests.StringPtr("foo"),
			Version:     tests.StringPtr("1.0"),
			Environment: tests.StringPtr("production"),
			Node:        ServiceNode{Name: tests.StringPtr("foo-bar")},
			Language:    Language{Name: tests.StringPtr("go"), Version: tests.StringPtr("++")},
			Runtime:     Runtime{Name: tests.StringPtr("gc"), Version: tests.StringPtr("1.0")},
			Framework:   Framework{Name: tests.StringPtr("never"), Version: tests.StringPtr("again")},
			Agent:       Agent{Name: tests.StringPtr("go"), Version: tests.StringPtr("2.0")},
		},
		Process: &Process{
			Pid:   123,
			Ppid:  tests.IntPtr(122),
			Title: tests.StringPtr("case"),
			Argv:  []string{"apm-server"},
		},
		System: &System{
			DetectedHostname:   tests.StringPtr("detected"),
			ConfiguredHostname: tests.StringPtr("configured"),
			Architecture:       tests.StringPtr("x86_64"),
			Platform:           tests.StringPtr("linux"),
			IP:                 net.ParseIP("10.1.1.1"),
			Container:          &Container{ID: "docker"},
			Kubernetes: &Kubernetes{
				Namespace: tests.StringPtr("system"),
				NodeName:  tests.StringPtr("node01"),
				PodName:   tests.StringPtr("pet"),
				PodUID:    tests.StringPtr("cattle"),
			},
		},
		User: &User{
			Id:        tests.StringPtr("123"),
			Email:     tests.StringPtr("me@example.com"),
			Name:      tests.StringPtr("bob"),
			IP:        net.ParseIP("10.1.1.2"),
			UserAgent: tests.StringPtr("user-agent"),
		},
		Labels: common.MapStr{"k": "v", "n": 1, "f": 1.5, "b": false},
	})
}
