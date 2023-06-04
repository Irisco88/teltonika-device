package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/packetify/teltonika-device/proto/pb"
)

var (
	ErrInvalidElementLen   = errors.New("invalid elements length")
	ErrInvalidPreamble     = errors.New("invalid Preamble")
	ErrInvalidNumberOfData = errors.New("invalid number of data")
	ErrInvalidHeader       = errors.New("parse header failed")
	ErrCheckCRC            = errors.New("CRC check failed")
	ErrUnsupportedCodec    = errors.New("codec not supported")
)

type Header struct {
	DataLength   uint32
	CodecID      uint8
	NumberOfData uint8
}

func ParseHeader(reader *bytes.Buffer) (*Header, error) {
	header := &Header{}
	preamble := binary.BigEndian.Uint32(reader.Next(4))
	if preamble != uint32(0) {
		return nil, ErrInvalidPreamble
	}
	header.DataLength = binary.BigEndian.Uint32(reader.Next(4))
	header.CodecID = reader.Next(1)[0]
	header.NumberOfData = reader.Next(1)[0]
	return header, nil
}

func ParsePacket(data []byte, imei string) ([]*pb.AVLData, error) {
	reader := bytes.NewBuffer(data)
	header, err := ParseHeader(reader)
	if err != nil {
		return nil, ErrInvalidHeader
	}
	var points []*pb.AVLData
	if header.CodecID == 0x8e {
		points, err = parseCodec8EPacket(reader, header, imei)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrUnsupportedCodec
	}
	// Once finished with the records we read the Record Number and the CRC
	if reader.Next(1)[0] != header.NumberOfData {
		return nil, ErrInvalidNumberOfData
	}
	crc := binary.BigEndian.Uint32(reader.Next(4))
	calculatedCRC := calculateCRC16(data)
	if uint32(calculatedCRC) != crc {
		//TODO check crc
		//return nil, ErrCheckCRC
	}
	return points, nil
}

func parseCodec8EPacket(reader *bytes.Buffer, header *Header, imei string) ([]*pb.AVLData, error) {
	points := make([]*pb.AVLData, header.NumberOfData)
	for i := uint8(0); i < header.NumberOfData; i++ {
		timestamp := binary.BigEndian.Uint64(reader.Next(8))
		priority := reader.Next(1)[0]
		// GPS Element
		longitude := int32(binary.BigEndian.Uint32(reader.Next(4)))
		if longitude>>31 == 1 {
			longitude *= -1
		}
		latitude := int32(binary.BigEndian.Uint32(reader.Next(4)))
		if latitude>>31 == 1 {
			latitude *= -1
		}
		altitude := int32(binary.BigEndian.Uint16(reader.Next(2)))
		angle := int32(binary.BigEndian.Uint16(reader.Next(2)))
		Satellites := int32(reader.Next(1)[0])
		speed := int32(binary.BigEndian.Uint16(reader.Next(2)))
		eventID := binary.BigEndian.Uint16(reader.Next(2))
		points[i] = &pb.AVLData{
			Imei:      imei,
			Timestamp: timestamp,
			Priority:  pb.PacketPriority(priority),
			EventId:   uint32(eventID),
			Gps: &pb.GPS{
				Longitude:  float64(longitude),
				Latitude:   float64(latitude),
				Altitude:   altitude,
				Angle:      angle,
				Speed:      speed,
				Satellites: Satellites,
			},
		}
		elements, err := parseCodec8eIOElements(reader)
		if err != nil {
			return nil, fmt.Errorf("parse io elements failed:%v", err)
		}
		points[i].IoElements = elements

	}
	return points, nil
}

func parseCodec8eIOElements(reader *bytes.Buffer) (elements []*pb.IOElement, err error) {
	totalElements := binary.BigEndian.Uint16(reader.Next(2))
	for stage := 1; stage <= 4; stage++ {
		stageElements := binary.BigEndian.Uint16(reader.Next(2))

		for elementIndex := uint16(0); elementIndex < stageElements; elementIndex++ {
			var (
				elementValue int64
				elementID    uint16
			)
			elementID = binary.BigEndian.Uint16(reader.Next(2))
			switch stage {
			case 1: // One byte IO Elements
				elementValue = int64(reader.Next(1)[0])
			case 2: // Two byte IO Elements
				elementValue = int64(binary.BigEndian.Uint16(reader.Next(2)))
			case 3: // Four byte IO Elements
				elementValue = int64(binary.BigEndian.Uint32(reader.Next(4)))
			case 4: // Eight byte IO Elements
				elementValue = int64(binary.BigEndian.Uint64(reader.Next(8)))
			}
			elements = append(elements, &pb.IOElement{
				ElementId: int32(elementID),
				Value:     elementValue,
			})
		}
	}
	reader.Next(2) //nx
	if len(elements) != int(totalElements) {
		return nil, ErrInvalidElementLen
	}
	return elements, nil
}
