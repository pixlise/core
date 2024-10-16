// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.19.4
// source: quantification-multi-msgs.proto

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

// requires(QUANTIFY)
type QuantCombineReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId      string              `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
	RoiZStack   []*QuantCombineItem `protobuf:"bytes,2,rep,name=roiZStack,proto3" json:"roiZStack,omitempty"`
	Name        string              `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Description string              `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	SummaryOnly bool                `protobuf:"varint,5,opt,name=summaryOnly,proto3" json:"summaryOnly,omitempty"`
}

func (x *QuantCombineReq) Reset() {
	*x = QuantCombineReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineReq) ProtoMessage() {}

func (x *QuantCombineReq) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineReq.ProtoReflect.Descriptor instead.
func (*QuantCombineReq) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{0}
}

func (x *QuantCombineReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *QuantCombineReq) GetRoiZStack() []*QuantCombineItem {
	if x != nil {
		return x.RoiZStack
	}
	return nil
}

func (x *QuantCombineReq) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *QuantCombineReq) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *QuantCombineReq) GetSummaryOnly() bool {
	if x != nil {
		return x.SummaryOnly
	}
	return false
}

type QuantCombineResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to CombineResult:
	//
	//	*QuantCombineResp_JobId
	//	*QuantCombineResp_Summary
	CombineResult isQuantCombineResp_CombineResult `protobuf_oneof:"CombineResult"`
}

func (x *QuantCombineResp) Reset() {
	*x = QuantCombineResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineResp) ProtoMessage() {}

func (x *QuantCombineResp) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineResp.ProtoReflect.Descriptor instead.
func (*QuantCombineResp) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{1}
}

func (m *QuantCombineResp) GetCombineResult() isQuantCombineResp_CombineResult {
	if m != nil {
		return m.CombineResult
	}
	return nil
}

func (x *QuantCombineResp) GetJobId() string {
	if x, ok := x.GetCombineResult().(*QuantCombineResp_JobId); ok {
		return x.JobId
	}
	return ""
}

func (x *QuantCombineResp) GetSummary() *QuantCombineSummary {
	if x, ok := x.GetCombineResult().(*QuantCombineResp_Summary); ok {
		return x.Summary
	}
	return nil
}

type isQuantCombineResp_CombineResult interface {
	isQuantCombineResp_CombineResult()
}

type QuantCombineResp_JobId struct {
	JobId string `protobuf:"bytes,1,opt,name=jobId,proto3,oneof"`
}

type QuantCombineResp_Summary struct {
	Summary *QuantCombineSummary `protobuf:"bytes,2,opt,name=summary,proto3,oneof"`
}

func (*QuantCombineResp_JobId) isQuantCombineResp_CombineResult() {}

func (*QuantCombineResp_Summary) isQuantCombineResp_CombineResult() {}

// requires(QUANTIFY)
type QuantCombineListGetReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
}

func (x *QuantCombineListGetReq) Reset() {
	*x = QuantCombineListGetReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineListGetReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineListGetReq) ProtoMessage() {}

func (x *QuantCombineListGetReq) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineListGetReq.ProtoReflect.Descriptor instead.
func (*QuantCombineListGetReq) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{2}
}

func (x *QuantCombineListGetReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

type QuantCombineListGetResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	List *QuantCombineItemList `protobuf:"bytes,1,opt,name=list,proto3" json:"list,omitempty"`
}

func (x *QuantCombineListGetResp) Reset() {
	*x = QuantCombineListGetResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineListGetResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineListGetResp) ProtoMessage() {}

func (x *QuantCombineListGetResp) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineListGetResp.ProtoReflect.Descriptor instead.
func (*QuantCombineListGetResp) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{3}
}

func (x *QuantCombineListGetResp) GetList() *QuantCombineItemList {
	if x != nil {
		return x.List
	}
	return nil
}

// requires(QUANTIFY)
type QuantCombineListWriteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId string                `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
	List   *QuantCombineItemList `protobuf:"bytes,2,opt,name=list,proto3" json:"list,omitempty"`
}

func (x *QuantCombineListWriteReq) Reset() {
	*x = QuantCombineListWriteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineListWriteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineListWriteReq) ProtoMessage() {}

func (x *QuantCombineListWriteReq) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineListWriteReq.ProtoReflect.Descriptor instead.
func (*QuantCombineListWriteReq) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{4}
}

func (x *QuantCombineListWriteReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *QuantCombineListWriteReq) GetList() *QuantCombineItemList {
	if x != nil {
		return x.List
	}
	return nil
}

type QuantCombineListWriteResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *QuantCombineListWriteResp) Reset() {
	*x = QuantCombineListWriteResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuantCombineListWriteResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuantCombineListWriteResp) ProtoMessage() {}

func (x *QuantCombineListWriteResp) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuantCombineListWriteResp.ProtoReflect.Descriptor instead.
func (*QuantCombineListWriteResp) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{5}
}

