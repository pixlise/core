// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: user-group-membership-msgs.proto

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

// //////////////////////////////////////////////////////////////
// Adding and deleting members from the group
// Should only be accessible to group admins and sys admins
// requires(NONE)
type UserGroupAddMemberReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GroupId string `protobuf:"bytes,1,opt,name=groupId,proto3" json:"groupId,omitempty"`
	// Can add a group or a user id to this
	//
	// Types that are assignable to Member:
	//
	//	*UserGroupAddMemberReq_GroupMemberId
	//	*UserGroupAddMemberReq_UserMemberId
	Member isUserGroupAddMemberReq_Member `protobuf_oneof:"Member"`
}

func (x *UserGroupAddMemberReq) Reset() {
	*x = UserGroupAddMemberReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupAddMemberReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupAddMemberReq) ProtoMessage() {}

func (x *UserGroupAddMemberReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupAddMemberReq.ProtoReflect.Descriptor instead.
func (*UserGroupAddMemberReq) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{0}
}

func (x *UserGroupAddMemberReq) GetGroupId() string {
	if x != nil {
		return x.GroupId
	}
	return ""
}

func (m *UserGroupAddMemberReq) GetMember() isUserGroupAddMemberReq_Member {
	if m != nil {
		return m.Member
	}
	return nil
}

func (x *UserGroupAddMemberReq) GetGroupMemberId() string {
	if x, ok := x.GetMember().(*UserGroupAddMemberReq_GroupMemberId); ok {
		return x.GroupMemberId
	}
	return ""
}

func (x *UserGroupAddMemberReq) GetUserMemberId() string {
	if x, ok := x.GetMember().(*UserGroupAddMemberReq_UserMemberId); ok {
		return x.UserMemberId
	}
	return ""
}

type isUserGroupAddMemberReq_Member interface {
	isUserGroupAddMemberReq_Member()
}

type UserGroupAddMemberReq_GroupMemberId struct {
	GroupMemberId string `protobuf:"bytes,2,opt,name=groupMemberId,proto3,oneof"`
}

type UserGroupAddMemberReq_UserMemberId struct {
	UserMemberId string `protobuf:"bytes,3,opt,name=userMemberId,proto3,oneof"`
}

func (*UserGroupAddMemberReq_GroupMemberId) isUserGroupAddMemberReq_Member() {}

func (*UserGroupAddMemberReq_UserMemberId) isUserGroupAddMemberReq_Member() {}

type UserGroupAddMemberResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Group *UserGroup `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
}

func (x *UserGroupAddMemberResp) Reset() {
	*x = UserGroupAddMemberResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupAddMemberResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupAddMemberResp) ProtoMessage() {}

func (x *UserGroupAddMemberResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupAddMemberResp.ProtoReflect.Descriptor instead.
func (*UserGroupAddMemberResp) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *UserGroupAddMemberResp) GetGroup() *UserGroup {
	if x != nil {
		return x.Group
	}
	return nil
}

// requires(NONE)
type UserGroupDeleteMemberReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GroupId string `protobuf:"bytes,1,opt,name=groupId,proto3" json:"groupId,omitempty"`
	// Can delete a group or a user id from this
	//
	// Types that are assignable to Member:
	//
	//	*UserGroupDeleteMemberReq_GroupMemberId
	//	*UserGroupDeleteMemberReq_UserMemberId
	Member isUserGroupDeleteMemberReq_Member `protobuf_oneof:"Member"`
}

func (x *UserGroupDeleteMemberReq) Reset() {
	*x = UserGroupDeleteMemberReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupDeleteMemberReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupDeleteMemberReq) ProtoMessage() {}

func (x *UserGroupDeleteMemberReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupDeleteMemberReq.ProtoReflect.Descriptor instead.
func (*UserGroupDeleteMemberReq) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{2}
}

func (x *UserGroupDeleteMemberReq) GetGroupId() string {
	if x != nil {
		return x.GroupId
	}
	return ""
}

func (m *UserGroupDeleteMemberReq) GetMember() isUserGroupDeleteMemberReq_Member {
	if m != nil {
		return m.Member
	}
	return nil
}

func (x *UserGroupDeleteMemberReq) GetGroupMemberId() string {
	if x, ok := x.GetMember().(*UserGroupDeleteMemberReq_GroupMemberId); ok {
		return x.GroupMemberId
	}
	return ""
}

func (x *UserGroupDeleteMemberReq) GetUserMemberId() string {
	if x, ok := x.GetMember().(*UserGroupDeleteMemberReq_UserMemberId); ok {
		return x.UserMemberId
	}
	return ""
}

type isUserGroupDeleteMemberReq_Member interface {
	isUserGroupDeleteMemberReq_Member()
}

type UserGroupDeleteMemberReq_GroupMemberId struct {
	GroupMemberId string `protobuf:"bytes,2,opt,name=groupMemberId,proto3,oneof"`
}

type UserGroupDeleteMemberReq_UserMemberId struct {
	UserMemberId string `protobuf:"bytes,3,opt,name=userMemberId,proto3,oneof"`
}

func (*UserGroupDeleteMemberReq_GroupMemberId) isUserGroupDeleteMemberReq_Member() {}

func (*UserGroupDeleteMemberReq_UserMemberId) isUserGroupDeleteMemberReq_Member() {}

type UserGroupDeleteMemberResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Group *UserGroup `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
}

func (x *UserGroupDeleteMemberResp) Reset() {
	*x = UserGroupDeleteMemberResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupDeleteMemberResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupDeleteMemberResp) ProtoMessage() {}

func (x *UserGroupDeleteMemberResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupDeleteMemberResp.ProtoReflect.Descriptor instead.
func (*UserGroupDeleteMemberResp) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *UserGroupDeleteMemberResp) GetGroup() *UserGroup {
	if x != nil {
		return x.Group
	}
	return nil
}

// //////////////////////////////////////////////////////////////
// Adding and deleting viewers from the group
// Should only be accessible to group admins and sys admins
// requires(NONE)
type UserGroupAddViewerReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GroupId string `protobuf:"bytes,1,opt,name=groupId,proto3" json:"groupId,omitempty"`
	// Can add a group or a user id to this
	//
	// Types that are assignable to Viewer:
	//
	//	*UserGroupAddViewerReq_GroupViewerId
	//	*UserGroupAddViewerReq_UserViewerId
	Viewer isUserGroupAddViewerReq_Viewer `protobuf_oneof:"Viewer"`
}

func (x *UserGroupAddViewerReq) Reset() {
	*x = UserGroupAddViewerReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupAddViewerReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupAddViewerReq) ProtoMessage() {}

func (x *UserGroupAddViewerReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupAddViewerReq.ProtoReflect.Descriptor instead.
func (*UserGroupAddViewerReq) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{4}
}

func (x *UserGroupAddViewerReq) GetGroupId() string {
	if x != nil {
		return x.GroupId
	}
	return ""
}

func (m *UserGroupAddViewerReq) GetViewer() isUserGroupAddViewerReq_Viewer {
	if m != nil {
		return m.Viewer
	}
	return nil
}

func (x *UserGroupAddViewerReq) GetGroupViewerId() string {
	if x, ok := x.GetViewer().(*UserGroupAddViewerReq_GroupViewerId); ok {
		return x.GroupViewerId
	}
	return ""
}

func (x *UserGroupAddViewerReq) GetUserViewerId() string {
	if x, ok := x.GetViewer().(*UserGroupAddViewerReq_UserViewerId); ok {
		return x.UserViewerId
	}
	return ""
}

type isUserGroupAddViewerReq_Viewer interface {
	isUserGroupAddViewerReq_Viewer()
}

