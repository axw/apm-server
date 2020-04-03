package metadata

import "github.com/elastic/beats/v7/libbeat/common"

type mapStr common.MapStr

func (m *mapStr) set(k string, v interface{}) {
	if *m == nil {
		*m = make(mapStr)
	}
	(*m)[k] = v
}

func (m *mapStr) maybeSetString(k, v string) bool {
	if v != "" {
		m.set(k, v)
		return true
	}
	return false
}
