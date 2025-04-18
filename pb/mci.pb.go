// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.30.1
// source: mci.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type JobStatus int32

const (
	JobStatus_PENDING     JobStatus = 0
	JobStatus_SUCCESS     JobStatus = 1
	JobStatus_FAILURE     JobStatus = 2
	JobStatus_IN_PROGRESS JobStatus = 3
	JobStatus_ERROR       JobStatus = 4
)

// Enum value maps for JobStatus.
var (
	JobStatus_name = map[int32]string{
		0: "PENDING",
		1: "SUCCESS",
		2: "FAILURE",
		3: "IN_PROGRESS",
		4: "ERROR",
	}
	JobStatus_value = map[string]int32{
		"PENDING":     0,
		"SUCCESS":     1,
		"FAILURE":     2,
		"IN_PROGRESS": 3,
		"ERROR":       4,
	}
)

func (x JobStatus) Enum() *JobStatus {
	p := new(JobStatus)
	*p = x
	return p
}

func (x JobStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (JobStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_mci_proto_enumTypes[0].Descriptor()
}

func (JobStatus) Type() protoreflect.EnumType {
	return &file_mci_proto_enumTypes[0]
}

func (x JobStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use JobStatus.Descriptor instead.
func (JobStatus) EnumDescriptor() ([]byte, []int) {
	return file_mci_proto_rawDescGZIP(), []int{0}
}

type Job struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	RepoUrl       string                 `protobuf:"bytes,1,opt,name=repo_url,json=repoUrl,proto3" json:"repo_url,omitempty"`
	CommitSha     string                 `protobuf:"bytes,2,opt,name=commit_sha,json=commitSha,proto3" json:"commit_sha,omitempty"`
	Id            string                 `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Job) Reset() {
	*x = Job{}
	mi := &file_mci_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Job) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Job) ProtoMessage() {}

func (x *Job) ProtoReflect() protoreflect.Message {
	mi := &file_mci_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Job.ProtoReflect.Descriptor instead.
func (*Job) Descriptor() ([]byte, []int) {
	return file_mci_proto_rawDescGZIP(), []int{0}
}

func (x *Job) GetRepoUrl() string {
	if x != nil {
		return x.RepoUrl
	}
	return ""
}

func (x *Job) GetCommitSha() string {
	if x != nil {
		return x.CommitSha
	}
	return ""
}

func (x *Job) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Log struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Log) Reset() {
	*x = Log{}
	mi := &file_mci_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Log) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Log) ProtoMessage() {}

func (x *Log) ProtoReflect() protoreflect.Message {
	mi := &file_mci_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Log.ProtoReflect.Descriptor instead.
func (*Log) Descriptor() ([]byte, []int) {
	return file_mci_proto_rawDescGZIP(), []int{1}
}

func (x *Log) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type Empty struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Empty) Reset() {
	*x = Empty{}
	mi := &file_mci_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_mci_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_mci_proto_rawDescGZIP(), []int{2}
}

var File_mci_proto protoreflect.FileDescriptor

const file_mci_proto_rawDesc = "" +
	"\n" +
	"\tmci.proto\"O\n" +
	"\x03Job\x12\x19\n" +
	"\brepo_url\x18\x01 \x01(\tR\arepoUrl\x12\x1d\n" +
	"\n" +
	"commit_sha\x18\x02 \x01(\tR\tcommitSha\x12\x0e\n" +
	"\x02id\x18\x03 \x01(\tR\x02id\"\x1f\n" +
	"\x03Log\x12\x18\n" +
	"\amessage\x18\x01 \x01(\tR\amessage\"\a\n" +
	"\x05Empty*N\n" +
	"\tJobStatus\x12\v\n" +
	"\aPENDING\x10\x00\x12\v\n" +
	"\aSUCCESS\x10\x01\x12\v\n" +
	"\aFAILURE\x10\x02\x12\x0f\n" +
	"\vIN_PROGRESS\x10\x03\x12\t\n" +
	"\x05ERROR\x10\x042%\n" +
	"\x05Agent\x12\x1c\n" +
	"\n" +
	"ExecuteJob\x12\x04.Job\x1a\x06.Empty\"\x002&\n" +
	"\x04Core\x12\x1e\n" +
	"\n" +
	"StreamLogs\x12\x04.Log\x1a\x06.Empty\"\x00(\x01B\x19Z\x17github.com/nireo/mci/pbb\x06proto3"

var (
	file_mci_proto_rawDescOnce sync.Once
	file_mci_proto_rawDescData []byte
)

func file_mci_proto_rawDescGZIP() []byte {
	file_mci_proto_rawDescOnce.Do(func() {
		file_mci_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_mci_proto_rawDesc), len(file_mci_proto_rawDesc)))
	})
	return file_mci_proto_rawDescData
}

var file_mci_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_mci_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_mci_proto_goTypes = []any{
	(JobStatus)(0), // 0: JobStatus
	(*Job)(nil),    // 1: Job
	(*Log)(nil),    // 2: Log
	(*Empty)(nil),  // 3: Empty
}
var file_mci_proto_depIdxs = []int32{
	1, // 0: Agent.ExecuteJob:input_type -> Job
	2, // 1: Core.StreamLogs:input_type -> Log
	3, // 2: Agent.ExecuteJob:output_type -> Empty
	3, // 3: Core.StreamLogs:output_type -> Empty
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_mci_proto_init() }
func file_mci_proto_init() {
	if File_mci_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_mci_proto_rawDesc), len(file_mci_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_mci_proto_goTypes,
		DependencyIndexes: file_mci_proto_depIdxs,
		EnumInfos:         file_mci_proto_enumTypes,
		MessageInfos:      file_mci_proto_msgTypes,
	}.Build()
	File_mci_proto = out.File
	file_mci_proto_goTypes = nil
	file_mci_proto_depIdxs = nil
}
