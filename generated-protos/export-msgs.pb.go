// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.20.3
// source: export-msgs.proto

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

type ExportDataType int32

const (
	ExportDataType_EDT_UNKNOWN   ExportDataType = 0 // https://protobuf.dev/programming-guides/dos-donts/ says specify an unknown as 0
	ExportDataType_EDT_QUANT_CSV ExportDataType = 1
)

// Enum value maps for ExportDataType.
var (
	ExportDataType_name = map[int32]string{
		0: "EDT_UNKNOWN",
		1: "EDT_QUANT_CSV",
	}
	ExportDataType_value = map[string]int32{
		"EDT_UNKNOWN":   0,
		"EDT_QUANT_CSV": 1,
	}
)

func (x ExportDataType) Enum() *ExportDataType {
	p := new(ExportDataType)
	*p = x
	return p
}

func (x ExportDataType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ExportDataType) Descriptor() protoreflect.EnumDescriptor {
	return file_export_msgs_proto_enumTypes[0].Descriptor()
}

func (ExportDataType) Type() protoreflect.EnumType {
	return &file_export_msgs_proto_enumTypes[0]
}

func (x ExportDataType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ExportDataType.Descriptor instead.
func (ExportDataType) EnumDescriptor() ([]byte, []int) {
	return file_export_msgs_proto_rawDescGZIP(), []int{0}
}

// requires(EXPORT)
type ExportFilesReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// What to export
	ExportTypes    []ExportDataType `protobuf:"varint,1,rep,packed,name=exportTypes,proto3,enum=ExportDataType" json:"exportTypes,omitempty"`
	ScanId         string           `protobuf:"bytes,2,opt,name=scanId,proto3" json:"scanId,omitempty"`
	QuantId        string           `protobuf:"bytes,3,opt,name=quantId,proto3" json:"quantId,omitempty"`
	RoiIds         []string         `protobuf:"bytes,4,rep,name=roiIds,proto3" json:"roiIds,omitempty"`
	ImageFileNames []string         `protobuf:"bytes,5,rep,name=imageFileNames,proto3" json:"imageFileNames,omitempty"`
}

func (x *ExportFilesReq) Reset() {
	*x = ExportFilesReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_export_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExportFilesReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExportFilesReq) ProtoMessage() {}

func (x *ExportFilesReq) ProtoReflect() protoreflect.Message {
	mi := &file_export_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExportFilesReq.ProtoReflect.Descriptor instead.
func (*ExportFilesReq) Descriptor() ([]byte, []int) {
	return file_export_msgs_proto_rawDescGZIP(), []int{0}
}

func (x *ExportFilesReq) GetExportTypes() []ExportDataType {
	if x != nil {
		return x.ExportTypes
	}
	return nil
}

func (x *ExportFilesReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *ExportFilesReq) GetQuantId() string {
	if x != nil {
		return x.QuantId
	}
	return ""
}

func (x *ExportFilesReq) GetRoiIds() []string {
	if x != nil {
		return x.RoiIds
	}
	return nil
}

func (x *ExportFilesReq) GetImageFileNames() []string {
	if x != nil {
		return x.ImageFileNames
	}
	return nil
}

type ExportFilesResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Just contains the zipped exported data. File name is irrelevant because we expect the UI to present this
	// as needed, potentially unzipping this in memory and zipping UI-generated files into the final output
	Files []*ExportFile `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
}

func (x *ExportFilesResp) Reset() {
	*x = ExportFilesResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_export_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExportFilesResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExportFilesResp) ProtoMessage() {}

func (x *ExportFilesResp) ProtoReflect() protoreflect.Message {
	mi := &file_export_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExportFilesResp.ProtoReflect.Descriptor instead.
func (*ExportFilesResp) Descriptor() ([]byte, []int) {
	return file_export_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *ExportFilesResp) GetFiles() []*ExportFile {
	if x != nil {
		return x.Files
	}
	return nil
}

var File_export_msgs_proto protoreflect.FileDescriptor

var file_export_msgs_proto_rawDesc = []byte{
	0x0a, 0x11, 0x65, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x2d, 0x6d, 0x73, 0x67, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x0c, 0x65, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xb5, 0x01, 0x0a, 0x0e, 0x45, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x46, 0x69, 0x6c, 0x65,
	0x73, 0x52, 0x65, 0x71, 0x12, 0x31, 0x0a, 0x0b, 0x65, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x45, 0x78, 0x70, 0x6f,
	0x72, 0x74, 0x44, 0x61, 0x74, 0x61, 0x54, 0x79, 0x70, 0x65, 0x52, 0x0b, 0x65, 0x78, 0x70, 0x6f,
	0x72, 0x74, 0x54, 0x79, 0x70, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x12,
	0x18, 0x0a, 0x07, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x6f, 0x69,
	0x49, 0x64, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x72, 0x6f, 0x69, 0x49, 0x64,
	0x73, 0x12, 0x26, 0x0a, 0x0e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x4e, 0x61,
	0x6d, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x46, 0x69, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x22, 0x34, 0x0a, 0x0f, 0x45, 0x78, 0x70,
	0x6f, 0x72, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x12, 0x21, 0x0a, 0x05,
	0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x45, 0x78,
	0x70, 0x6f, 0x72, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x2a,
	0x34, 0x0a, 0x0e, 0x45, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x44, 0x61, 0x74, 0x61, 0x54, 0x79, 0x70,
	0x65, 0x12, 0x0f, 0x0a, 0x0b, 0x45, 0x44, 0x54, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e,
	0x10, 0x00, 0x12, 0x11, 0x0a, 0x0d, 0x45, 0x44, 0x54, 0x5f, 0x51, 0x55, 0x41, 0x4e, 0x54, 0x5f,
	0x43, 0x53, 0x56, 0x10, 0x01, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_export_msgs_proto_rawDescOnce sync.Once
	file_export_msgs_proto_rawDescData = file_export_msgs_proto_rawDesc
)

func file_export_msgs_proto_rawDescGZIP() []byte {
	file_export_msgs_proto_rawDescOnce.Do(func() {
		file_export_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_export_msgs_proto_rawDescData)
	})
	return file_export_msgs_proto_rawDescData
}

var file_export_msgs_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_export_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_export_msgs_proto_goTypes = []interface{}{
	(ExportDataType)(0),     // 0: ExportDataType
	(*ExportFilesReq)(nil),  // 1: ExportFilesReq
	(*ExportFilesResp)(nil), // 2: ExportFilesResp
	(*ExportFile)(nil),      // 3: ExportFile
}
var file_export_msgs_proto_depIdxs = []int32{
	0, // 0: ExportFilesReq.exportTypes:type_name -> ExportDataType
	3, // 1: ExportFilesResp.files:type_name -> ExportFile
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_export_msgs_proto_init() }
func file_export_msgs_proto_init() {
	if File_export_msgs_proto != nil {
		return
	}
	file_export_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_export_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExportFilesReq); i {
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
		file_export_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExportFilesResp); i {
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
			RawDescriptor: file_export_msgs_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_export_msgs_proto_goTypes,
		DependencyIndexes: file_export_msgs_proto_depIdxs,
		EnumInfos:         file_export_msgs_proto_enumTypes,
		MessageInfos:      file_export_msgs_proto_msgTypes,
	}.Build()
	File_export_msgs_proto = out.File
	file_export_msgs_proto_rawDesc = nil
	file_export_msgs_proto_goTypes = nil
	file_export_msgs_proto_depIdxs = nil
}
