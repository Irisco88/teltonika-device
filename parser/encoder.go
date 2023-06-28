package parser

import (
	"encoding/binary"
)

type AVLData struct {
	Timestamp  uint64
	Priority   PacketPriority
	Longitude  float64
	Latitude   float64
	Altitude   int16
	Angle      uint16
	Satellites uint8
	Speed      uint16
	EventID    uint16
	IOElements []*IOElement
}

type IOElement struct {
	ID    uint16
	Value any
}

type PacketPriority uint8

const (
	PriorityLow   PacketPriority = 0
	PriorityHigh  PacketPriority = 1
	PriorityPanic PacketPriority = 2
)

func MakeCodec8Packet(points []*AVLData) ([]byte, error) {
	data := make([]byte, 0)
	data = append(data, 0, 0, 0, 0)
	avlDataBytes, err := EncodeCodec8ExtendedAVLData(points)
	if err != nil {
		return nil, err
	}
	data = binary.BigEndian.AppendUint32(data, uint32(len(avlDataBytes))+1)
	data = append(data, 0x8e)
	data = append(data, uint8(len(points)))
	data = append(data, avlDataBytes...)
	data = append(data, uint8(len(points)))
	data = binary.BigEndian.AppendUint32(data, 0) //crc16
	return data, nil
}

func EncodeCodec8ExtendedAVLData(points []*AVLData) ([]byte, error) {
	var data []byte
	for _, point := range points {
		// Timestamp (8 bytes)
		data = binary.BigEndian.AppendUint64(data, point.Timestamp)

		// Priority (1 byte)
		data = append(data, uint8(point.Priority))

		// Longitude (8 bytes)
		data = binary.BigEndian.AppendUint32(data, uint32(point.Longitude*1e7))

		// Latitude (4 bytes)
		data = binary.BigEndian.AppendUint32(data, uint32(point.Latitude*1e7))

		// Altitude (2 bytes)
		data = binary.BigEndian.AppendUint16(data, uint16(point.Altitude))

		// Angle (2 bytes)
		data = binary.BigEndian.AppendUint16(data, point.Angle)

		// Satellites (1 byte)
		data = append(data, point.Satellites)

		// Speed (2 bytes)
		data = binary.BigEndian.AppendUint16(data, point.Speed)

		// event ID
		data = binary.BigEndian.AppendUint16(data, point.EventID)

		// IO Elements
		data = binary.BigEndian.AppendUint16(data, uint16(len(point.IOElements)))
		stageOne, stageTwo, stageThree, stageFour := make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0)
		stageCounts := struct {
			stage1, stage2, stage3, stage4 uint16
		}{}
		for _, element := range point.IOElements {
			bytes, err := numberToStream(element.Value)
			if err != nil {
				return nil, err
			}
			switch len(bytes) {
			case 1:
				stageCounts.stage1++
				stageOne = binary.BigEndian.AppendUint16(stageOne, element.ID)
				stageOne = append(stageOne, bytes...)
			case 2:
				stageCounts.stage2++
				stageTwo = binary.BigEndian.AppendUint16(stageTwo, element.ID)
				stageTwo = append(stageTwo, bytes...)
			case 4:
				stageCounts.stage3++
				stageThree = binary.BigEndian.AppendUint16(stageThree, element.ID)
				stageThree = append(stageThree, bytes...)
			case 8:
				stageCounts.stage4++
				stageFour = binary.BigEndian.AppendUint16(stageFour, element.ID)
				stageFour = append(stageFour, bytes...)
			}
		}
		data = binary.BigEndian.AppendUint16(data, stageCounts.stage1)
		data = append(data, stageOne...)
		data = binary.BigEndian.AppendUint16(data, stageCounts.stage2)
		data = append(data, stageTwo...)
		data = binary.BigEndian.AppendUint16(data, stageCounts.stage3)
		data = append(data, stageThree...)
		data = binary.BigEndian.AppendUint16(data, stageCounts.stage4)
		data = append(data, stageFour...)
		data = binary.BigEndian.AppendUint16(data, uint16(0)) //nx
	}
	return data, nil
}
