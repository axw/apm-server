// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: transaction.proto

package modelpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Transaction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                  string                       `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type                string                       `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Name                string                       `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Result              string                       `protobuf:"bytes,4,opt,name=result,proto3" json:"result,omitempty"`
	Sampled             *bool                        `protobuf:"varint,5,opt,name=sampled,proto3,oneof" json:"sampled,omitempty"`
	RepresentativeCount float64                      `protobuf:"fixed64,6,opt,name=representative_count,json=representativeCount,proto3" json:"representative_count,omitempty"`
	SpanCount           *TransactionSpanCount        `protobuf:"bytes,7,opt,name=span_count,json=spanCount,proto3" json:"span_count,omitempty"`
	Custom              *structpb.Struct             `protobuf:"bytes,8,opt,name=custom,proto3" json:"custom,omitempty"`
	Experimental        *structpb.Value              `protobuf:"bytes,9,opt,name=experimental,proto3" json:"experimental,omitempty"`
	Experience          *TransactionUserExperience   `protobuf:"bytes,10,opt,name=experience,proto3" json:"experience,omitempty"`
	Marks               map[string]*TransactionMarks `protobuf:"bytes,11,rep,name=marks,proto3" json:"marks,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Message             *Message                     `protobuf:"bytes,12,opt,name=message,proto3" json:"message,omitempty"`
	Root                bool                         `protobuf:"varint,13,opt,name=root,proto3" json:"root,omitempty"` // only populated by metricset
}

func (x *Transaction) Reset() {
	*x = Transaction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_transaction_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Transaction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Transaction) ProtoMessage() {}

func (x *Transaction) ProtoReflect() protoreflect.Message {
	mi := &file_transaction_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Transaction.ProtoReflect.Descriptor instead.
func (*Transaction) Descriptor() ([]byte, []int) {
	return file_transaction_proto_rawDescGZIP(), []int{0}
}

func (x *Transaction) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Transaction) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Transaction) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Transaction) GetResult() string {
	if x != nil {
		return x.Result
	}
	return ""
}

func (x *Transaction) GetSampled() bool {
	if x != nil && x.Sampled != nil {
		return *x.Sampled
	}
	return false
}

func (x *Transaction) GetRepresentativeCount() float64 {
	if x != nil {
		return x.RepresentativeCount
	}
	return 0
}

func (x *Transaction) GetSpanCount() *TransactionSpanCount {
	if x != nil {
		return x.SpanCount
	}
	return nil
}

func (x *Transaction) GetCustom() *structpb.Struct {
	if x != nil {
		return x.Custom
	}
	return nil
}

func (x *Transaction) GetExperimental() *structpb.Value {
	if x != nil {
		return x.Experimental
	}
	return nil
}

func (x *Transaction) GetExperience() *TransactionUserExperience {
	if x != nil {
		return x.Experience
	}
	return nil
}

func (x *Transaction) GetMarks() map[string]*TransactionMarks {
	if x != nil {
		return x.Marks
	}
	return nil
}

func (x *Transaction) GetMessage() *Message {
	if x != nil {
		return x.Message
	}
	return nil
}

func (x *Transaction) GetRoot() bool {
	if x != nil {
		return x.Root
	}
	return false
}

type TransactionSpanCount struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dropped int32 `protobuf:"varint,1,opt,name=dropped,proto3" json:"dropped,omitempty"`
	Started int32 `protobuf:"varint,2,opt,name=started,proto3" json:"started,omitempty"`
}

func (x *TransactionSpanCount) Reset() {
	*x = TransactionSpanCount{}
	if protoimpl.UnsafeEnabled {
		mi := &file_transaction_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionSpanCount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionSpanCount) ProtoMessage() {}

func (x *TransactionSpanCount) ProtoReflect() protoreflect.Message {
	mi := &file_transaction_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionSpanCount.ProtoReflect.Descriptor instead.
func (*TransactionSpanCount) Descriptor() ([]byte, []int) {
	return file_transaction_proto_rawDescGZIP(), []int{1}
}

func (x *TransactionSpanCount) GetDropped() int32 {
	if x != nil {
		return x.Dropped
	}
	return 0
}

func (x *TransactionSpanCount) GetStarted() int32 {
	if x != nil {
		return x.Started
	}
	return 0
}

type TransactionMarks struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Marks []*TransactionMark `protobuf:"bytes,1,rep,name=marks,proto3" json:"marks,omitempty"`
}

func (x *TransactionMarks) Reset() {
	*x = TransactionMarks{}
	if protoimpl.UnsafeEnabled {
		mi := &file_transaction_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionMarks) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionMarks) ProtoMessage() {}

func (x *TransactionMarks) ProtoReflect() protoreflect.Message {
	mi := &file_transaction_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionMarks.ProtoReflect.Descriptor instead.
func (*TransactionMarks) Descriptor() ([]byte, []int) {
	return file_transaction_proto_rawDescGZIP(), []int{2}
}

func (x *TransactionMarks) GetMarks() []*TransactionMark {
	if x != nil {
		return x.Marks
	}
	return nil
}

type TransactionMark struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string  `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value float64 `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *TransactionMark) Reset() {
	*x = TransactionMark{}
	if protoimpl.UnsafeEnabled {
		mi := &file_transaction_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionMark) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionMark) ProtoMessage() {}

func (x *TransactionMark) ProtoReflect() protoreflect.Message {
	mi := &file_transaction_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionMark.ProtoReflect.Descriptor instead.
func (*TransactionMark) Descriptor() ([]byte, []int) {
	return file_transaction_proto_rawDescGZIP(), []int{3}
}

func (x *TransactionMark) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *TransactionMark) GetValue() float64 {
	if x != nil {
		return x.Value
	}
	return 0
}

type TransactionUserExperience struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CumulativeLayoutShift float64 `protobuf:"fixed64,1,opt,name=CumulativeLayoutShift,proto3" json:"CumulativeLayoutShift,omitempty"`
	FirstInputDelay       float64 `protobuf:"fixed64,2,opt,name=FirstInputDelay,proto3" json:"FirstInputDelay,omitempty"`
	Longtask              float64 `protobuf:"fixed64,3,opt,name=Longtask,proto3" json:"Longtask,omitempty"`
	TotalBlockingTime     float64 `protobuf:"fixed64,4,opt,name=TotalBlockingTime,proto3" json:"TotalBlockingTime,omitempty"`
}

