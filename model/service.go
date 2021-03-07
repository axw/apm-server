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
	"github.com/elastic/go-structform"
)

//Service bundles together information related to the monitored service and the agent used for monitoring
type Service struct {
	Name        string
	Version     string
	Environment string
	Language    Language
	Runtime     Runtime
	Framework   Framework
	Agent       Agent
	Node        ServiceNode
}

//Language has an optional version and name
type Language struct {
	Name    string
	Version string
}

//Runtime has an optional version and name
type Runtime struct {
	Name    string
	Version string
}

//Framework has an optional version and name
type Framework struct {
	Name    string
	Version string
}

//Agent has an optional version, name and an ephemeral id
type Agent struct {
	Name        string
	Version     string
	EphemeralID string
}

type ServiceNode struct {
	Name string
}

func (s *Service) Fold(v structform.ExtVisitor) error {
	v.OnObjectStart(-1, structform.AnyType)
	maybeFoldString(v, "name", s.Name)
	maybeFoldString(v, "version", s.Version)
	maybeFoldString(v, "environment", s.Environment)
	if s.Node != (ServiceNode{}) {
		v.OnKey("node")
		v.OnObjectStart(1, structform.StringType)
		maybeFoldString(v, "name", s.Node.Name)
		v.OnObjectFinished()
	}
	if s.Language != (Language{}) {
		v.OnKey("language")
		v.OnObjectStart(2, structform.StringType)
		maybeFoldString(v, "name", s.Language.Name)
		maybeFoldString(v, "version", s.Language.Version)
		v.OnObjectFinished()
	}
	if s.Runtime != (Runtime{}) {
		v.OnKey("runtime")
		v.OnObjectStart(2, structform.StringType)
		maybeFoldString(v, "name", s.Runtime.Name)
		maybeFoldString(v, "version", s.Runtime.Version)
		v.OnObjectFinished()
	}
	if s.Framework != (Framework{}) {
		v.OnKey("framework")
		v.OnObjectStart(2, structform.StringType)
		maybeFoldString(v, "name", s.Framework.Name)
		maybeFoldString(v, "version", s.Framework.Version)
		v.OnObjectFinished()
	}
	return v.OnObjectFinished()
}

func (a *Agent) Fold(v structform.ExtVisitor) error {
	v.OnObjectStart(-1, structform.StringType)
	maybeFoldString(v, "name", a.Name)
	maybeFoldString(v, "version", a.Version)
	maybeFoldString(v, "ephemeral_id", a.EphemeralID)
	return v.OnObjectFinished()
}
