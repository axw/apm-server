// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: cloud.proto

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

type Cloud struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AvailabilityZone string         `protobuf:"bytes,1,opt,name=availability_zone,json=availabilityZone,proto3" json:"availability_zone,omitempty"`
	Region           string         `protobuf:"bytes,2,opt,name=region,proto3" json:"region,omitempty"`
	Provider         string         `protobuf:"bytes,3,opt,name=provider,proto3" json:"provider,omitempty"`
	Account          *CloudAccount  `protobuf:"bytes,4,opt,name=account,proto3" json:"account,omitempty"`
	Instance         *CloudInstance `protobuf:"bytes,5,opt,name=instance,proto3" json:"instance,omitempty"`
	Machine          *CloudMachine  `protobuf:"bytes,6,opt,name=machine,proto3" json:"machine,omitempty"`
	Project          *CloudProject  `protobuf:"bytes,7,opt,name=project,proto3" json:"project,omitempty"`
	Service          *CloudService  `protobuf:"bytes,8,opt,name=service,proto3" json:"service,omitempty"`
}

func (x *Cloud) Reset() {
	*x = Cloud{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cloud) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cloud) ProtoMessage() {}

func (x *Cloud) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cloud.ProtoReflect.Descriptor instead.
func (*Cloud) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{0}
}

func (x *Cloud) GetAvailabilityZone() string {
	if x != nil {
		return x.AvailabilityZone
	}
	return ""
}

func (x *Cloud) GetRegion() string {
	if x != nil {
		return x.Region
	}
	return ""
}

func (x *Cloud) GetProvider() string {
	if x != nil {
		return x.Provider
	}
	return ""
}

func (x *Cloud) GetAccount() *CloudAccount {
	if x != nil {
		return x.Account
	}
	return nil
}

func (x *Cloud) GetInstance() *CloudInstance {
	if x != nil {
		return x.Instance
	}
	return nil
}

func (x *Cloud) GetMachine() *CloudMachine {
	if x != nil {
		return x.Machine
	}
	return nil
}

func (x *Cloud) GetProject() *CloudProject {
	if x != nil {
		return x.Project
	}
	return nil
}

func (x *Cloud) GetService() *CloudService {
	if x != nil {
		return x.Service
	}
	return nil
}

type CloudAccount struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *CloudAccount) Reset() {
	*x = CloudAccount{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudAccount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudAccount) ProtoMessage() {}

func (x *CloudAccount) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudAccount.ProtoReflect.Descriptor instead.
func (*CloudAccount) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{1}
}

func (x *CloudAccount) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CloudAccount) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type CloudInstance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *CloudInstance) Reset() {
	*x = CloudInstance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudInstance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudInstance) ProtoMessage() {}

func (x *CloudInstance) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudInstance.ProtoReflect.Descriptor instead.
func (*CloudInstance) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{2}
}

