// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: scan-beam-location.proto

package protos

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

// We store physical location in these
type Coordinate3D struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
}

func (x *Coordinate3D) Reset() {
	*x = Coordinate3D{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_beam_location_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Coordinate3D) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Coordinate3D) ProtoMessage() {}

func (x *Coordinate3D) ProtoReflect() protoreflect.Message {
	mi := &file_scan_beam_location_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Coordinate3D.ProtoReflect.Descriptor instead.
func (*Coordinate3D) Descriptor() ([]byte, []int) {
	return file_scan_beam_location_proto_rawDescGZIP(), []int{0}
}

func (x *Coordinate3D) GetX() float32 {
	if x != nil {
		return x.X
	}
	return 0
}

func (x *Coordinate3D) GetY() float32 {
	if x != nil {
		return x.Y
	}
	return 0
}

func (x *Coordinate3D) GetZ() float32 {
	if x != nil {
		return x.Z
	}
	return 0
}

var File_scan_beam_location_proto protoreflect.FileDescriptor

var file_scan_beam_location_proto_rawDesc = []byte{
	0x0a, 0x18, 0x73, 0x63, 0x61, 0x6e, 0x2d, 0x62, 0x65, 0x61, 0x6d, 0x2d, 0x6c, 0x6f, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x38, 0x0a, 0x0c, 0x43, 0x6f,
	0x6f, 0x72, 0x64, 0x69, 0x6e, 0x61, 0x74, 0x65, 0x33, 0x44, 0x12, 0x0c, 0x0a, 0x01, 0x78, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x78, 0x12, 0x0c, 0x0a, 0x01, 0x79, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x02, 0x52, 0x01, 0x79, 0x12, 0x0c, 0x0a, 0x01, 0x7a, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x02, 0x52, 0x01, 0x7a, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_scan_beam_location_proto_rawDescOnce sync.Once
	file_scan_beam_location_proto_rawDescData = file_scan_beam_location_proto_rawDesc
)

func file_scan_beam_location_proto_rawDescGZIP() []byte {
	file_scan_beam_location_proto_rawDescOnce.Do(func() {
		file_scan_beam_location_proto_rawDescData = protoimpl.X.CompressGZIP(file_scan_beam_location_proto_rawDescData)
	})
	return file_scan_beam_location_proto_rawDescData
}

var file_scan_beam_location_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_scan_beam_location_proto_goTypes = []interface{}{
	(*Coordinate3D)(nil), // 0: Coordinate3D
}
var file_scan_beam_location_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_scan_beam_location_proto_init() }
func file_scan_beam_location_proto_init() {
	if File_scan_beam_location_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_scan_beam_location_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Coordinate3D); i {
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
			RawDescriptor: file_scan_beam_location_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_scan_beam_location_proto_goTypes,
		DependencyIndexes: file_scan_beam_location_proto_depIdxs,
		MessageInfos:      file_scan_beam_location_proto_msgTypes,
	}.Build()
	File_scan_beam_location_proto = out.File
	file_scan_beam_location_proto_rawDesc = nil
	file_scan_beam_location_proto_goTypes = nil
	file_scan_beam_location_proto_depIdxs = nil
}