type UserGroupAddViewerReq_GroupViewerId struct {
	GroupViewerId string `protobuf:"bytes,2,opt,name=groupViewerId,proto3,oneof"`
}

type UserGroupAddViewerReq_UserViewerId struct {
	UserViewerId string `protobuf:"bytes,3,opt,name=userViewerId,proto3,oneof"`
}

func (*UserGroupAddViewerReq_GroupViewerId) isUserGroupAddViewerReq_Viewer() {}

func (*UserGroupAddViewerReq_UserViewerId) isUserGroupAddViewerReq_Viewer() {}

type UserGroupAddViewerResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Group *UserGroup `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
}

func (x *UserGroupAddViewerResp) Reset() {
	*x = UserGroupAddViewerResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupAddViewerResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupAddViewerResp) ProtoMessage() {}

func (x *UserGroupAddViewerResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupAddViewerResp.ProtoReflect.Descriptor instead.
func (*UserGroupAddViewerResp) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{5}
}

func (x *UserGroupAddViewerResp) GetGroup() *UserGroup {
	if x != nil {
		return x.Group
	}
	return nil
}

// requires(NONE)
type UserGroupDeleteViewerReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GroupId string `protobuf:"bytes,1,opt,name=groupId,proto3" json:"groupId,omitempty"`
	// Can delete a group or a user id from this
	//
	// Types that are assignable to Viewer:
	//
	//	*UserGroupDeleteViewerReq_GroupViewerId
	//	*UserGroupDeleteViewerReq_UserViewerId
	Viewer isUserGroupDeleteViewerReq_Viewer `protobuf_oneof:"Viewer"`
}

func (x *UserGroupDeleteViewerReq) Reset() {
	*x = UserGroupDeleteViewerReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupDeleteViewerReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupDeleteViewerReq) ProtoMessage() {}

func (x *UserGroupDeleteViewerReq) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupDeleteViewerReq.ProtoReflect.Descriptor instead.
func (*UserGroupDeleteViewerReq) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{6}
}

func (x *UserGroupDeleteViewerReq) GetGroupId() string {
	if x != nil {
		return x.GroupId
	}
	return ""
}

func (m *UserGroupDeleteViewerReq) GetViewer() isUserGroupDeleteViewerReq_Viewer {
	if m != nil {
		return m.Viewer
	}
	return nil
}

func (x *UserGroupDeleteViewerReq) GetGroupViewerId() string {
	if x, ok := x.GetViewer().(*UserGroupDeleteViewerReq_GroupViewerId); ok {
		return x.GroupViewerId
	}
	return ""
}

func (x *UserGroupDeleteViewerReq) GetUserViewerId() string {
	if x, ok := x.GetViewer().(*UserGroupDeleteViewerReq_UserViewerId); ok {
		return x.UserViewerId
	}
	return ""
}

type isUserGroupDeleteViewerReq_Viewer interface {
	isUserGroupDeleteViewerReq_Viewer()
}

type UserGroupDeleteViewerReq_GroupViewerId struct {
	GroupViewerId string `protobuf:"bytes,2,opt,name=groupViewerId,proto3,oneof"`
}

type UserGroupDeleteViewerReq_UserViewerId struct {
	UserViewerId string `protobuf:"bytes,3,opt,name=userViewerId,proto3,oneof"`
}

func (*UserGroupDeleteViewerReq_GroupViewerId) isUserGroupDeleteViewerReq_Viewer() {}

func (*UserGroupDeleteViewerReq_UserViewerId) isUserGroupDeleteViewerReq_Viewer() {}

type UserGroupDeleteViewerResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Group *UserGroup `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
}

func (x *UserGroupDeleteViewerResp) Reset() {
	*x = UserGroupDeleteViewerResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_user_group_membership_msgs_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UserGroupDeleteViewerResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserGroupDeleteViewerResp) ProtoMessage() {}

