// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: scan-msgs.proto

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

// Allows listing scans. Contains search fields. If these are all blank
// all scans are returned
// requires(NONE)
type ScanListReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Allows flexible fields, because scans have flexible metadata
	// but for PIXL suggestions are:
	// - driveId, drive
	// - siteId, site,
	// - targetId, target
	// - sol
	// - RTT (round-trip token)
	// - SCLK
	// - hasDwell
	// - hasNormal
	// Others (generic):
	// - title
	// - description
	// - instrument
	// - timeStampUnixSec
	SearchFilters map[string]string `protobuf:"bytes,1,rep,name=searchFilters,proto3" json:"searchFilters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Allows specifying limits around meta values, such as in PIXL's
	// case, we would allow:
	// - sol
	// - RTT
	// - SCLK
	// - PMCs
	// Others (generic):
	// - timeStampUnixSec
	// (Otherwise use exact matching in searchFilters)
	SearchMinMaxFilters map[string]*ScanListReq_MinMaxInt `protobuf:"bytes,2,rep,name=searchMinMaxFilters,proto3" json:"searchMinMaxFilters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ScanListReq) Reset() {
	*x = ScanListReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanListReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanListReq) ProtoMessage() {}

func (x *ScanListReq) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanListReq.ProtoReflect.Descriptor instead.
func (*ScanListReq) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{0}
}

func (x *ScanListReq) GetSearchFilters() map[string]string {
	if x != nil {
		return x.SearchFilters
	}
	return nil
}

func (x *ScanListReq) GetSearchMinMaxFilters() map[string]*ScanListReq_MinMaxInt {
	if x != nil {
		return x.SearchMinMaxFilters
	}
	return nil
}

type ScanListResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Scans []*ScanItem `protobuf:"bytes,1,rep,name=scans,proto3" json:"scans,omitempty"`
}

func (x *ScanListResp) Reset() {
	*x = ScanListResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanListResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanListResp) ProtoMessage() {}

func (x *ScanListResp) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanListResp.ProtoReflect.Descriptor instead.
func (*ScanListResp) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{1}
}

func (x *ScanListResp) GetScans() []*ScanItem {
	if x != nil {
		return x.Scans
	}
	return nil
}

type ScanListUpd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ScanListUpd) Reset() {
	*x = ScanListUpd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanListUpd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanListUpd) ProtoMessage() {}

func (x *ScanListUpd) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanListUpd.ProtoReflect.Descriptor instead.
func (*ScanListUpd) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{2}
}

// This should trigger a ScanListUpd to go out
// requires(EDIT_SCAN)
type ScanUploadReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Format     string `protobuf:"bytes,2,opt,name=format,proto3" json:"format,omitempty"`         // currently only allows jpl-breadboard
	ZippedData []byte `protobuf:"bytes,3,opt,name=zippedData,proto3" json:"zippedData,omitempty"` // jpl-breadboard implies this is a zip file of MSA files
}

func (x *ScanUploadReq) Reset() {
	*x = ScanUploadReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanUploadReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanUploadReq) ProtoMessage() {}

func (x *ScanUploadReq) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanUploadReq.ProtoReflect.Descriptor instead.
func (*ScanUploadReq) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *ScanUploadReq) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ScanUploadReq) GetFormat() string {
	if x != nil {
		return x.Format
	}
	return ""
}

func (x *ScanUploadReq) GetZippedData() []byte {
	if x != nil {
		return x.ZippedData
	}
	return nil
}

type ScanUploadResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ScanUploadResp) Reset() {
	*x = ScanUploadResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanUploadResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanUploadResp) ProtoMessage() {}

func (x *ScanUploadResp) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanUploadResp.ProtoReflect.Descriptor instead.
func (*ScanUploadResp) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{4}
}

// This should trigger a ScanListUpd to go out
// requires(EDIT_SCAN)
type ScanMetaWriteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId      string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
	Title       string `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"` //map<string, string> metaFields = 4;
}

