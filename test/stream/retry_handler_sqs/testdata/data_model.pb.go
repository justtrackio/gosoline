// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v4.24.2
// source: data_model.proto

package testdata

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DataModel struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *DataModel) Reset() {
	*x = DataModel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_data_model_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataModel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataModel) ProtoMessage() {}

func (x *DataModel) ProtoReflect() protoreflect.Message {
	mi := &file_data_model_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataModel.ProtoReflect.Descriptor instead.
func (*DataModel) Descriptor() ([]byte, []int) {
	return file_data_model_proto_rawDescGZIP(), []int{0}
}

func (x *DataModel) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

var File_data_model_proto protoreflect.FileDescriptor

var file_data_model_proto_rawDesc = []byte{
	0x0a, 0x10, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x1b, 0x0a, 0x09, 0x44, 0x61, 0x74, 0x61, 0x4d, 0x6f, 0x64, 0x65, 0x6c, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x42,
	0x1c, 0x5a, 0x1a, 0x72, 0x65, 0x74, 0x72, 0x79, 0x5f, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72,
	0x5f, 0x73, 0x71, 0x73, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_data_model_proto_rawDescOnce sync.Once
	file_data_model_proto_rawDescData = file_data_model_proto_rawDesc
)

func file_data_model_proto_rawDescGZIP() []byte {
	file_data_model_proto_rawDescOnce.Do(func() {
		file_data_model_proto_rawDescData = protoimpl.X.CompressGZIP(file_data_model_proto_rawDescData)
	})
	return file_data_model_proto_rawDescData
}

var (
	file_data_model_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
	file_data_model_proto_goTypes  = []any{
		(*DataModel)(nil), // 0: DataModel
	}
)

var file_data_model_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_data_model_proto_init() }
func file_data_model_proto_init() {
	if File_data_model_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_data_model_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*DataModel); i {
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
			RawDescriptor: file_data_model_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_data_model_proto_goTypes,
		DependencyIndexes: file_data_model_proto_depIdxs,
		MessageInfos:      file_data_model_proto_msgTypes,
	}.Build()
	File_data_model_proto = out.File
	file_data_model_proto_rawDesc = nil
	file_data_model_proto_goTypes = nil
	file_data_model_proto_depIdxs = nil
}
