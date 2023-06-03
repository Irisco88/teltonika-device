package server

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
	points := make([]*pb.AVLData, header.NumberOfData)
	for i := uint8(0); i < header.NumberOfData; i++ {
		timestamp, err := streamToNumber[int64](reader.Next(8))
		if err != nil {
			return nil, err
		}
		priority, err := streamToNumber[uint8](reader.Next(1))
		if err != nil {
			return nil, err
		}

		// GPS Element
		longitudeInt, err := streamToInt32(reader.Next(4))
		if err != nil {
			return nil, err
		}
		//longitude := float64(longitudeInt) / PRECISION
		latitudeInt, err := streamToInt32(reader.Next(4))
		if err != nil {
			return nil, err
		}
		//latitude := float64(latitudeInt) / PRECISION

		altitude, err := streamToNumber[int16](reader.Next(2))
		if err != nil {
			return nil, err
		}
		angle, err := streamToNumber[int16](reader.Next(2))
		if err != nil {
			return nil, err
		}
		Satellites, err := streamToNumber[uint8](reader.Next(1))
		if err != nil {
			return nil, err
		}
		speed, err := streamToNumber[int16](reader.Next(2))
		if err != nil {
			return nil, err
		}

		points[i] = &pb.AVLData{
			Imei:      imei,
			Timestamp: timestamp,
			Priority:  pb.PacketPriority(priority),
			Gps: &pb.GPS{
				Longitude:  longitudeInt,
				Latitude:   latitudeInt,
				Altitude:   int32(altitude),
				Angle:      int32(angle),
				Speed:      int32(speed),
				Satellites: int32(Satellites),
			},
		}
		eventID, elements, err := ParseIOElements(reader)
		if err != nil {
			return nil, fmt.Errorf("parse io elements failed:%v", err)
		}
		points[i].IoElements = elements
		points[i].EventId = uint32(eventID)

	}
	// Once finished with the records we read the Record Number and the CRC
	numberOfData2, err := streamToNumber[uint8](reader.Next(1)) // Number of Records
	if err != nil {
		return nil, err
	}
	if numberOfData2 != header.NumberOfData {
		return nil, ErrInvalidNumberOfData
	}
	_, err = streamToNumber[uint32](reader.Next(4)) // CRC

	return points, nil
}

func ParseIOElements(reader *bytes.Buffer) (eventID uint16, elements []*pb.IOElement, err error) {
	eventID, err = streamToNumber[uint16](reader.Next(2))
	if err != nil {
		return 0, nil, err
	}
	totalElements, err := streamToNumber[uint16](reader.Next(2))
	if err != nil {
		return 0, nil, err
	}
	for stage := 1; stage <= 4; stage++ {
		stageElements, err := streamToNumber[uint16](reader.Next(2))
		if err != nil {
			break
		}
		for elementIndex := uint16(0); elementIndex < stageElements; elementIndex++ {
			var (
				elementValue int64
				elementID    uint16
			)
			elementID, err = streamToNumber[uint16](reader.Next(2))
			if err != nil {
				return 0, nil, err
			}

			switch stage {
			case 1: // One byte IO Elements
				tmp, e := streamToNumber[int8](reader.Next(1))
				if e != nil {
					return 0, nil, e
				}
				elementValue = int64(tmp)
			case 2: // Two byte IO Elements
				tmp, e := streamToNumber[int16](reader.Next(2))
				if e != nil {
					return 0, nil, e
				}
				elementValue = int64(tmp)
			case 3: // Four byte IO Elements
				tmp, e := streamToNumber[int32](reader.Next(4))
				if e != nil {
					return 0, nil, e
				}
				elementValue = int64(tmp)
			case 4: // Eight byte IO Elements
				elementValue, err = streamToNumber[int64](reader.Next(8))
				if err != nil {
					return 0, nil, err
				}
			}
			elements = append(elements, &pb.IOElement{
				ElementId: int32(elementID),
				Value:     elementValue,
			})
		}
	}
	reader.Next(2) //nx
	if len(elements) != int(totalElements) {
		return 0, nil, ErrInvalidElementLen
	}
	return eventID, elements, nil
}