func (x *ScanMetaWriteReq) Reset() {
	*x = ScanMetaWriteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanMetaWriteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanMetaWriteReq) ProtoMessage() {}

func (x *ScanMetaWriteReq) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanMetaWriteReq.ProtoReflect.Descriptor instead.
func (*ScanMetaWriteReq) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{5}
}

func (x *ScanMetaWriteReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *ScanMetaWriteReq) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *ScanMetaWriteReq) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

type ScanMetaWriteResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ScanMetaWriteResp) Reset() {
	*x = ScanMetaWriteResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanMetaWriteResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanMetaWriteResp) ProtoMessage() {}

func (x *ScanMetaWriteResp) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanMetaWriteResp.ProtoReflect.Descriptor instead.
func (*ScanMetaWriteResp) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{6}
}

// Triggering a re-import, should publish a ScanListUpd to go out
// Useful really only if there is a pipeline hooked up for this kind of data that we
// can re-trigger for. If it's a user-uploaded scan, we can't do anything really...
// requires(EDIT_SCAN)
type ScanTriggerReImportReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
}

func (x *ScanTriggerReImportReq) Reset() {
	*x = ScanTriggerReImportReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanTriggerReImportReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanTriggerReImportReq) ProtoMessage() {}

func (x *ScanTriggerReImportReq) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanTriggerReImportReq.ProtoReflect.Descriptor instead.
func (*ScanTriggerReImportReq) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{7}
}

func (x *ScanTriggerReImportReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

type ScanTriggerReImportResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ScanTriggerReImportResp) Reset() {
	*x = ScanTriggerReImportResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanTriggerReImportResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanTriggerReImportResp) ProtoMessage() {}

func (x *ScanTriggerReImportResp) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanTriggerReImportResp.ProtoReflect.Descriptor instead.
func (*ScanTriggerReImportResp) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{8}
}

// requires(NONE)
type ScanMetaLabelsReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
}

func (x *ScanMetaLabelsReq) Reset() {
	*x = ScanMetaLabelsReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanMetaLabelsReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanMetaLabelsReq) ProtoMessage() {}

func (x *ScanMetaLabelsReq) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanMetaLabelsReq.ProtoReflect.Descriptor instead.
func (*ScanMetaLabelsReq) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{9}
}

func (x *ScanMetaLabelsReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

type ScanMetaLabelsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MetaLabels []string `protobuf:"bytes,1,rep,name=metaLabels,proto3" json:"metaLabels,omitempty"`
}

func (x *ScanMetaLabelsResp) Reset() {
	*x = ScanMetaLabelsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanMetaLabelsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanMetaLabelsResp) ProtoMessage() {}

func (x *ScanMetaLabelsResp) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanMetaLabelsResp.ProtoReflect.Descriptor instead.
func (*ScanMetaLabelsResp) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{10}
}

func (x *ScanMetaLabelsResp) GetMetaLabels() []string {
	if x != nil {
		return x.MetaLabels
	}
	return nil
}

type ScanListReq_MinMaxInt struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Min int64 `protobuf:"varint,1,opt,name=min,proto3" json:"min,omitempty"`
	Max int64 `protobuf:"varint,2,opt,name=max,proto3" json:"max,omitempty"`
}

func (x *ScanListReq_MinMaxInt) Reset() {
	*x = ScanListReq_MinMaxInt{}
	if protoimpl.UnsafeEnabled {
		mi := &file_scan_msgs_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScanListReq_MinMaxInt) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScanListReq_MinMaxInt) ProtoMessage() {}

