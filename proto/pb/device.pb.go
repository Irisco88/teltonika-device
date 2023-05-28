// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        (unknown)
// source: device.proto

package pb

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

type AVLData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Imei       string       `protobuf:"bytes,1,opt,name=imei,proto3" json:"imei,omitempty"`
	Timestamp  int64        `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Gps        *GPS         `protobuf:"bytes,3,opt,name=gps,proto3" json:"gps,omitempty"`
	IoElements []*IOElement `protobuf:"bytes,4,rep,name=io_elements,json=ioElements,proto3" json:"io_elements,omitempty"`
}

func (x *AVLData) Reset() {
	*x = AVLData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_device_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AVLData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AVLData) ProtoMessage() {}

func (x *AVLData) ProtoReflect() protoreflect.Message {
	mi := &file_device_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AVLData.ProtoReflect.Descriptor instead.
func (*AVLData) Descriptor() ([]byte, []int) {
	return file_device_proto_rawDescGZIP(), []int{0}
}

func (x *AVLData) GetImei() string {
	if x != nil {
		return x.Imei
	}
	return ""
}

func (x *AVLData) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *AVLData) GetGps() *GPS {
	if x != nil {
		return x.Gps
	}
	return nil
}

func (x *AVLData) GetIoElements() []*IOElement {
	if x != nil {
		return x.IoElements
	}
	return nil
}

type GPS struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Longitude  int32 `protobuf:"varint,1,opt,name=longitude,proto3" json:"longitude,omitempty"`
	Latitude   int32 `protobuf:"varint,2,opt,name=latitude,proto3" json:"latitude,omitempty"`
	Altitude   int32 `protobuf:"varint,3,opt,name=altitude,proto3" json:"altitude,omitempty"`
	Angle      int32 `protobuf:"varint,4,opt,name=angle,proto3" json:"angle,omitempty"`
	Satellites int32 `protobuf:"varint,5,opt,name=satellites,proto3" json:"satellites,omitempty"`
	Speed      int32 `protobuf:"varint,6,opt,name=speed,proto3" json:"speed,omitempty"`
}

func (x *GPS) Reset() {
	*x = GPS{}
	if protoimpl.UnsafeEnabled {
		mi := &file_device_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GPS) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GPS) ProtoMessage() {}

func (x *GPS) ProtoReflect() protoreflect.Message {
	mi := &file_device_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GPS.ProtoReflect.Descriptor instead.
func (*GPS) Descriptor() ([]byte, []int) {
	return file_device_proto_rawDescGZIP(), []int{1}
}

func (x *GPS) GetLongitude() int32 {
	if x != nil {
		return x.Longitude
	}
	return 0
}

func (x *GPS) GetLatitude() int32 {
	if x != nil {
		return x.Latitude
	}
	return 0
}

func (x *GPS) GetAltitude() int32 {
	if x != nil {
		return x.Altitude
	}
	return 0
}

func (x *GPS) GetAngle() int32 {
	if x != nil {
		return x.Angle
	}
	return 0
}

func (x *GPS) GetSatellites() int32 {
	if x != nil {
		return x.Satellites
	}
	return 0
}

func (x *GPS) GetSpeed() int32 {
	if x != nil {
		return x.Speed
	}
	return 0
}

type IOElement struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ElementId int32 `protobuf:"varint,1,opt,name=element_id,json=elementId,proto3" json:"element_id,omitempty"`
	Value     int32 `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *IOElement) Reset() {
	*x = IOElement{}
	if protoimpl.UnsafeEnabled {
		mi := &file_device_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IOElement) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IOElement) ProtoMessage() {}