func (x *UserGroupDeleteViewerResp) ProtoReflect() protoreflect.Message {
	mi := &file_user_group_membership_msgs_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserGroupDeleteViewerResp.ProtoReflect.Descriptor instead.
func (*UserGroupDeleteViewerResp) Descriptor() ([]byte, []int) {
	return file_user_group_membership_msgs_proto_rawDescGZIP(), []int{7}
}

func (x *UserGroupDeleteViewerResp) GetGroup() *UserGroup {
	if x != nil {
		return x.Group
	}
	return nil
}

var File_user_group_membership_msgs_proto protoreflect.FileDescriptor

var file_user_group_membership_msgs_proto_rawDesc = []byte{
	0x0a, 0x20, 0x75, 0x73, 0x65, 0x72, 0x2d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x2d, 0x6d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x2d, 0x6d, 0x73, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x10, 0x75, 0x73, 0x65, 0x72, 0x2d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x89, 0x01, 0x0a, 0x15, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f,
	0x75, 0x70, 0x41, 0x64, 0x64, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12, 0x18,
	0x0a, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x12, 0x26, 0x0a, 0x0d, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48,
	0x00, 0x52, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x64,
	0x12, 0x24, 0x0a, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x4d, 0x65,
	0x6d, 0x62, 0x65, 0x72, 0x49, 0x64, 0x42, 0x08, 0x0a, 0x06, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72,
	0x22, 0x3a, 0x0a, 0x16, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x41, 0x64, 0x64,
	0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x12, 0x20, 0x0a, 0x05, 0x67, 0x72,
	0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x55, 0x73, 0x65, 0x72,
	0x47, 0x72, 0x6f, 0x75, 0x70, 0x52, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x22, 0x8c, 0x01, 0x0a,
	0x18, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12, 0x18, 0x0a, 0x07, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x49, 0x64, 0x12, 0x26, 0x0a, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x65, 0x6d, 0x62,
	0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0d, 0x67, 0x72,
	0x6f, 0x75, 0x70, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x64, 0x12, 0x24, 0x0a, 0x0c, 0x75,
	0x73, 0x65, 0x72, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x48, 0x00, 0x52, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49,
	0x64, 0x42, 0x08, 0x0a, 0x06, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x3d, 0x0a, 0x19, 0x55,
	0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x4d, 0x65,
	0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x12, 0x20, 0x0a, 0x05, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x52, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x22, 0x89, 0x01, 0x0a, 0x15, 0x55,
	0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x41, 0x64, 0x64, 0x56, 0x69, 0x65, 0x77, 0x65,
	0x72, 0x52, 0x65, 0x71, 0x12, 0x18, 0x0a, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x12, 0x26,
	0x0a, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x56, 0x69,
	0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x12, 0x24, 0x0a, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x56, 0x69,
	0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0c,
	0x75, 0x73, 0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x42, 0x08, 0x0a, 0x06,
	0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x22, 0x3a, 0x0a, 0x16, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x41, 0x64, 0x64, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70,
	0x12, 0x20, 0x0a, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0a, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x52, 0x05, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x22, 0x8c, 0x01, 0x0a, 0x18, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70,
	0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12,
	0x18, 0x0a, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x12, 0x26, 0x0a, 0x0d, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x48, 0x00, 0x52, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x49,
	0x64, 0x12, 0x24, 0x0a, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x49,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x56,
	0x69, 0x65, 0x77, 0x65, 0x72, 0x49, 0x64, 0x42, 0x08, 0x0a, 0x06, 0x56, 0x69, 0x65, 0x77, 0x65,
	0x72, 0x22, 0x3d, 0x0a, 0x19, 0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x56, 0x69, 0x65, 0x77, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x12, 0x20,
	0x0a, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e,
	0x55, 0x73, 0x65, 0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x52, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70,
	0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_user_group_membership_msgs_proto_rawDescOnce sync.Once
	file_user_group_membership_msgs_proto_rawDescData = file_user_group_membership_msgs_proto_rawDesc
)

func file_user_group_membership_msgs_proto_rawDescGZIP() []byte {
	file_user_group_membership_msgs_proto_rawDescOnce.Do(func() {
		file_user_group_membership_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_user_group_membership_msgs_proto_rawDescData)
	})
	return file_user_group_membership_msgs_proto_rawDescData
}

