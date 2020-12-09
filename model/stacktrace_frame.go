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
	"github.com/elastic/apm-server/utility"
)

const (
	errMsgSourcemapColumnMandatory = "Colno mandatory for sourcemapping."
	errMsgSourcemapLineMandatory   = "Lineno mandatory for sourcemapping."
	errMsgSourcemapPathMandatory   = "AbsPath mandatory for sourcemapping."
)

type StacktraceFrame struct {
	AbsPath      *string
	Filename     *string
	Classname    *string
	Lineno       *int
	Colno        *int
	ContextLine  *string
	Module       *string
	Function     *string
	LibraryFrame *bool
	Vars         common.MapStr
	PreContext   []string
	PostContext  []string
}

func (s *StacktraceFrame) transform(cfg *transform.Config, rum bool) common.MapStr {
	m := common.MapStr{}
	utility.Set(m, "library_frame", s.LibraryFrame)
	utility.Set(m, "filename", s.Filename)
	utility.Set(m, "classname", s.Classname)
	utility.Set(m, "abs_path", s.AbsPath)
	utility.Set(m, "module", s.Module)
	utility.Set(m, "function", s.Function)
	utility.Set(m, "vars", s.Vars)

	if rum && s.AbsPath != nil {
		// Set the path to match against the bundle_filepath recorded in a sourcemap.
		bundleFilepath := *s.AbsPath
		u, err := url.Parse(bundleFilepath)
		if err == nil {
			bundleFilepath = path.Clean(u.Path)
		}
		utility.Set(m, "rum.bundle_filepath", bundleFilepath)
	}

	context := common.MapStr{}
	utility.Set(context, "pre", s.PreContext)
	utility.Set(context, "post", s.PostContext)
	utility.Set(m, "context", context)

	line := common.MapStr{}
	utility.Set(line, "number", s.Lineno)
	utility.Set(line, "column", s.Colno)
	utility.Set(line, "context", s.ContextLine)
	utility.Set(m, "line", line)

	return m
}

func (s *StacktraceFrame) validForSourcemapping() (bool, string) {
	if s.Colno == nil {
		return false, errMsgSourcemapColumnMandatory
	}
	if s.Lineno == nil {
		return false, errMsgSourcemapLineMandatory
	}
	if s.AbsPath == nil {
		return false, errMsgSourcemapPathMandatory
	}
	return true, ""
}
