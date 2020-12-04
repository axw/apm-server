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
	"net/url"
	"path"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/elastic/apm-server/transform"
)

const (
	errMsgSourcemapColumnMandatory = "Colno mandatory for sourcemapping."
	errMsgSourcemapLineMandatory   = "Lineno mandatory for sourcemapping."
	errMsgSourcemapPathMandatory   = "AbsPath mandatory for sourcemapping."
)

type StacktraceFrame struct {
	AbsPath      string
	Filename     string
	Classname    string
	Lineno       *int
	Colno        *int
	ContextLine  string
	Module       string
	Function     string
	LibraryFrame *bool
	Vars         common.MapStr
	PreContext   []string
	PostContext  []string
}

func (s *StacktraceFrame) transform(cfg *transform.Config, rum bool) common.MapStr {
	var m mapStr
	m.maybeSetBool("library_frame", s.LibraryFrame)
	m.maybeSetString("filename", s.Filename)
	m.maybeSetString("classname", s.Classname)
	m.maybeSetString("abs_path", s.AbsPath)
	m.maybeSetString("module", s.Module)
	m.maybeSetString("function", s.Function)
	m.maybeSetMapStr("vars", s.Vars)

	if rum && s.AbsPath != "" {
		// Set the path to match against the bundle_filepath recorded in a sourcemap.
		bundleFilepath := s.AbsPath
		if u, err := url.Parse(bundleFilepath); err == nil {
			bundleFilepath = path.Clean(u.Path)
		}
		m.set("rum.bundle_filepath", bundleFilepath)
	}

	var context mapStr
	if len(s.PreContext) > 0 {
		context.set("pre", s.PreContext)
	}
	if len(s.PostContext) > 0 {
		context.set("post", s.PostContext)
	}
	m.maybeSetMapStr("context", common.MapStr(context))

	var line mapStr
	line.maybeSetIntptr("number", s.Lineno)
	line.maybeSetIntptr("column", s.Colno)
	line.maybeSetString("context", s.ContextLine)
	m.maybeSetMapStr("line", common.MapStr(line))

	return common.MapStr(m)
}

func (s *StacktraceFrame) validForSourcemapping() (bool, string) {
	if s.Colno == nil {
		return false, errMsgSourcemapColumnMandatory
	}
	if s.Lineno == nil {
		return false, errMsgSourcemapLineMandatory
	}
	if s.AbsPath == "" {
		return false, errMsgSourcemapPathMandatory
	}
	return true, ""
}
