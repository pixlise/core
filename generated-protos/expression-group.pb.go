// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: expression-group.proto

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

type ExpressionGroupItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ExpressionId string  `protobuf:"bytes,1,opt,name=expressionId,proto3" json:"expressionId,omitempty"`
	RangeMin     float32 `protobuf:"fixed32,2,opt,name=rangeMin,proto3" json:"rangeMin,omitempty"`
	RangeMax     float32 `protobuf:"fixed32,3,opt,name=rangeMax,proto3" json:"rangeMax,omitempty"`
}

func (x *ExpressionGroupItem) Reset() {
	*x = ExpressionGroupItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_expression_group_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExpressionGroupItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExpressionGroupItem) ProtoMessage() {}

func (x *ExpressionGroupItem) ProtoReflect() protoreflect.Message {
	mi := &file_expression_group_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExpressionGroupItem.ProtoReflect.Descriptor instead.
func (*ExpressionGroupItem) Descriptor() ([]byte, []int) {
	return file_expression_group_proto_rawDescGZIP(), []int{0}
}

func (x *ExpressionGroupItem) GetExpressionId() string {
	if x != nil {
		return x.ExpressionId
	}
	return ""
}

func (x *ExpressionGroupItem) GetRangeMin() float32 {
	if x != nil {
		return x.RangeMin
	}
	return 0
}

func (x *ExpressionGroupItem) GetRangeMax() float32 {
	if x != nil {
		return x.RangeMax
	}
	return 0
}

type ExpressionGroup struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id              string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" bson:"_id,omitempty"`  
	Name            string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	GroupItems      []*ExpressionGroupItem `protobuf:"bytes,3,rep,name=groupItems,proto3" json:"groupItems,omitempty"`
	Tags            []string               `protobuf:"bytes,4,rep,name=tags,proto3" json:"tags,omitempty"`
	ModifiedUnixSec uint32                 `protobuf:"varint,5,opt,name=modifiedUnixSec,proto3" json:"modifiedUnixSec,omitempty"`
	Description     string                 `protobuf:"bytes,7,opt,name=description,proto3" json:"description,omitempty"`
	// Only sent out by API, not stored in DB this way
	Owner *OwnershipSummary `protobuf:"bytes,6,opt,name=owner,proto3" json:"owner,omitempty" bson:"-"`  
}

func (x *ExpressionGroup) Reset() {
	*x = ExpressionGroup{}
	if protoimpl.UnsafeEnabled {
		mi := &file_expression_group_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExpressionGroup) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExpressionGroup) ProtoMessage() {}

func (x *ExpressionGroup) ProtoReflect() protoreflect.Message {
	mi := &file_expression_group_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExpressionGroup.ProtoReflect.Descriptor instead.
func (*ExpressionGroup) Descriptor() ([]byte, []int) {
	return file_expression_group_proto_rawDescGZIP(), []int{1}
}

func (x *ExpressionGroup) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ExpressionGroup) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ExpressionGroup) GetGroupItems() []*ExpressionGroupItem {
	if x != nil {
		return x.GroupItems
	}
	return nil
}

func (x *ExpressionGroup) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

func (x *ExpressionGroup) GetModifiedUnixSec() uint32 {
	if x != nil {
		return x.ModifiedUnixSec
	}
	return 0
}

func (x *ExpressionGroup) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *ExpressionGroup) GetOwner() *OwnershipSummary {
	if x != nil {
		return x.Owner
	}
	return nil
}

var File_expression_group_proto protoreflect.FileDescriptor

var file_expression_group_proto_rawDesc = []byte{
	0x0a, 0x16, 0x65, 0x78, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x2d, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x73,
	0x68, 0x69, 0x70, 0x2d, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x71, 0x0a, 0x13, 0x45, 0x78, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x49, 0x74, 0x65, 0x6d, 0x12, 0x22, 0x0a, 0x0c, 0x65, 0x78, 0x70, 0x72, 0x65,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x65,
	0x78, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x72,
	0x61, 0x6e, 0x67, 0x65, 0x4d, 0x69, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x08, 0x72,
	0x61, 0x6e, 0x67, 0x65, 0x4d, 0x69, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x61, 0x6e, 0x67, 0x65,
	0x4d, 0x61, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02, 0x52, 0x08, 0x72, 0x61, 0x6e, 0x67, 0x65,
	0x4d, 0x61, 0x78, 0x22, 0xf4, 0x01, 0x0a, 0x0f, 0x45, 0x78, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x34, 0x0a, 0x0a, 0x67,
	0x72, 0x6f, 0x75, 0x70, 0x49, 0x74, 0x65, 0x6d, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x45, 0x78, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x47, 0x72, 0x6f, 0x75,
	0x70, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x0a, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x74, 0x65, 0x6d,
	0x73, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x04, 0x74, 0x61, 0x67, 0x73, 0x12, 0x28, 0x0a, 0x0f, 0x6d, 0x6f, 0x64, 0x69, 0x66, 0x69, 0x65,
	0x64, 0x55, 0x6e, 0x69, 0x78, 0x53, 0x65, 0x63, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0f,
	0x6d, 0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x55, 0x6e, 0x69, 0x78, 0x53, 0x65, 0x63, 0x12,
	0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x27, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x53, 0x75, 0x6d, 0x6d,
	0x61, 0x72, 0x79, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_expression_group_proto_rawDescOnce sync.Once
	file_expression_group_proto_rawDescData = file_expression_group_proto_rawDesc
)

func file_expression_group_proto_rawDescGZIP() []byte {
	file_expression_group_proto_rawDescOnce.Do(func() {
		file_expression_group_proto_rawDescData = protoimpl.X.CompressGZIP(file_expression_group_proto_rawDescData)
	})
	return file_expression_group_proto_rawDescData
}

var file_expression_group_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_expression_group_proto_goTypes = []interface{}{
	(*ExpressionGroupItem)(nil), // 0: ExpressionGroupItem
	(*ExpressionGroup)(nil),     // 1: ExpressionGroup
	(*OwnershipSummary)(nil),    // 2: OwnershipSummary
}
var file_expression_group_proto_depIdxs = []int32{
	0, // 0: ExpressionGroup.groupItems:type_name -> ExpressionGroupItem
	2, // 1: ExpressionGroup.owner:type_name -> OwnershipSummary
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_expression_group_proto_init() }
func file_expression_group_proto_init() {
	if File_expression_group_proto != nil {
		return
	}
	file_ownership_access_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_expression_group_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExpressionGroupItem); i {
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
		file_expression_group_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExpressionGroup); i {
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
			RawDescriptor: file_expression_group_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_expression_group_proto_goTypes,
		DependencyIndexes: file_expression_group_proto_depIdxs,
		MessageInfos:      file_expression_group_proto_msgTypes,
	}.Build()
	File_expression_group_proto = out.File
	file_expression_group_proto_rawDesc = nil
	file_expression_group_proto_goTypes = nil
	file_expression_group_proto_depIdxs = nil
}