func (x *IOElement) ProtoReflect() protoreflect.Message {
	mi := &file_device_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IOElement.ProtoReflect.Descriptor instead.
func (*IOElement) Descriptor() ([]byte, []int) {
	return file_device_proto_rawDescGZIP(), []int{2}
}

func (x *IOElement) GetElementId() int32 {
	if x != nil {
		return x.ElementId
	}
	return 0
}

func (x *IOElement) GetValue() int32 {
	if x != nil {
		return x.Value
	}
	return 0
}

var File_device_proto protoreflect.FileDescriptor

var file_device_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8c, 0x01, 0x0a, 0x07, 0x41, 0x56, 0x4c, 0x44, 0x61, 0x74,
	0x61, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x6d, 0x65, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x69, 0x6d, 0x65, 0x69, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x12, 0x1c, 0x0a, 0x03, 0x67, 0x70, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x50, 0x53, 0x52, 0x03, 0x67, 0x70,
	0x73, 0x12, 0x31, 0x0a, 0x0b, 0x69, 0x6f, 0x5f, 0x65, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x49,
	0x4f, 0x45, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x0a, 0x69, 0x6f, 0x45, 0x6c, 0x65, 0x6d,
	0x65, 0x6e, 0x74, 0x73, 0x22, 0xa7, 0x01, 0x0a, 0x03, 0x47, 0x50, 0x53, 0x12, 0x1c, 0x0a, 0x09,
	0x6c, 0x6f, 0x6e, 0x67, 0x69, 0x74, 0x75, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x09, 0x6c, 0x6f, 0x6e, 0x67, 0x69, 0x74, 0x75, 0x64, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x61,
	0x74, 0x69, 0x74, 0x75, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x6c, 0x61,
	0x74, 0x69, 0x74, 0x75, 0x64, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x6c, 0x74, 0x69, 0x74, 0x75,
	0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x61, 0x6c, 0x74, 0x69, 0x74, 0x75,
	0x64, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6e, 0x67, 0x6c, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x05, 0x61, 0x6e, 0x67, 0x6c, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x61, 0x74, 0x65,
	0x6c, 0x6c, 0x69, 0x74, 0x65, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x73, 0x61,
	0x74, 0x65, 0x6c, 0x6c, 0x69, 0x74, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x70, 0x65, 0x65,
	0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x73, 0x70, 0x65, 0x65, 0x64, 0x22, 0x40,
	0x0a, 0x09, 0x49, 0x4f, 0x45, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x65,
	0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x09, 0x65, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x42, 0x52, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x42, 0x0b, 0x44,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x04, 0x2e, 0x3b,
	0x70, 0x62, 0xa2, 0x02, 0x03, 0x50, 0x58, 0x58, 0xaa, 0x02, 0x05, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0xca, 0x02, 0x05, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0xe2, 0x02, 0x11, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x05, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_device_proto_rawDescOnce sync.Once
	file_device_proto_rawDescData = file_device_proto_rawDesc
)

func file_device_proto_rawDescGZIP() []byte {
	file_device_proto_rawDescOnce.Do(func() {
		file_device_proto_rawDescData = protoimpl.X.CompressGZIP(file_device_proto_rawDescData)
	})
	return file_device_proto_rawDescData
}

var file_device_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_device_proto_goTypes = []interface{}{
	(*AVLData)(nil),   // 0: proto.AVLData
	(*GPS)(nil),       // 1: proto.GPS
	(*IOElement)(nil), // 2: proto.IOElement
}
var file_device_proto_depIdxs = []int32{
	1, // 0: proto.AVLData.gps:type_name -> proto.GPS
	2, // 1: proto.AVLData.io_elements:type_name -> proto.IOElement
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_device_proto_init() }
func file_device_proto_init() {
	if File_device_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_device_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AVLData); i {
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
		file_device_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GPS); i {
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
		file_device_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IOElement); i {
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
			RawDescriptor: file_device_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_device_proto_goTypes,
		DependencyIndexes: file_device_proto_depIdxs,
		MessageInfos:      file_device_proto_msgTypes,
	}.Build()
	File_device_proto = out.File
	file_device_proto_rawDesc = nil
	file_device_proto_goTypes = nil
	file_device_proto_depIdxs = nil
}