var file_user_group_membership_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_user_group_membership_msgs_proto_goTypes = []interface{}{
	(*UserGroupAddMemberReq)(nil),     // 0: UserGroupAddMemberReq
	(*UserGroupAddMemberResp)(nil),    // 1: UserGroupAddMemberResp
	(*UserGroupDeleteMemberReq)(nil),  // 2: UserGroupDeleteMemberReq
	(*UserGroupDeleteMemberResp)(nil), // 3: UserGroupDeleteMemberResp
	(*UserGroupAddViewerReq)(nil),     // 4: UserGroupAddViewerReq
	(*UserGroupAddViewerResp)(nil),    // 5: UserGroupAddViewerResp
	(*UserGroupDeleteViewerReq)(nil),  // 6: UserGroupDeleteViewerReq
	(*UserGroupDeleteViewerResp)(nil), // 7: UserGroupDeleteViewerResp
	(*UserGroup)(nil),                 // 8: UserGroup
}
var file_user_group_membership_msgs_proto_depIdxs = []int32{
	8, // 0: UserGroupAddMemberResp.group:type_name -> UserGroup
	8, // 1: UserGroupDeleteMemberResp.group:type_name -> UserGroup
	8, // 2: UserGroupAddViewerResp.group:type_name -> UserGroup
	8, // 3: UserGroupDeleteViewerResp.group:type_name -> UserGroup
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_user_group_membership_msgs_proto_init() }
func file_user_group_membership_msgs_proto_init() {
	if File_user_group_membership_msgs_proto != nil {
		return
	}
	file_user_group_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_user_group_membership_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupAddMemberReq); i {
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
		file_user_group_membership_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupAddMemberResp); i {
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
		file_user_group_membership_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupDeleteMemberReq); i {
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
		file_user_group_membership_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupDeleteMemberResp); i {
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
		file_user_group_membership_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupAddViewerReq); i {
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
		file_user_group_membership_msgs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupAddViewerResp); i {
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
		file_user_group_membership_msgs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupDeleteViewerReq); i {
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
		file_user_group_membership_msgs_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UserGroupDeleteViewerResp); i {
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
	file_user_group_membership_msgs_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*UserGroupAddMemberReq_GroupMemberId)(nil),
		(*UserGroupAddMemberReq_UserMemberId)(nil),
	}
	file_user_group_membership_msgs_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*UserGroupDeleteMemberReq_GroupMemberId)(nil),
		(*UserGroupDeleteMemberReq_UserMemberId)(nil),
	}
	file_user_group_membership_msgs_proto_msgTypes[4].OneofWrappers = []interface{}{
		(*UserGroupAddViewerReq_GroupViewerId)(nil),
		(*UserGroupAddViewerReq_UserViewerId)(nil),
	}
	file_user_group_membership_msgs_proto_msgTypes[6].OneofWrappers = []interface{}{
		(*UserGroupDeleteViewerReq_GroupViewerId)(nil),
		(*UserGroupDeleteViewerReq_UserViewerId)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_user_group_membership_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_user_group_membership_msgs_proto_goTypes,
		DependencyIndexes: file_user_group_membership_msgs_proto_depIdxs,
		MessageInfos:      file_user_group_membership_msgs_proto_msgTypes,
	}.Build()
	File_user_group_membership_msgs_proto = out.File
	file_user_group_membership_msgs_proto_rawDesc = nil
	file_user_group_membership_msgs_proto_goTypes = nil
	file_user_group_membership_msgs_proto_depIdxs = nil
}