// requires(QUANTIFY)
type MultiQuantCompareReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScanId              string   `protobuf:"bytes,1,opt,name=scanId,proto3" json:"scanId,omitempty"`
	ReqRoiId            string   `protobuf:"bytes,2,opt,name=reqRoiId,proto3" json:"reqRoiId,omitempty"`
	QuantIds            []string `protobuf:"bytes,3,rep,name=quantIds,proto3" json:"quantIds,omitempty"`
	RemainingPointsPMCs []int32  `protobuf:"varint,4,rep,packed,name=remainingPointsPMCs,proto3" json:"remainingPointsPMCs,omitempty"`
}

func (x *MultiQuantCompareReq) Reset() {
	*x = MultiQuantCompareReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MultiQuantCompareReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MultiQuantCompareReq) ProtoMessage() {}

func (x *MultiQuantCompareReq) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MultiQuantCompareReq.ProtoReflect.Descriptor instead.
func (*MultiQuantCompareReq) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{6}
}

func (x *MultiQuantCompareReq) GetScanId() string {
	if x != nil {
		return x.ScanId
	}
	return ""
}

func (x *MultiQuantCompareReq) GetReqRoiId() string {
	if x != nil {
		return x.ReqRoiId
	}
	return ""
}

func (x *MultiQuantCompareReq) GetQuantIds() []string {
	if x != nil {
		return x.QuantIds
	}
	return nil
}

func (x *MultiQuantCompareReq) GetRemainingPointsPMCs() []int32 {
	if x != nil {
		return x.RemainingPointsPMCs
	}
	return nil
}

type MultiQuantCompareResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RoiId       string                  `protobuf:"bytes,1,opt,name=roiId,proto3" json:"roiId,omitempty"`
	QuantTables []*QuantComparisonTable `protobuf:"bytes,2,rep,name=quantTables,proto3" json:"quantTables,omitempty"`
}

func (x *MultiQuantCompareResp) Reset() {
	*x = MultiQuantCompareResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_quantification_multi_msgs_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MultiQuantCompareResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MultiQuantCompareResp) ProtoMessage() {}

func (x *MultiQuantCompareResp) ProtoReflect() protoreflect.Message {
	mi := &file_quantification_multi_msgs_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MultiQuantCompareResp.ProtoReflect.Descriptor instead.
func (*MultiQuantCompareResp) Descriptor() ([]byte, []int) {
	return file_quantification_multi_msgs_proto_rawDescGZIP(), []int{7}
}

func (x *MultiQuantCompareResp) GetRoiId() string {
	if x != nil {
		return x.RoiId
	}
	return ""
}

func (x *MultiQuantCompareResp) GetQuantTables() []*QuantComparisonTable {
	if x != nil {
		return x.QuantTables
	}
	return nil
}

var File_quantification_multi_msgs_proto protoreflect.FileDescriptor

var file_quantification_multi_msgs_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2d, 0x6d, 0x73, 0x67, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1a, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb2, 0x01,
	0x0a, 0x0f, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x52, 0x65,
	0x71, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x12, 0x2f, 0x0a, 0x09, 0x72, 0x6f, 0x69,
	0x5a, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x51,
	0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x52,
	0x09, 0x72, 0x6f, 0x69, 0x5a, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20,
	0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x20, 0x0a, 0x0b, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x4f, 0x6e, 0x6c, 0x79, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x4f, 0x6e,
	0x6c, 0x79, 0x22, 0x6d, 0x0a, 0x10, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69,
	0x6e, 0x65, 0x52, 0x65, 0x73, 0x70, 0x12, 0x16, 0x0a, 0x05, 0x6a, 0x6f, 0x62, 0x49, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x05, 0x6a, 0x6f, 0x62, 0x49, 0x64, 0x12, 0x30,
	0x0a, 0x07, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x53, 0x75,
	0x6d, 0x6d, 0x61, 0x72, 0x79, 0x48, 0x00, 0x52, 0x07, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79,
	0x42, 0x0f, 0x0a, 0x0d, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x73, 0x75, 0x6c,
	0x74, 0x22, 0x30, 0x0a, 0x16, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e,
	0x65, 0x4c, 0x69, 0x73, 0x74, 0x47, 0x65, 0x74, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06, 0x73,
	0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61,
	0x6e, 0x49, 0x64, 0x22, 0x44, 0x0a, 0x17, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62,
	0x69, 0x6e, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x12, 0x29,
	0x0a, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x51,
	0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x22, 0x5d, 0x0a, 0x18, 0x51, 0x75, 0x61,
	0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x57, 0x72, 0x69,
	0x74, 0x65, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x12, 0x29, 0x0a,
	0x04, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x51, 0x75,
	0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x49, 0x74, 0x65, 0x6d, 0x4c, 0x69,
	0x73, 0x74, 0x52, 0x04, 0x6c, 0x69, 0x73, 0x74, 0x22, 0x1b, 0x0a, 0x19, 0x51, 0x75, 0x61, 0x6e,
	0x74, 0x43, 0x6f, 0x6d, 0x62, 0x69, 0x6e, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x57, 0x72, 0x69, 0x74,
	0x65, 0x52, 0x65, 0x73, 0x70, 0x22, 0x98, 0x01, 0x0a, 0x14, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x51,
	0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x70, 0x61, 0x72, 0x65, 0x52, 0x65, 0x71, 0x12, 0x16,
	0x0a, 0x06, 0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x73, 0x63, 0x61, 0x6e, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x71, 0x52, 0x6f, 0x69,
	0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x71, 0x52, 0x6f, 0x69,
	0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x73, 0x12, 0x30,
	0x0a, 0x13, 0x72, 0x65, 0x6d, 0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x69, 0x6e, 0x74,
	0x73, 0x50, 0x4d, 0x43, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x05, 0x52, 0x13, 0x72, 0x65, 0x6d,
	0x61, 0x69, 0x6e, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x50, 0x4d, 0x43, 0x73,
	0x22, 0x66, 0x0a, 0x15, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f,
	0x6d, 0x70, 0x61, 0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x6f, 0x69,
	0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x72, 0x6f, 0x69, 0x49, 0x64, 0x12,
	0x37, 0x0a, 0x0b, 0x71, 0x75, 0x61, 0x6e, 0x74, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x51, 0x75, 0x61, 0x6e, 0x74, 0x43, 0x6f, 0x6d, 0x70,
	0x61, 0x72, 0x69, 0x73, 0x6f, 0x6e, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x0b, 0x71, 0x75, 0x61,
	0x6e, 0x74, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x3b, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_quantification_multi_msgs_proto_rawDescOnce sync.Once
	file_quantification_multi_msgs_proto_rawDescData = file_quantification_multi_msgs_proto_rawDesc
)

