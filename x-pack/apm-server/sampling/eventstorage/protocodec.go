// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eventstorage

import (
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/modelpb"
)

// JSONCodec is an implementation of Codec, using JSON encoding.
type JSONCodec struct{}

// DecodeSpan decodes data as JSON into span.
func (JSONCodec) DecodeSpan(data []byte, span *model.Span) error {
	panic("not implemented")
	//return jsoniter.ConfigFastest.Unmarshal(data, span)
}

// DecodeTransaction decodes data as JSON into tx.
func (JSONCodec) DecodeTransaction(data []byte, tx *model.Transaction) error {
	//return jsoniter.ConfigFastest.Unmarshal(data, tx)
}

// EncodeSpan encodes span as JSON.
func (JSONCodec) EncodeSpan(span *model.Span) ([]byte, error) {
	panic("not implemented")
	//return json.Marshal(span)
}

// EncodeTransaction encodes tx as JSON.
func (JSONCodec) EncodeTransaction(tx *model.Transaction) ([]byte, error) {
	pb := modelpb.Transaction{
		TraceID: tx.TraceID,
		ID:      tx.ID,
	}

	//return json.Marshal(tx)
}
