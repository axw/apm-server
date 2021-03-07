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
	"time"

	"github.com/elastic/go-structform"
	"github.com/elastic/go-structform/gotype"
)

func millisAsMicros(ms float64) gotype.Folder {
	return microsFolder(int64(ms * 1000))
}

func timeAsMicros(t time.Time) gotype.Folder {
	if t.IsZero() {
		return nil
	}
	return microsFolder(t.UnixNano() / 1000)
}

type microsFolder int64

func (f microsFolder) Fold(v structform.ExtVisitor) error {
	v.OnObjectStart(1, structform.Int64Type)
	v.OnKey("us")
	v.OnInt64(int64(f))
	return v.OnObjectFinished()
}
