// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: tag-msgs.proto

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

// requires(NONE)
type TagListReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"` // Is this required?
}

func (x *TagListReq) Reset() {
	*x = TagListReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagListReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagListReq) ProtoMessage() {}

func (x *TagListReq) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagListReq.ProtoReflect.Descriptor instead.
func (*TagListReq) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{0}
}

func (x *TagListReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

type TagListResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tags []*Tag `protobuf:"bytes,1,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *TagListResp) Reset() {
	*x = TagListResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagListResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagListResp) ProtoMessage() {}

func (x *TagListResp) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagListResp.ProtoReflect.Descriptor instead.
func (*TagListResp) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *TagListResp) GetTags() []*Tag {
	if x != nil {
		return x.Tags
	}
	return nil
}

// requires(EDIT_TAGS)
type TagCreateReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"` // Seems to be optional?
}

func (x *TagCreateReq) Reset() {
	*x = TagCreateReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagCreateReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagCreateReq) ProtoMessage() {}

func (x *TagCreateReq) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagCreateReq.ProtoReflect.Descriptor instead.
func (*TagCreateReq) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{2}
}

func (x *TagCreateReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

type TagCreateResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tag *Tag `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
}

func (x *TagCreateResp) Reset() {
	*x = TagCreateResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagCreateResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagCreateResp) ProtoMessage() {}

func (x *TagCreateResp) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagCreateResp.ProtoReflect.Descriptor instead.
func (*TagCreateResp) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *TagCreateResp) GetTag() *Tag {
	if x != nil {
		return x.Tag
	}
	return nil
}

// requires(EDIT_TAGS)
type TagDeleteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"` // Is this required?
	TagId  string `protobuf:"bytes,2,opt,name=tagId,proto3" json:"tagId,omitempty"`
}

func (x *TagDeleteReq) Reset() {
	*x = TagDeleteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagDeleteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagDeleteReq) ProtoMessage() {}

func (x *TagDeleteReq) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagDeleteReq.ProtoReflect.Descriptor instead.
func (*TagDeleteReq) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{4}
}

func (x *TagDeleteReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *TagDeleteReq) GetTagId() string {
	if x != nil {
		return x.TagId
	}
	return ""
}

type TagDeleteResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *TagDeleteResp) Reset() {
	*x = TagDeleteResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_tag_msgs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TagDeleteResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TagDeleteResp) ProtoMessage() {}

func (x *TagDeleteResp) ProtoReflect() protoreflect.Message {
	mi := &file_tag_msgs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TagDeleteResp.ProtoReflect.Descriptor instead.
func (*TagDeleteResp) Descriptor() ([]byte, []int) {
	return file_tag_msgs_proto_rawDescGZIP(), []int{5}
}

var File_tag_msgs_proto protoreflect.FileDescriptor

var file_tag_msgs_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x74, 0x61, 0x67, 0x2d, 0x6d, 0x73, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x0a, 0x74, 0x61, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x24, 0x0a, 0x0a,
	0x54, 0x61, 0x67, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63,
	0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6e,
	0x49, 0x64, 0x22, 0x27, 0x0a, 0x0b, 0x54, 0x61, 0x67, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x12, 0x18, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x04, 0x2e, 0x54, 0x61, 0x67, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x22, 0x26, 0x0a, 0x0c, 0x54,
	0x61, 0x67, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06, 0x73,
	0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61,
	0x6e, 0x49, 0x64, 0x22, 0x27, 0x0a, 0x0d, 0x54, 0x61, 0x67, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x52, 0x65, 0x73, 0x70, 0x12, 0x16, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x04, 0x2e, 0x54, 0x61, 0x67, 0x52, 0x03, 0x74, 0x61, 0x67, 0x22, 0x3c, 0x0a, 0x0c,
	0x54, 0x61, 0x67, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06,
	0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63,
	0x61, 0x6e, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x61, 0x67, 0x49, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x61, 0x67, 0x49, 0x64, 0x22, 0x0f, 0x0a, 0x0d, 0x54, 0x61,
	0x67, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x42, 0x0a, 0x5a, 0x08, 0x2e,
	0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_tag_msgs_proto_rawDescOnce sync.Once
	file_tag_msgs_proto_rawDescData = file_tag_msgs_proto_rawDesc
)

func file_tag_msgs_proto_rawDescGZIP() []byte {
	file_tag_msgs_proto_rawDescOnce.Do(func() {
		file_tag_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_tag_msgs_proto_rawDescData)
	})
	return file_tag_msgs_proto_rawDescData
}

var file_tag_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_tag_msgs_proto_goTypes = []interface{}{
	(*TagListReq)(nil),    // 0: TagListReq
	(*TagListResp)(nil),   // 1: TagListResp
	(*TagCreateReq)(nil),  // 2: TagCreateReq
	(*TagCreateResp)(nil), // 3: TagCreateResp
	(*TagDeleteReq)(nil),  // 4: TagDeleteReq
	(*TagDeleteResp)(nil), // 5: TagDeleteResp
	(*Tag)(nil),           // 6: Tag
}
var file_tag_msgs_proto_depIdxs = []int32{
	6, // 0: TagListResp.tags:type_name -> Tag
	6, // 1: TagCreateResp.tag:type_name -> Tag
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_tag_msgs_proto_init() }
func file_tag_msgs_proto_init() {
	if File_tag_msgs_proto != nil {
		return
	}
	file_tags_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_tag_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagListReq); i {
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
		file_tag_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagListResp); i {
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
		file_tag_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagCreateReq); i {
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
		file_tag_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagCreateResp); i {
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
		file_tag_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagDeleteReq); i {
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
		file_tag_msgs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TagDeleteResp); i {
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
			RawDescriptor: file_tag_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_tag_msgs_proto_goTypes,
		DependencyIndexes: file_tag_msgs_proto_depIdxs,
		MessageInfos:      file_tag_msgs_proto_msgTypes,
	}.Build()
	File_tag_msgs_proto = out.File
	file_tag_msgs_proto_rawDesc = nil
	file_tag_msgs_proto_goTypes = nil
	file_tag_msgs_proto_depIdxs = nil
}