func (x *ScanListReq_MinMaxInt) ProtoReflect() protoreflect.Message {
	mi := &file_scan_msgs_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScanListReq_MinMaxInt.ProtoReflect.Descriptor instead.
func (*ScanListReq_MinMaxInt) Descriptor() ([]byte, []int) {
	return file_scan_msgs_proto_rawDescGZIP(), []int{0, 1}
}

func (x *ScanListReq_MinMaxInt) GetMin() int64 {
	if x != nil {
		return x.Min
	}
	return 0
}

func (x *ScanListReq_MinMaxInt) GetMax() int64 {
	if x != nil {
		return x.Max
	}
	return 0
}

var File_scan_msgs_proto protoreflect.FileDescriptor

var file_scan_msgs_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x73, 0x63, 0x61, 0x6e, 0x2d, 0x6d, 0x73, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x0a, 0x73, 0x63, 0x61, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x80, 0x03,
	0x0a, 0x0b, 0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x12, 0x45, 0x0a,
	0x0d, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x52,
	0x65, 0x71, 0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0d, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x46, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x12, 0x57, 0x0a, 0x13, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x4d, 0x69,
	0x6e, 0x4d, 0x61, 0x78, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x25, 0x2e, 0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x2e,
	0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x4d, 0x69, 0x6e, 0x4d, 0x61, 0x78, 0x46, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x13, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68,
	0x4d, 0x69, 0x6e, 0x4d, 0x61, 0x78, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x1a, 0x40, 0x0a,
	0x12, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a,
	0x2f, 0x0a, 0x09, 0x4d, 0x69, 0x6e, 0x4d, 0x61, 0x78, 0x49, 0x6e, 0x74, 0x12, 0x10, 0x0a, 0x03,
	0x6d, 0x69, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6d, 0x69, 0x6e, 0x12, 0x10,
	0x0a, 0x03, 0x6d, 0x61, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6d, 0x61, 0x78,
	0x1a, 0x5e, 0x0a, 0x18, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x4d, 0x69, 0x6e, 0x4d, 0x61, 0x78,
	0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x2c,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e,
	0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x2e, 0x4d, 0x69, 0x6e, 0x4d,
	0x61, 0x78, 0x49, 0x6e, 0x74, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0x2f, 0x0a, 0x0c, 0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70,
	0x12, 0x1f, 0x0a, 0x05, 0x73, 0x63, 0x61, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x09, 0x2e, 0x53, 0x63, 0x61, 0x6e, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x05, 0x73, 0x63, 0x61, 0x6e,
	0x73, 0x22, 0x0d, 0x0a, 0x0b, 0x53, 0x63, 0x61, 0x6e, 0x4c, 0x69, 0x73, 0x74, 0x55, 0x70, 0x64,
	0x22, 0x57, 0x0a, 0x0d, 0x53, 0x63, 0x61, 0x6e, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x65,
	0x71, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x16, 0x0a, 0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x7a, 0x69, 0x70,
	0x70, 0x65, 0x64, 0x44, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x7a,
	0x69, 0x70, 0x70, 0x65, 0x64, 0x44, 0x61, 0x74, 0x61, 0x22, 0x10, 0x0a, 0x0e, 0x53, 0x63, 0x61,
	0x6e, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x22, 0x62, 0x0a, 0x10, 0x53,
	0x63, 0x61, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12,
	0x16, 0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x12, 0x20, 0x0a,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22,
	0x13, 0x0a, 0x11, 0x53, 0x63, 0x61, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x57, 0x72, 0x69, 0x74, 0x65,
	0x52, 0x65, 0x73, 0x70, 0x22, 0x30, 0x0a, 0x16, 0x53, 0x63, 0x61, 0x6e, 0x54, 0x72, 0x69, 0x67,
	0x67, 0x65, 0x72, 0x52, 0x65, 0x49, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x71, 0x12, 0x16,
	0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x22, 0x19, 0x0a, 0x17, 0x53, 0x63, 0x61, 0x6e, 0x54, 0x72,
	0x69, 0x67, 0x67, 0x65, 0x72, 0x52, 0x65, 0x49, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x22, 0x2b, 0x0a, 0x11, 0x53, 0x63, 0x61, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x22, 0x34,
	0x0a, 0x12, 0x53, 0x63, 0x61, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x12, 0x1e, 0x0a, 0x0a, 0x6d, 0x65, 0x74, 0x61, 0x4c, 0x61, 0x62, 0x65,
	0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x6d, 0x65, 0x74, 0x61, 0x4c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_scan_msgs_proto_rawDescOnce sync.Once
	file_scan_msgs_proto_rawDescData = file_scan_msgs_proto_rawDesc
)