func file_quantification_multi_msgs_proto_rawDescGZIP() []byte {
	file_quantification_multi_msgs_proto_rawDescOnce.Do(func() {
		file_quantification_multi_msgs_proto_rawDescData = protoimpl.X.CompressGZIP(file_quantification_multi_msgs_proto_rawDescData)
	})
	return file_quantification_multi_msgs_proto_rawDescData
}

var file_quantification_multi_msgs_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_quantification_multi_msgs_proto_goTypes = []interface{}{
	(*QuantCombineReq)(nil),           // 0: QuantCombineReq
	(*QuantCombineResp)(nil),          // 1: QuantCombineResp
	(*QuantCombineListGetReq)(nil),    // 2: QuantCombineListGetReq
	(*QuantCombineListGetResp)(nil),   // 3: QuantCombineListGetResp
	(*QuantCombineListWriteReq)(nil),  // 4: QuantCombineListWriteReq
	(*QuantCombineListWriteResp)(nil), // 5: QuantCombineListWriteResp
	(*MultiQuantCompareReq)(nil),      // 6: MultiQuantCompareReq
	(*MultiQuantCompareResp)(nil),     // 7: MultiQuantCompareResp
	(*QuantCombineItem)(nil),          // 8: QuantCombineItem
	(*QuantCombineSummary)(nil),       // 9: QuantCombineSummary
	(*QuantCombineItemList)(nil),      // 10: QuantCombineItemList
	(*QuantComparisonTable)(nil),      // 11: QuantComparisonTable
}
var file_quantification_multi_msgs_proto_depIdxs = []int32{
	8,  // 0: QuantCombineReq.roiZStack:type_name -> QuantCombineItem
	9,  // 1: QuantCombineResp.summary:type_name -> QuantCombineSummary
	10, // 2: QuantCombineListGetResp.list:type_name -> QuantCombineItemList
	10, // 3: QuantCombineListWriteReq.list:type_name -> QuantCombineItemList
	11, // 4: MultiQuantCompareResp.quantTables:type_name -> QuantComparisonTable
	5,  // [5:5] is the sub-list for method output_type
	5,  // [5:5] is the sub-list for method input_type
	5,  // [5:5] is the sub-list for extension type_name
	5,  // [5:5] is the sub-list for extension extendee
	0,  // [0:5] is the sub-list for field type_name
}

func init() { file_quantification_multi_msgs_proto_init() }
func file_quantification_multi_msgs_proto_init() {
	if File_quantification_multi_msgs_proto != nil {
		return
	}
	file_quantification_multi_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_quantification_multi_msgs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineReq); i {
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
		file_quantification_multi_msgs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineResp); i {
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
		file_quantification_multi_msgs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineListGetReq); i {
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
		file_quantification_multi_msgs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineListGetResp); i {
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
		file_quantification_multi_msgs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineListWriteReq); i {
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
		file_quantification_multi_msgs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuantCombineListWriteResp); i {
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
		file_quantification_multi_msgs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MultiQuantCompareReq); i {
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
		file_quantification_multi_msgs_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MultiQuantCompareResp); i {
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
	file_quantification_multi_msgs_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*QuantCombineResp_JobId)(nil),
		(*QuantCombineResp_Summary)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_quantification_multi_msgs_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_quantification_multi_msgs_proto_goTypes,
		DependencyIndexes: file_quantification_multi_msgs_proto_depIdxs,
		MessageInfos:      file_quantification_multi_msgs_proto_msgTypes,
	}.Build()
	File_quantification_multi_msgs_proto = out.File
	file_quantification_multi_msgs_proto_rawDesc = nil
	file_quantification_multi_msgs_proto_goTypes = nil
	file_quantification_multi_msgs_proto_depIdxs = nil
}