func (x *CloudInstance) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CloudInstance) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type CloudMachine struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type string `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *CloudMachine) Reset() {
	*x = CloudMachine{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudMachine) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudMachine) ProtoMessage() {}

func (x *CloudMachine) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudMachine.ProtoReflect.Descriptor instead.
func (*CloudMachine) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{3}
}

func (x *CloudMachine) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type CloudProject struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *CloudProject) Reset() {
	*x = CloudProject{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudProject) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudProject) ProtoMessage() {}

func (x *CloudProject) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudProject.ProtoReflect.Descriptor instead.
func (*CloudProject) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{4}
}

func (x *CloudProject) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CloudProject) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type CloudService struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *CloudService) Reset() {
	*x = CloudService{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cloud_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudService) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudService) ProtoMessage() {}

func (x *CloudService) ProtoReflect() protoreflect.Message {
	mi := &file_cloud_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudService.ProtoReflect.Descriptor instead.
func (*CloudService) Descriptor() ([]byte, []int) {
	return file_cloud_proto_rawDescGZIP(), []int{5}
}

func (x *CloudService) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

var File_cloud_proto protoreflect.FileDescriptor

var file_cloud_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x65,
	0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c,
	0x22, 0x92, 0x03, 0x0a, 0x05, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x12, 0x2b, 0x0a, 0x11, 0x61, 0x76,
	0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x5f, 0x7a, 0x6f, 0x6e, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c,
	0x69, 0x74, 0x79, 0x5a, 0x6f, 0x6e, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f,
	0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x12,
	0x1a, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x12, 0x39, 0x0a, 0x07, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x65,
	0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c,
	0x2e, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x07, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x3c, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e,
	0x63, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74,
	0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x43, 0x6c, 0x6f,
	0x75, 0x64, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74,
	0x61, 0x6e, 0x63, 0x65, 0x12, 0x39, 0x0a, 0x07, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e,
	0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x4d,
	0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x52, 0x07, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x12,
	0x39, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63,
	0x74, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x39, 0x0a, 0x07, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x65, 0x6c,
	0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e,
	0x43, 0x6c, 0x6f, 0x75, 0x64, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x52, 0x07, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x22, 0x32, 0x0a, 0x0c, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x41, 0x63,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x33, 0x0a, 0x0d, 0x43, 0x6c, 0x6f,
	0x75, 0x64, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x22,
	0x0a, 0x0c, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x22, 0x32, 0x0a, 0x0c, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x50, 0x72, 0x6f, 0x6a, 0x65,
	0x63, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x22, 0x0a, 0x0c, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63,
	0x2f, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_cloud_proto_rawDescOnce sync.Once
	file_cloud_proto_rawDescData = file_cloud_proto_rawDesc
)

func file_cloud_proto_rawDescGZIP() []byte {
	file_cloud_proto_rawDescOnce.Do(func() {
		file_cloud_proto_rawDescData = protoimpl.X.CompressGZIP(file_cloud_proto_rawDescData)
	})
	return file_cloud_proto_rawDescData
}

var file_cloud_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_cloud_proto_goTypes = []interface{}{
	(*Cloud)(nil),         // 0: elastic.apm.model.Cloud
	(*CloudAccount)(nil),  // 1: elastic.apm.model.CloudAccount
	(*CloudInstance)(nil), // 2: elastic.apm.model.CloudInstance
	(*CloudMachine)(nil),  // 3: elastic.apm.model.CloudMachine
	(*CloudProject)(nil),  // 4: elastic.apm.model.CloudProject
	(*CloudService)(nil),  // 5: elastic.apm.model.CloudService
}
var file_cloud_proto_depIdxs = []int32{
	1, // 0: elastic.apm.model.Cloud.account:type_name -> elastic.apm.model.CloudAccount
	2, // 1: elastic.apm.model.Cloud.instance:type_name -> elastic.apm.model.CloudInstance
	3, // 2: elastic.apm.model.Cloud.machine:type_name -> elastic.apm.model.CloudMachine
	4, // 3: elastic.apm.model.Cloud.project:type_name -> elastic.apm.model.CloudProject
	5, // 4: elastic.apm.model.Cloud.service:type_name -> elastic.apm.model.CloudService
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_cloud_proto_init() }
func file_cloud_proto_init() {
	if File_cloud_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cloud_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cloud); i {
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
		file_cloud_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudAccount); i {
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
		file_cloud_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudInstance); i {
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
		file_cloud_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudMachine); i {
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
		file_cloud_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudProject); i {
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
		file_cloud_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudService); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cloud_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cloud_proto_goTypes,
		DependencyIndexes: file_cloud_proto_depIdxs,
		MessageInfos:      file_cloud_proto_msgTypes,
	}.Build()
	File_cloud_proto = out.File
	file_cloud_proto_rawDesc = nil
	file_cloud_proto_goTypes = nil
	file_cloud_proto_depIdxs = nil
}