func file_scan_msgs_proto_rawDescGZIP() []byte {
	file_scan_msgs_proto_rawDescOnce.Do(func() {
		file_scan_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_scan_msgs_proto_rawDescData)
	})
	return file_scan_msgs_proto_rawDescData
}

var file_scan_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 14)
var file_scan_msgs_proto_goTypes = []interface{}{
	(*ScanListReq)(nil),             // 0: ScanListReq
	(*ScanListResp)(nil),            // 1: ScanListResp
	(*ScanListUpd)(nil),             // 2: ScanListUpd
	(*ScanUploadReq)(nil),           // 3: ScanUploadReq
	(*ScanUploadResp)(nil),          // 4: ScanUploadResp
	(*ScanMetaWriteReq)(nil),        // 5: ScanMetaWriteReq
	(*ScanMetaWriteResp)(nil),       // 6: ScanMetaWriteResp
	(*ScanTriggerReImportReq)(nil),  // 7: ScanTriggerReImportReq
	(*ScanTriggerReImportResp)(nil), // 8: ScanTriggerReImportResp
	(*ScanMetaLabelsReq)(nil),       // 9: ScanMetaLabelsReq
	(*ScanMetaLabelsResp)(nil),      // 10: ScanMetaLabelsResp
	nil,                             // 11: ScanListReq.SearchFiltersEntry
	(*ScanListReq_MinMaxInt)(nil),   // 12: ScanListReq.MinMaxInt
	nil,                             // 13: ScanListReq.SearchMinMaxFiltersEntry
	(*ScanItem)(nil),                // 14: ScanItem
}
var file_scan_msgs_proto_depIdxs = []int32{
	11, // 0: ScanListReq.searchFilters:type_name -> ScanListReq.SearchFiltersEntry
	13, // 1: ScanListReq.searchMinMaxFilters:type_name -> ScanListReq.SearchMinMaxFiltersEntry
	14, // 2: ScanListResp.scans:type_name -> ScanItem
	12, // 3: ScanListReq.SearchMinMaxFiltersEntry.value:type_name -> ScanListReq.MinMaxInt
	4,  // [4:4] is the sub-list for method output_type
	4,  // [4:4] is the sub-list for method input_type
	4,  // [4:4] is the sub-list for extension type_name
	4,  // [4:4] is the sub-list for extension extendee
	0,  // [0:4] is the sub-list for field type_name
}

func init() { file_scan_msgs_proto_init() }
func file_scan_msgs_proto_init() {
	if File_scan_msgs_proto != nil {
		return
	}
	file_scan_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_scan_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanListReq); i {
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
		file_scan_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanListResp); i {
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
		file_scan_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanListUpd); i {
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
		file_scan_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanUploadReq); i {
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
		file_scan_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanUploadResp); i {
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
		file_scan_msgs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanMetaWriteReq); i {
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
		file_scan_msgs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanMetaWriteResp); i {
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
		file_scan_msgs_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanTriggerReImportReq); i {
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
		file_scan_msgs_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanTriggerReImportResp); i {
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
		file_scan_msgs_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanMetaLabelsReq); i {
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
		file_scan_msgs_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanMetaLabelsResp); i {
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
		file_scan_msgs_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScanListReq_MinMaxInt); i {
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
			RawDescriptor: file_scan_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   14,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_scan_msgs_proto_goTypes,
		DependencyIndexes: file_scan_msgs_proto_depIdxs,
		MessageInfos:      file_scan_msgs_proto_msgTypes,
	}.Build()
	File_scan_msgs_proto = out.File
	file_scan_msgs_proto_rawDesc = nil
	file_scan_msgs_proto_goTypes = nil
	file_scan_msgs_proto_depIdxs = nil
}