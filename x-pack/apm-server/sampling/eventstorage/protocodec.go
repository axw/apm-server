// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eventstorage

import (
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/modelpb"
	"google.golang.org/protobuf/proto"
)

// ProtobufCodec is an implementation of Codec, using protobuf encoding.
type ProtobufCodec struct{}

// DecodeSpan decodes data as protobuf into span.
func (ProtobufCodec) DecodeSpan(data []byte, span *model.Span) error {
	panic("not implemented")
	//return jsoniter.ConfigFastest.Unmarshal(data, span)
}

// DecodeTransaction decodes data as protobuf into tx.
func (ProtobufCodec) DecodeTransaction(data []byte, tx *model.Transaction) error {
	var pb modelpb.Transaction
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	tx.ID = pb.ID
	tx.TraceID = pb.TraceID
	return nil
}

// EncodeSpan encodes span as protobuf.
func (ProtobufCodec) EncodeSpan(span *model.Span) ([]byte, error) {
	panic("not implemented")
}

// EncodeTransaction encodes tx as protobuf.
func (ProtobufCodec) EncodeTransaction(tx *model.Transaction) ([]byte, error) {
	pb := modelpb.Transaction{
		TraceID: tx.TraceID,
		ID:      tx.ID,
	}
	return proto.Marshal(&pb)
}
