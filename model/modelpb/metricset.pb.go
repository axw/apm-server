// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: metricset.proto

package modelpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Metricset struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name         string    `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	TimeseriesId string    `protobuf:"bytes,2,opt,name=timeseries_id,json=timeseriesId,proto3" json:"timeseries_id,omitempty"`
	Metrics      []*Metric `protobuf:"bytes,3,rep,name=metrics,proto3" json:"metrics,omitempty"`
}

func (x *Metricset) Reset() {
	*x = Metricset{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricset_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metricset) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metricset) ProtoMessage() {}

func (x *Metricset) ProtoReflect() protoreflect.Message {
	mi := &file_metricset_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metricset.ProtoReflect.Descriptor instead.
func (*Metricset) Descriptor() ([]byte, []int) {
	return file_metricset_proto_rawDescGZIP(), []int{0}
}

func (x *Metricset) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Metricset) GetTimeseriesId() string {
	if x != nil {
		return x.TimeseriesId
	}
	return ""
}

func (x *Metricset) GetMetrics() []*Metric {
	if x != nil {
		return x.Metrics
	}
	return nil
}

type Metric struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Unit string `protobuf:"bytes,3,opt,name=unit,proto3" json:"unit,omitempty"`
	// Types that are assignable to Value:
	//	*Metric_Double
	//	*Metric_Histogram
	Value isMetric_Value `protobuf_oneof:"value"`
}

func (x *Metric) Reset() {
	*x = Metric{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricset_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metric) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metric) ProtoMessage() {}

func (x *Metric) ProtoReflect() protoreflect.Message {
	mi := &file_metricset_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metric.ProtoReflect.Descriptor instead.
func (*Metric) Descriptor() ([]byte, []int) {
	return file_metricset_proto_rawDescGZIP(), []int{1}
}

func (x *Metric) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Metric) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Metric) GetUnit() string {
	if x != nil {
		return x.Unit
	}
	return ""
}

func (m *Metric) GetValue() isMetric_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *Metric) GetDouble() float64 {
	if x, ok := x.GetValue().(*Metric_Double); ok {
		return x.Double
	}
	return 0
}

func (x *Metric) GetHistogram() *MetricHistogram {
	if x, ok := x.GetValue().(*Metric_Histogram); ok {
		return x.Histogram
	}
	return nil
}

type isMetric_Value interface {
	isMetric_Value()
}

type Metric_Double struct {
	Double float64 `protobuf:"fixed64,4,opt,name=double,proto3,oneof"`
}

type Metric_Histogram struct {
	Histogram *MetricHistogram `protobuf:"bytes,5,opt,name=histogram,proto3,oneof"`
}

func (*Metric_Double) isMetric_Value() {}

func (*Metric_Histogram) isMetric_Value() {}

type MetricHistogram struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Values []float64 `protobuf:"fixed64,1,rep,packed,name=values,proto3" json:"values,omitempty"`
	Counts []float64 `protobuf:"fixed64,2,rep,packed,name=counts,proto3" json:"counts,omitempty"`
}

func (x *MetricHistogram) Reset() {
	*x = MetricHistogram{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricset_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetricHistogram) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetricHistogram) ProtoMessage() {}

func (x *MetricHistogram) ProtoReflect() protoreflect.Message {
	mi := &file_metricset_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetricHistogram.ProtoReflect.Descriptor instead.
func (*MetricHistogram) Descriptor() ([]byte, []int) {
	return file_metricset_proto_rawDescGZIP(), []int{2}
}

func (x *MetricHistogram) GetValues() []float64 {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *MetricHistogram) GetCounts() []float64 {
	if x != nil {
		return x.Counts
	}
	return nil
}

var File_metricset_proto protoreflect.FileDescriptor

var file_metricset_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x65, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x11, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x22, 0x79, 0x0a, 0x09, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x65,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x65, 0x72,
	0x69, 0x65, 0x73, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x65, 0x72, 0x69, 0x65, 0x73, 0x49, 0x64, 0x12, 0x33, 0x0a, 0x07, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x65, 0x6c,
	0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e,
	0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x52, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x22,
	0xab, 0x01, 0x0a, 0x06, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x6e, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x75, 0x6e, 0x69, 0x74, 0x12, 0x18, 0x0a, 0x06, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x48, 0x00, 0x52, 0x06, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65,
	0x12, 0x42, 0x0a, 0x09, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x67, 0x72, 0x61, 0x6d, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70,
	0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x48, 0x69,
	0x73, 0x74, 0x6f, 0x67, 0x72, 0x61, 0x6d, 0x48, 0x00, 0x52, 0x09, 0x68, 0x69, 0x73, 0x74, 0x6f,
	0x67, 0x72, 0x61, 0x6d, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x41, 0x0a,
	0x0f, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x48, 0x69, 0x73, 0x74, 0x6f, 0x67, 0x72, 0x61, 0x6d,
	0x12, 0x16, 0x0a, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x01,
	0x52, 0x06, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x01, 0x52, 0x06, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73,
	0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65,
	0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2f, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_metricset_proto_rawDescOnce sync.Once
	file_metricset_proto_rawDescData = file_metricset_proto_rawDesc
)

func file_metricset_proto_rawDescGZIP() []byte {
	file_metricset_proto_rawDescOnce.Do(func() {
		file_metricset_proto_rawDescData = protoimpl.X.CompressGZIP(file_metricset_proto_rawDescData)
	})
	return file_metricset_proto_rawDescData
}

var file_metricset_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_metricset_proto_goTypes = []interface{}{
	(*Metricset)(nil),       // 0: elastic.apm.model.Metricset
	(*Metric)(nil),          // 1: elastic.apm.model.Metric
	(*MetricHistogram)(nil), // 2: elastic.apm.model.MetricHistogram
}
var file_metricset_proto_depIdxs = []int32{
	1, // 0: elastic.apm.model.Metricset.metrics:type_name -> elastic.apm.model.Metric
	2, // 1: elastic.apm.model.Metric.histogram:type_name -> elastic.apm.model.MetricHistogram
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_metricset_proto_init() }
func file_metricset_proto_init() {
	if File_metricset_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metricset_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metricset); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricset_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metric); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricset_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetricHistogram); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_metricset_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Metric_Double)(nil),
		(*Metric_Histogram)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_metricset_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_metricset_proto_goTypes,
		DependencyIndexes: file_metricset_proto_depIdxs,
		MessageInfos:      file_metricset_proto_msgTypes,
	}.Build()
	File_metricset_proto = out.File
	file_metricset_proto_rawDesc = nil
	file_metricset_proto_goTypes = nil
	file_metricset_proto_depIdxs = nil
}
