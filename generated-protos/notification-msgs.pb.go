// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: notification-msgs.proto

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
type NotificationReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *NotificationReq) Reset() {
	*x = NotificationReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotificationReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotificationReq) ProtoMessage() {}

func (x *NotificationReq) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotificationReq.ProtoReflect.Descriptor instead.
func (*NotificationReq) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{0}
}

type NotificationResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notification []*Notification `protobuf:"bytes,1,rep,name=notification,proto3" json:"notification,omitempty"`
}

func (x *NotificationResp) Reset() {
	*x = NotificationResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotificationResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotificationResp) ProtoMessage() {}

func (x *NotificationResp) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotificationResp.ProtoReflect.Descriptor instead.
func (*NotificationResp) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *NotificationResp) GetNotification() []*Notification {
	if x != nil {
		return x.Notification
	}
	return nil
}

type NotificationUpd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notification *Notification `protobuf:"bytes,2,opt,name=notification,proto3" json:"notification,omitempty"`
}

func (x *NotificationUpd) Reset() {
	*x = NotificationUpd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotificationUpd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotificationUpd) ProtoMessage() {}

func (x *NotificationUpd) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotificationUpd.ProtoReflect.Descriptor instead.
func (*NotificationUpd) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{2}
}

func (x *NotificationUpd) GetNotification() *Notification {
	if x != nil {
		return x.Notification
	}
	return nil
}

// requires(NONE)
type NotificationDismissReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *NotificationDismissReq) Reset() {
	*x = NotificationDismissReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotificationDismissReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotificationDismissReq) ProtoMessage() {}

func (x *NotificationDismissReq) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotificationDismissReq.ProtoReflect.Descriptor instead.
func (*NotificationDismissReq) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *NotificationDismissReq) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type NotificationDismissResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *NotificationDismissResp) Reset() {
	*x = NotificationDismissResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotificationDismissResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotificationDismissResp) ProtoMessage() {}

func (x *NotificationDismissResp) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotificationDismissResp.ProtoReflect.Descriptor instead.
func (*NotificationDismissResp) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{4}
}

// Admin-only feature, to send out a notification to all users
// requires(PIXLISE_ADMIN)
type SendUserNotificationReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserIds      []string      `protobuf:"bytes,1,rep,name=userIds,proto3" json:"userIds,omitempty"`
	GroupIds     []string      `protobuf:"bytes,2,rep,name=groupIds,proto3" json:"groupIds,omitempty"`
	Notification *Notification `protobuf:"bytes,3,opt,name=notification,proto3" json:"notification,omitempty"`
}

func (x *SendUserNotificationReq) Reset() {
	*x = SendUserNotificationReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendUserNotificationReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendUserNotificationReq) ProtoMessage() {}

func (x *SendUserNotificationReq) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendUserNotificationReq.ProtoReflect.Descriptor instead.
func (*SendUserNotificationReq) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{5}
}

func (x *SendUserNotificationReq) GetUserIds() []string {
	if x != nil {
		return x.UserIds
	}
	return nil
}

func (x *SendUserNotificationReq) GetGroupIds() []string {
	if x != nil {
		return x.GroupIds
	}
	return nil
}

func (x *SendUserNotificationReq) GetNotification() *Notification {
	if x != nil {
		return x.Notification
	}
	return nil
}

type SendUserNotificationResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SendUserNotificationResp) Reset() {
	*x = SendUserNotificationResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_notification_msgs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendUserNotificationResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendUserNotificationResp) ProtoMessage() {}

func (x *SendUserNotificationResp) ProtoReflect() protoreflect.Message {
	mi := &file_notification_msgs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendUserNotificationResp.ProtoReflect.Descriptor instead.
func (*SendUserNotificationResp) Descriptor() ([]byte, []int) {
	return file_notification_msgs_proto_rawDescGZIP(), []int{6}
}

var File_notification_msgs_proto protoreflect.FileDescriptor

var file_notification_msgs_proto_rawDesc = []byte{
	0x0a, 0x17, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2d, 0x6d,
	0x73, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x12, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x11, 0x0a,
	0x0f, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71,
	0x22, 0x45, 0x0a, 0x10, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x52, 0x65, 0x73, 0x70, 0x12, 0x31, 0x0a, 0x0c, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x4e, 0x6f, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x44, 0x0a, 0x0f, 0x4e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x55, 0x70, 0x64, 0x12, 0x31, 0x0a, 0x0c, 0x6e, 0x6f,
	0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0d, 0x2e, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x0c, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x28, 0x0a,
	0x16, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x69, 0x73,
	0x6d, 0x69, 0x73, 0x73, 0x52, 0x65, 0x71, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x19, 0x0a, 0x17, 0x4e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x69, 0x73, 0x6d, 0x69, 0x73, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x22, 0x82, 0x01, 0x0a, 0x17, 0x53, 0x65, 0x6e, 0x64, 0x55, 0x73, 0x65, 0x72, 0x4e,
	0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x12, 0x18,
	0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x07, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x49, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x49, 0x64, 0x73, 0x12, 0x31, 0x0a, 0x0c, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x4e, 0x6f, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x1a, 0x0a, 0x18, 0x53, 0x65, 0x6e, 0x64, 0x55,
	0x73, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x73, 0x70, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_notification_msgs_proto_rawDescOnce sync.Once
	file_notification_msgs_proto_rawDescData = file_notification_msgs_proto_rawDesc
)

func file_notification_msgs_proto_rawDescGZIP() []byte {
	file_notification_msgs_proto_rawDescOnce.Do(func() {
		file_notification_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_notification_msgs_proto_rawDescData)
	})
	return file_notification_msgs_proto_rawDescData
}

var file_notification_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_notification_msgs_proto_goTypes = []interface{}{
	(*NotificationReq)(nil),          // 0: NotificationReq
	(*NotificationResp)(nil),         // 1: NotificationResp
	(*NotificationUpd)(nil),          // 2: NotificationUpd
	(*NotificationDismissReq)(nil),   // 3: NotificationDismissReq
	(*NotificationDismissResp)(nil),  // 4: NotificationDismissResp
	(*SendUserNotificationReq)(nil),  // 5: SendUserNotificationReq
	(*SendUserNotificationResp)(nil), // 6: SendUserNotificationResp
	(*Notification)(nil),             // 7: Notification
}
var file_notification_msgs_proto_depIdxs = []int32{
	7, // 0: NotificationResp.notification:type_name -> Notification
	7, // 1: NotificationUpd.notification:type_name -> Notification
	7, // 2: SendUserNotificationReq.notification:type_name -> Notification
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_notification_msgs_proto_init() }
func file_notification_msgs_proto_init() {
	if File_notification_msgs_proto != nil {
		return
	}
	file_notification_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_notification_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotificationReq); i {
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
		file_notification_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotificationResp); i {
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
		file_notification_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotificationUpd); i {
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
		file_notification_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotificationDismissReq); i {
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
		file_notification_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotificationDismissResp); i {
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
		file_notification_msgs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendUserNotificationReq); i {
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
		file_notification_msgs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendUserNotificationResp); i {
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
			RawDescriptor: file_notification_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_notification_msgs_proto_goTypes,
		DependencyIndexes: file_notification_msgs_proto_depIdxs,
		MessageInfos:      file_notification_msgs_proto_msgTypes,
	}.Build()
	File_notification_msgs_proto = out.File
	file_notification_msgs_proto_rawDesc = nil
	file_notification_msgs_proto_goTypes = nil
	file_notification_msgs_proto_depIdxs = nil
}
