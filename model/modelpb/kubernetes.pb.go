// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: apm-server/model/proto/kubernetes.proto

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

type Kubernetes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string          `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Node      *KubernetesNode `protobuf:"bytes,2,opt,name=node,proto3" json:"node,omitempty"`
	Pod       *KubernetesPod  `protobuf:"bytes,3,opt,name=pod,proto3" json:"pod,omitempty"`
}

func (x *Kubernetes) Reset() {
	*x = Kubernetes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Kubernetes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Kubernetes) ProtoMessage() {}

func (x *Kubernetes) ProtoReflect() protoreflect.Message {
	mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Kubernetes.ProtoReflect.Descriptor instead.
func (*Kubernetes) Descriptor() ([]byte, []int) {
	return file_apm_server_model_proto_kubernetes_proto_rawDescGZIP(), []int{0}
}

func (x *Kubernetes) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *Kubernetes) GetNode() *KubernetesNode {
	if x != nil {
		return x.Node
	}
	return nil
}

func (x *Kubernetes) GetPod() *KubernetesPod {
	if x != nil {
		return x.Pod
	}
	return nil
}

type KubernetesNode struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *KubernetesNode) Reset() {
	*x = KubernetesNode{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KubernetesNode) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KubernetesNode) ProtoMessage() {}

func (x *KubernetesNode) ProtoReflect() protoreflect.Message {
	mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KubernetesNode.ProtoReflect.Descriptor instead.
func (*KubernetesNode) Descriptor() ([]byte, []int) {
	return file_apm_server_model_proto_kubernetes_proto_rawDescGZIP(), []int{1}
}

func (x *KubernetesNode) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type KubernetesPod struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Uid  string `protobuf:"bytes,2,opt,name=uid,proto3" json:"uid,omitempty"`
}

func (x *KubernetesPod) Reset() {
	*x = KubernetesPod{}
	if protoimpl.UnsafeEnabled {
		mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KubernetesPod) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KubernetesPod) ProtoMessage() {}

func (x *KubernetesPod) ProtoReflect() protoreflect.Message {
	mi := &file_apm_server_model_proto_kubernetes_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KubernetesPod.ProtoReflect.Descriptor instead.
func (*KubernetesPod) Descriptor() ([]byte, []int) {
	return file_apm_server_model_proto_kubernetes_proto_rawDescGZIP(), []int{2}
}

func (x *KubernetesPod) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *KubernetesPod) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

var File_apm_server_model_proto_kubernetes_proto protoreflect.FileDescriptor

var file_apm_server_model_proto_kubernetes_proto_rawDesc = []byte{
	0x0a, 0x27, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65,
	0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x65, 0x6c, 0x61, 0x73, 0x74,
	0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x22, 0x95, 0x01, 0x0a,
	0x0a, 0x4b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x6e,
	0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x35, 0x0a, 0x04, 0x6e, 0x6f, 0x64,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x65, 0x6c, 0x61, 0x73, 0x74, 0x69,
	0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x4b, 0x75, 0x62, 0x65,
	0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x04, 0x6e, 0x6f, 0x64, 0x65,
	0x12, 0x32, 0x0a, 0x03, 0x70, 0x6f, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e,
	0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2e, 0x61, 0x70, 0x6d, 0x2e, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x2e, 0x4b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x50, 0x6f, 0x64, 0x52,
	0x03, 0x70, 0x6f, 0x64, 0x22, 0x24, 0x0a, 0x0e, 0x4b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74,
	0x65, 0x73, 0x4e, 0x6f, 0x64, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x35, 0x0a, 0x0d, 0x4b, 0x75,
	0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x50, 0x6f, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x10, 0x0a, 0x03, 0x75, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x69,
	0x64, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x65, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x2f, 0x61, 0x70, 0x6d, 0x2d, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x70, 0x62,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_apm_server_model_proto_kubernetes_proto_rawDescOnce sync.Once
	file_apm_server_model_proto_kubernetes_proto_rawDescData = file_apm_server_model_proto_kubernetes_proto_rawDesc
)

func file_apm_server_model_proto_kubernetes_proto_rawDescGZIP() []byte {
	file_apm_server_model_proto_kubernetes_proto_rawDescOnce.Do(func() {
		file_apm_server_model_proto_kubernetes_proto_rawDescData = protoimpl.X.CompressGZIP(file_apm_server_model_proto_kubernetes_proto_rawDescData)
	})
	return file_apm_server_model_proto_kubernetes_proto_rawDescData
}

var file_apm_server_model_proto_kubernetes_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_apm_server_model_proto_kubernetes_proto_goTypes = []interface{}{
	(*Kubernetes)(nil),     // 0: elastic.apm.model.Kubernetes
	(*KubernetesNode)(nil), // 1: elastic.apm.model.KubernetesNode
	(*KubernetesPod)(nil),  // 2: elastic.apm.model.KubernetesPod
}
var file_apm_server_model_proto_kubernetes_proto_depIdxs = []int32{
	1, // 0: elastic.apm.model.Kubernetes.node:type_name -> elastic.apm.model.KubernetesNode
	2, // 1: elastic.apm.model.Kubernetes.pod:type_name -> elastic.apm.model.KubernetesPod
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_apm_server_model_proto_kubernetes_proto_init() }
func file_apm_server_model_proto_kubernetes_proto_init() {
	if File_apm_server_model_proto_kubernetes_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_apm_server_model_proto_kubernetes_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Kubernetes); i {
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
		file_apm_server_model_proto_kubernetes_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KubernetesNode); i {
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
		file_apm_server_model_proto_kubernetes_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KubernetesPod); i {
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
			RawDescriptor: file_apm_server_model_proto_kubernetes_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_apm_server_model_proto_kubernetes_proto_goTypes,
		DependencyIndexes: file_apm_server_model_proto_kubernetes_proto_depIdxs,
		MessageInfos:      file_apm_server_model_proto_kubernetes_proto_msgTypes,
	}.Build()
	File_apm_server_model_proto_kubernetes_proto = out.File
	file_apm_server_model_proto_kubernetes_proto_rawDesc = nil
	file_apm_server_model_proto_kubernetes_proto_goTypes = nil
	file_apm_server_model_proto_kubernetes_proto_depIdxs = nil
}