func (x *TransactionUserExperience) Reset() {
	*x = TransactionUserExperience{}
	if protoimpl.UnsafeEnabled {
		mi := &file_transaction_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionUserExperience) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionUserExperience) ProtoMessage() {}

func (x *TransactionUserExperience) ProtoReflect() protoreflect.Message {
	mi := &file_transaction_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionUserExperience.ProtoReflect.Descriptor instead.
func (*TransactionUserExperience) Descriptor() ([]byte, []int) {
	return file_transaction_proto_rawDescGZIP(), []int{4}
}

func (x *TransactionUserExperience) GetCumulativeLayoutShift() float64 {
	if x != nil {
		return x.CumulativeLayoutShift
	}
	return 0
}

func (x *TransactionUserExperience) GetFirstInputDelay() float64 {
	if x != nil {
		return x.FirstInputDelay
	}
	return 0
}

func (x *TransactionUserExperience) GetLongtask() float64 {
	if x != nil {
		return x.Longtask
	}
	return 0
}

func (x *TransactionUserExperience) GetTotalBlockingTime() float64 {
	if x != nil {
		return x.TotalBlockingTime
	}
	return 0
}

var File_transaction_proto protoreflect.FileDescriptor

var file_transaction_proto_rawDesc = []byte{
	0x0a, 0x11, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x11, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d,
	0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x1a, 0x0d, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0xa8, 0x05, 0x0a, 0x0b, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x72,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x12, 0x1d, 0x0a, 0x07, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x64, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x07, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x64, 0x88,
	0x01, 0x01, 0x12, 0x31, 0x0a, 0x14, 0x72, 0x65, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x74, 0x61,
	0x74, 0x69, 0x76, 0x65, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x01,
	0x52, 0x13, 0x72, 0x65, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x74, 0x61, 0x74, 0x69, 0x76, 0x65,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x46, 0x0a, 0x0a, 0x73, 0x70, 0x61, 0x6e, 0x5f, 0x63, 0x6f,
	0x75, 0x6e, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x65, 0x6c, 0x61, 0x73,
	0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x52, 0x09, 0x73, 0x70, 0x61, 0x6e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x2f, 0x0a,
	0x06, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x06, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x12, 0x3a,
	0x0a, 0x0c, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x6d, 0x65, 0x6e, 0x74, 0x61, 0x6c, 0x18, 0x09,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0c, 0x65, 0x78,
	0x70, 0x65, 0x72, 0x69, 0x6d, 0x65, 0x6e, 0x74, 0x61, 0x6c, 0x12, 0x4c, 0x0a, 0x0a, 0x65, 0x78,
	0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c,
	0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x55, 0x73,
	0x65, 0x72, 0x45, 0x78, 0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x52, 0x0a, 0x65, 0x78,
	0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x3f, 0x0a, 0x05, 0x6d, 0x61, 0x72, 0x6b,
	0x73, 0x18, 0x0b, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69,
	0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x4d, 0x61, 0x72, 0x6b, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x05, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x12, 0x34, 0x0a, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x65, 0x6c, 0x61,
	0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6f, 0x74, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x72,
	0x6f, 0x6f, 0x74, 0x1a, 0x5d, 0x0a, 0x0a, 0x4d, 0x61, 0x72, 0x6b, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x39, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x23, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d,
	0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x4d, 0x61, 0x72, 0x6b, 0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0x0a, 0x0a, 0x08, 0x5f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x64, 0x22, 0x4a,
	0x0a, 0x14, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x61,
	0x6e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64,
	0x12, 0x18, 0x0a, 0x07, 0x73, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x07, 0x73, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x22, 0x4c, 0x0a, 0x10, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x4d, 0x61, 0x72, 0x6b, 0x73, 0x12, 0x38,
	0x0a, 0x05, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e,
	0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x4d, 0x61, 0x72,
	0x6b, 0x52, 0x05, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x22, 0x39, 0x0a, 0x0f, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x4d, 0x61, 0x72, 0x6b, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x01, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x22, 0xc5, 0x01, 0x0a, 0x19, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x55, 0x73, 0x65, 0x72, 0x45, 0x78, 0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63,
	0x65, 0x12, 0x34, 0x0a, 0x15, 0x43, 0x75, 0x6d, 0x75, 0x6c, 0x61, 0x74, 0x69, 0x76, 0x65, 0x4c,
	0x61, 0x79, 0x6f, 0x75, 0x74, 0x53, 0x68, 0x69, 0x66, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x01,
	0x52, 0x15, 0x43, 0x75, 0x6d, 0x75, 0x6c, 0x61, 0x74, 0x69, 0x76, 0x65, 0x4c, 0x61, 0x79, 0x6f,
	0x75, 0x74, 0x53, 0x68, 0x69, 0x66, 0x74, 0x12, 0x28, 0x0a, 0x0f, 0x46, 0x69, 0x72, 0x73, 0x74,
	0x49, 0x6e, 0x70, 0x75, 0x74, 0x44, 0x65, 0x6c, 0x61, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x01,
	0x52, 0x0f, 0x46, 0x69, 0x72, 0x73, 0x74, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x44, 0x65, 0x6c, 0x61,
	0x79, 0x12, 0x1a, 0x0a, 0x08, 0x4c, 0x6f, 0x6e, 0x67, 0x74, 0x61, 0x73, 0x6b, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x01, 0x52, 0x08, 0x4c, 0x6f, 0x6e, 0x67, 0x74, 0x61, 0x73, 0x6b, 0x12, 0x2c, 0x0a,
	0x11, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x69, 0x6e, 0x67, 0x54, 0x69,
	0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x52, 0x11, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x42,
	0x6c, 0x6f, 0x63, 0x6b, 0x69, 0x6e, 0x67, 0x54, 0x69, 0x6d, 0x65, 0x42, 0x2d, 0x5a, 0x2b, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69,
	0x63, 0x2f, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_transaction_proto_rawDescOnce sync.Once
	file_transaction_proto_rawDescData = file_transaction_proto_rawDesc
)

func file_transaction_proto_rawDescGZIP() []byte {
	file_transaction_proto_rawDescOnce.Do(func() {
		file_transaction_proto_rawDescData = protoimpl.X.CompressGZIP(file_transaction_proto_rawDescData)
	})
	return file_transaction_proto_rawDescData
}

var file_transaction_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_transaction_proto_goTypes = []interface{}{
	(*Transaction)(nil),               // 0: elastic.apm.model.Transaction
	(*TransactionSpanCount)(nil),      // 1: elastic.apm.model.TransactionSpanCount
	(*TransactionMarks)(nil),          // 2: elastic.apm.model.TransactionMarks
	(*TransactionMark)(nil),           // 3: elastic.apm.model.TransactionMark
	(*TransactionUserExperience)(nil), // 4: elastic.apm.model.TransactionUserExperience
	nil,                               // 5: elastic.apm.model.Transaction.MarksEntry
	(*structpb.Struct)(nil),           // 6: google.protobuf.Struct
	(*structpb.Value)(nil),            // 7: google.protobuf.Value
	(*Message)(nil),                   // 8: elastic.apm.model.Message
}
var file_transaction_proto_depIdxs = []int32{
	1, // 0: elastic.apm.model.Transaction.span_count:type_name -> elastic.apm.model.TransactionSpanCount
	6, // 1: elastic.apm.model.Transaction.custom:type_name -> google.protobuf.Struct
	7, // 2: elastic.apm.model.Transaction.experimental:type_name -> google.protobuf.Value
	4, // 3: elastic.apm.model.Transaction.experience:type_name -> elastic.apm.model.TransactionUserExperience
	5, // 4: elastic.apm.model.Transaction.marks:type_name -> elastic.apm.model.Transaction.MarksEntry
	8, // 5: elastic.apm.model.Transaction.message:type_name -> elastic.apm.model.Message
	3, // 6: elastic.apm.model.TransactionMarks.marks:type_name -> elastic.apm.model.TransactionMark
	2, // 7: elastic.apm.model.Transaction.MarksEntry.value:type_name -> elastic.apm.model.TransactionMarks
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_transaction_proto_init() }
func file_transaction_proto_init() {
	if File_transaction_proto != nil {
		return
	}
	file_message_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_transaction_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Transaction); i {
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
		file_transaction_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionSpanCount); i {
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
		file_transaction_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionMarks); i {
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
		file_transaction_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionMark); i {
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
		file_transaction_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionUserExperience); i {
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
	file_transaction_proto_msgTypes[0].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_transaction_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_transaction_proto_goTypes,
		DependencyIndexes: file_transaction_proto_depIdxs,
		MessageInfos:      file_transaction_proto_msgTypes,
	}.Build()
	File_transaction_proto = out.File
	file_transaction_proto_rawDesc = nil
	file_transaction_proto_goTypes = nil
	file_transaction_proto_depIdxs = nil
}
