package model

import (
	"github.com/elastic/apm-server/model/modelpb"
)

type keyValueList struct {
	pb     modelpb.KeyValueList
	values []KeyValue
}

func (l *keyValueList) set(values ...KeyValue) {
	l.pb.Values = l.pb.Values[:0]
	l.values = l.values[:0]
	for _, value := range values {
		l.values = append(l.values, value)
	}
	for i := range l.values {
		l.pb.Values[i] = &l.values[i].pb
	}
}

// KeyValue holds a key-value pair for use in fields which accept a mapping
// with arbitrary values, such as transaction.context.custom.
type KeyValue struct {
	pb    modelpb.KeyValue
	value Any
}

// NewKeyValue returns a new KeyValue with key and value.
func NewKeyValue(key string, value Any) KeyValue {
	kv := KeyValue{value: value}
	kv.pb.Key = key
	kv.pb.Value = &kv.value.pb
	return kv
}

// Label holds a key-value pair suitable for use as a label.
type Label struct {
	pb       modelpb.KeyValue
	value    modelpb.AnyValue
	pbString modelpb.AnyValue_StringValue
	pbBool   modelpb.AnyValue_BoolValue
	pbInt    modelpb.AnyValue_IntValue
	pbDouble modelpb.AnyValue_DoubleValue
}

func newNullLabel(k string) Label {
	var l Label
	l.pb.Key = k
	l.pb.Value = &l.value
	return l
}

func NewStringLabel(k, v string) Label {
	l := newNullLabel(k)
	l.pbString.StringValue = v
	l.value.Value = &l.pbString
	return l
}

func NewBoolLabel(k string, v bool) Label {
	l := newNullLabel(k)
	l.pbBool.BoolValue = v
	l.value.Value = &l.pbBool
	return l
}

func NewInt64Label(k string, v int64) Label {
	l := newNullLabel(k)
	l.pbInt.IntValue = v
	l.value.Value = &l.pbInt
	return l
}

func NewFloat64Label(k string, v float64) Label {
	l := newNullLabel(k)
	l.pbDouble.DoubleValue = v
	l.value.Value = &l.pbDouble
	return l
}

type anyValueList struct {
	pb     modelpb.AnyValueList
	values []Any
}

func (l *anyValueList) set(values ...Any) {
	l.pb.Values = l.pb.Values[:0]
	l.values = l.values[:0]
	for _, value := range values {
		l.values = append(l.values, value)
	}
	for i := range l.values {
		l.pb.Values[i] = &l.values[i].pb
	}
}

// Any holds a value supported by "custom". The zero value
// represents a null value.
type Any struct {
	pb             modelpb.AnyValue
	pbString       modelpb.AnyValue_StringValue
	pbBool         modelpb.AnyValue_BoolValue
	pbInt          modelpb.AnyValue_IntValue
	pbDouble       modelpb.AnyValue_DoubleValue
	pbArray        modelpb.AnyValue_ArrayValue
	pbKeyValueList modelpb.AnyValue_KvlistValue

	anyValues anyValueList
	keyValues keyValueList
}

// SetString sets a to a string value.
func (a *Any) SetString(v string) {
	a.pbString.StringValue = v
	a.pb.Value = &a.pbString
}

// SetBool sets a to a boolean value.
func (a *Any) SetBool(v bool) {
	a.pbBool.BoolValue = v
	a.pb.Value = &a.pbBool
}

// SetInt64 sets a to an int64 value.
func (a *Any) SetInt64(v int64) {
	a.pbInt.IntValue = v
	a.pb.Value = &a.pbInt
}

// SetFloat64 sets a to a float64 value.
func (a *Any) SetFloat64(v float64) {
	a.pbDouble.DoubleValue = v
	a.pb.Value = &a.pbDouble
}

// SetArray sets a to an array of values.
func (a *Any) SetArray(values ...Any) {
	a.anyValues.set(values...)
	a.pbArray.ArrayValue = &a.anyValues.pb
	a.pb.Value = &a.pbKeyValueList
}

// SetKeyValues sets a to a key-value list -- a map of strings to Any.
func (a *Any) SetKeyValues(keyValues ...KeyValue) {
	a.keyValues.set(keyValues...)
	a.pbKeyValueList.KvlistValue = &a.keyValues.pb
	a.pb.Value = &a.pbKeyValueList
}
