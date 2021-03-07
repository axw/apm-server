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

package model

import (
	"net"

	"github.com/elastic/go-structform"
)

type System struct {
	// DetectedHostname holds the detected hostname.
	//
	// This will be written to the event as "host.hostname".
	//
	// TODO(axw) rename this to Hostname.
	DetectedHostname string

	// ConfiguredHostname holds the user-defined or detected hostname.
	//
	// If defined, this will be written to the event as "host.name".
	//
	// TODO(axw) rename this to Name.
	ConfiguredHostname string

	Architecture string
	Platform     string
	IP           net.IP

	Container  Container
	Kubernetes Kubernetes
}

func (s *System) Fold(v structform.ExtVisitor) error {
	v.OnObjectStart(-1, structform.AnyType)
	maybeFoldString(v, "hostname", s.DetectedHostname)
	maybeFoldString(v, "name", s.ConfiguredHostname)
	maybeFoldString(v, "architecture", s.Architecture)
	if s.Platform != "" {
		v.OnKey("os")
		v.OnObjectStart(1, structform.StringType)
		v.OnKey("platform")
		v.OnString(s.Platform)
		v.OnObjectFinished()
	}
	if s.IP != nil {
		v.OnKey("ip")
		v.OnString(s.IP.String())
	}
	return v.OnObjectFinished()
}
