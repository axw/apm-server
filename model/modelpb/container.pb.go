// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: apm-server/model/proto/container.proto

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

type Container struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string          `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name    string          `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Runtime string          `protobuf:"bytes,3,opt,name=runtime,proto3" json:"runtime,omitempty"`
	Image   *ContainerImage `protobuf:"bytes,4,opt,name=image,proto3" json:"image,omitempty"`
}

func (x *Container) Reset() {
	*x = Container{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apm_server_model_proto_container_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Container) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Container) ProtoMessage() {}

func (x *Container) ProtoReflect() protoreflect.Message {
	mi := &file_apm_server_model_proto_container_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Container.ProtoReflect.Descriptor instead.
func (*Container) Descriptor() ([]byte, []int) {
	return file_apm_server_model_proto_container_proto_rawDescGZIP(), []int{0}
}

func (x *Container) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Container) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Container) GetRuntime() string {
	if x != nil {
		return x.Runtime
	}
	return ""
}

func (x *Container) GetImage() *ContainerImage {
	if x != nil {
		return x.Image
	}
	return nil
}

type ContainerImage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Tag  string `protobuf:"bytes,2,opt,name=tag,proto3" json:"tag,omitempty"`
}

func (x *ContainerImage) Reset() {
	*x = ContainerImage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apm_server_model_proto_container_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ContainerImage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContainerImage) ProtoMessage() {}

func (x *ContainerImage) ProtoReflect() protoreflect.Message {
	mi := &file_apm_server_model_proto_container_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContainerImage.ProtoReflect.Descriptor instead.
func (*ContainerImage) Descriptor() ([]byte, []int) {
	return file_apm_server_model_proto_container_proto_rawDescGZIP(), []int{1}
}

func (x *ContainerImage) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ContainerImage) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

var File_apm_server_model_proto_container_proto protoreflect.FileDescriptor

var file_apm_server_model_proto_container_proto_rawDesc = []byte{
	0x0a, 0x26, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e,
	0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69,
	0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x22, 0x82, 0x01, 0x0a, 0x09,
	0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a,
	0x07, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x12, 0x37, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63,
	0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x61,
	0x69, 0x6e, 0x65, 0x72, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x22, 0x36, 0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x49, 0x6d, 0x61,
	0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x74, 0x61, 0x67, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2f, 0x61,
	0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2f,
	0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_apm_server_model_proto_container_proto_rawDescOnce sync.Once
	file_apm_server_model_proto_container_proto_rawDescData = file_apm_server_model_proto_container_proto_rawDesc
)

func file_apm_server_model_proto_container_proto_rawDescGZIP() []byte {
	file_apm_server_model_proto_container_proto_rawDescOnce.Do(func() {
		file_apm_server_model_proto_container_proto_rawDescData = protoimpl.X.CompressGZIP(file_apm_server_model_proto_container_proto_rawDescData)
	})
	return file_apm_server_model_proto_container_proto_rawDescData
}

var file_apm_server_model_proto_container_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_apm_server_model_proto_container_proto_goTypes = []interface{}{
	(*Container)(nil),      // 0: elastic.apm.model.Container
	(*ContainerImage)(nil), // 1: elastic.apm.model.ContainerImage
}
var file_apm_server_model_proto_container_proto_depIdxs = []int32{
	1, // 0: elastic.apm.model.Container.image:type_name -> elastic.apm.model.ContainerImage
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_apm_server_model_proto_container_proto_init() }
func file_apm_server_model_proto_container_proto_init() {
	if File_apm_server_model_proto_container_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_apm_server_model_proto_container_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Container); i {
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
		file_apm_server_model_proto_container_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ContainerImage); i {
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
			RawDescriptor: file_apm_server_model_proto_container_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_apm_server_model_proto_container_proto_goTypes,
		DependencyIndexes: file_apm_server_model_proto_container_proto_depIdxs,
		MessageInfos:      file_apm_server_model_proto_container_proto_msgTypes,
	}.Build()
	File_apm_server_model_proto_container_proto = out.File
	file_apm_server_model_proto_container_proto_rawDesc = nil
	file_apm_server_model_proto_container_proto_goTypes = nil
	file_apm_server_model_proto_container_proto_depIdxs = nil
}
