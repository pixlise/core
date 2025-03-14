// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: user-notification-setting-msgs.proto

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

// Retrieving a users notification settings (NOT the notifications themselves)
// requires(NONE)
type UserNotificationSettingsReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *UserNotificationSettingsReq) Reset() {
	*x = UserNotificationSettingsReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_notification_setting_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserNotificationSettingsReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserNotificationSettingsReq) ProtoMessage() {}

func (x *UserNotificationSettingsReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_notification_setting_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserNotificationSettingsReq.ProtoReflect.Descriptor instead.
func (*UserNotificationSettingsReq) Descriptor() ([]byte, []int) {
	return file_user_notification_setting_msgs_proto_rawDescGZIP(), []int{0}
}

type UserNotificationSettingsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notifications *UserNotificationSettings `protobuf:"bytes,1,opt,name=notifications,proto3" json:"notifications,omitempty"`
}

func (x *UserNotificationSettingsResp) Reset() {
	*x = UserNotificationSettingsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_notification_setting_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserNotificationSettingsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserNotificationSettingsResp) ProtoMessage() {}

func (x *UserNotificationSettingsResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_notification_setting_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserNotificationSettingsResp.ProtoReflect.Descriptor instead.
func (*UserNotificationSettingsResp) Descriptor() ([]byte, []int) {
	return file_user_notification_setting_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *UserNotificationSettingsResp) GetNotifications() *UserNotificationSettings {
	if x != nil {
		return x.Notifications
	}
	return nil
}

type UserNotificationSettingsUpd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notifications *UserNotificationSettings `protobuf:"bytes,1,opt,name=notifications,proto3" json:"notifications,omitempty"`
}

func (x *UserNotificationSettingsUpd) Reset() {
	*x = UserNotificationSettingsUpd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_notification_setting_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserNotificationSettingsUpd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserNotificationSettingsUpd) ProtoMessage() {}

func (x *UserNotificationSettingsUpd) ProtoReflect() protoreflect.Message {
	mi := &file_user_notification_setting_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserNotificationSettingsUpd.ProtoReflect.Descriptor instead.
func (*UserNotificationSettingsUpd) Descriptor() ([]byte, []int) {
	return file_user_notification_setting_msgs_proto_rawDescGZIP(), []int{2}
}

func (x *UserNotificationSettingsUpd) GetNotifications() *UserNotificationSettings {
	if x != nil {
		return x.Notifications
	}
	return nil
}

// Modifying notifications should publish a UserNotificationSettingsUpd
// requires(EDIT_OWN_USER)
type UserNotificationSettingsWriteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notifications *UserNotificationSettings `protobuf:"bytes,1,opt,name=notifications,proto3" json:"notifications,omitempty"`
}

func (x *UserNotificationSettingsWriteReq) Reset() {
	*x = UserNotificationSettingsWriteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_notification_setting_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserNotificationSettingsWriteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserNotificationSettingsWriteReq) ProtoMessage() {}

func (x *UserNotificationSettingsWriteReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_notification_setting_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserNotificationSettingsWriteReq.ProtoReflect.Descriptor instead.
func (*UserNotificationSettingsWriteReq) Descriptor() ([]byte, []int) {
	return file_user_notification_setting_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *UserNotificationSettingsWriteReq) GetNotifications() *UserNotificationSettings {
	if x != nil {
		return x.Notifications
	}
	return nil
}

type UserNotificationSettingsWriteResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *UserNotificationSettingsWriteResp) Reset() {
	*x = UserNotificationSettingsWriteResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_notification_setting_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserNotificationSettingsWriteResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserNotificationSettingsWriteResp) ProtoMessage() {}

func (x *UserNotificationSettingsWriteResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_notification_setting_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserNotificationSettingsWriteResp.ProtoReflect.Descriptor instead.
func (*UserNotificationSettingsWriteResp) Descriptor() ([]byte, []int) {
	return file_user_notification_setting_msgs_proto_rawDescGZIP(), []int{4}
}

var File_user_notification_setting_msgs_proto protoreflect.FileDescriptor

var file_user_notification_setting_msgs_proto_rawDesc = []byte{
	0x0a, 0x24, 0x75, 0x73, 0x65, 0x72, 0x2d, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2d, 0x73, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x2d, 0x6d, 0x73, 0x67, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x75, 0x73, 0x65, 0x72, 0x2d, 0x6e, 0x6f, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2d, 0x73, 0x65, 0x74, 0x74, 0x69, 0x6e,
	0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x1d, 0x0a, 0x1b, 0x55, 0x73, 0x65, 0x72,
	0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74,
	0x69, 0x6e, 0x67, 0x73, 0x52, 0x65, 0x71, 0x22, 0x5f, 0x0a, 0x1c, 0x55, 0x73, 0x65, 0x72, 0x4e,
	0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74, 0x69,
	0x6e, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x12, 0x3f, 0x0a, 0x0d, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x55, 0x73, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x52, 0x0d, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x5e, 0x0a, 0x1b, 0x55, 0x73, 0x65, 0x72,
	0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74,
	0x69, 0x6e, 0x67, 0x73, 0x55, 0x70, 0x64, 0x12, 0x3f, 0x0a, 0x0d, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x55, 0x73, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x52, 0x0d, 0x6e, 0x6f, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x63, 0x0a, 0x20, 0x55, 0x73, 0x65, 0x72,
	0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74,
	0x69, 0x6e, 0x67, 0x73, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12, 0x3f, 0x0a, 0x0d,
	0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x52, 0x0d,
	0x6e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x23, 0x0a,
	0x21, 0x55, 0x73, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_user_notification_setting_msgs_proto_rawDescOnce sync.Once
	file_user_notification_setting_msgs_proto_rawDescData = file_user_notification_setting_msgs_proto_rawDesc
)

func file_user_notification_setting_msgs_proto_rawDescGZIP() []byte {
	file_user_notification_setting_msgs_proto_rawDescOnce.Do(func() {
		file_user_notification_setting_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_user_notification_setting_msgs_proto_rawDescData)
	})
	return file_user_notification_setting_msgs_proto_rawDescData
}

var file_user_notification_setting_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_user_notification_setting_msgs_proto_goTypes = []interface{}{
	(*UserNotificationSettingsReq)(nil),       // 0: UserNotificationSettingsReq
	(*UserNotificationSettingsResp)(nil),      // 1: UserNotificationSettingsResp
	(*UserNotificationSettingsUpd)(nil),       // 2: UserNotificationSettingsUpd
	(*UserNotificationSettingsWriteReq)(nil),  // 3: UserNotificationSettingsWriteReq
	(*UserNotificationSettingsWriteResp)(nil), // 4: UserNotificationSettingsWriteResp
	(*UserNotificationSettings)(nil),          // 5: UserNotificationSettings
}
var file_user_notification_setting_msgs_proto_depIdxs = []int32{
	5, // 0: UserNotificationSettingsResp.notifications:type_name -> UserNotificationSettings
	5, // 1: UserNotificationSettingsUpd.notifications:type_name -> UserNotificationSettings
	5, // 2: UserNotificationSettingsWriteReq.notifications:type_name -> UserNotificationSettings
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_user_notification_setting_msgs_proto_init() }
func file_user_notification_setting_msgs_proto_init() {
	if File_user_notification_setting_msgs_proto != nil {
		return
	}
	file_user_notification_settings_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_user_notification_setting_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserNotificationSettingsReq); i {
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
		file_user_notification_setting_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserNotificationSettingsResp); i {
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
		file_user_notification_setting_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserNotificationSettingsUpd); i {
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
		file_user_notification_setting_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserNotificationSettingsWriteReq); i {
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
		file_user_notification_setting_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserNotificationSettingsWriteResp); i {
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
			RawDescriptor: file_user_notification_setting_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_user_notification_setting_msgs_proto_goTypes,
		DependencyIndexes: file_user_notification_setting_msgs_proto_depIdxs,
		MessageInfos:      file_user_notification_setting_msgs_proto_msgTypes,
	}.Build()
	File_user_notification_setting_msgs_proto = out.File
	file_user_notification_setting_msgs_proto_rawDesc = nil
	file_user_notification_setting_msgs_proto_goTypes = nil
	file_user_notification_setting_msgs_proto_depIdxs = nil
}
