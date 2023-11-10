package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	pb "github.com/irisco88/protos/gen/device/v1"
	"go.uber.org/zap"
	"reflect"

	"golang.org/x/exp/slices"
	//"net"
	"strconv"
	//"sync"
)

var (
	ErrInvalidElementLen   = errors.New("invalid elements length")
	ErrInvalidPreamble     = errors.New("invalid Preamble")
	ErrInvalidNumberOfData = errors.New("invalid number of data")
	ErrInvalidHeader       = errors.New("parse header failed")
	ErrCheckCRC            = errors.New("CRC check failed")
	ErrUnsupportedCodec    = errors.New("codec not supported")
)

const PRECISION = 10000000.0

type Header struct {
	DataLength   uint32
	CodecID      uint8
	NumberOfData uint8
	logger       *zap.Logger
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
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
	logger.Info("salaaaaaaaaaaaaaaaaam111",
		zap.Any("crc:", crc),
		zap.Any("header:", header),
		zap.Any("calculatedCRC:", calculatedCRC),
	)
	return points, nil
}

func parseCodec8EPacket(reader *bytes.Buffer, header *Header, imei string) ([]*pb.AVLData, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
		logger.Info("salaaaaaaaaaaaaaaaaam112",
			zap.Any("Longitude:", longitude),
			zap.Any("Latitude:", latitude),
			zap.Any("Altitude:", altitude),
			zap.Any("Angle:", angle),
			zap.Any("Speed:", speed),
			zap.Any("Satellites:", Satellites),
			zap.Any("eventID:", eventID),
			zap.Any("priority:", priority),
			zap.Any("timestamp:", timestamp),
		)
		points[i] = &pb.AVLData{
			Imei:      imei,
			Timestamp: timestamp,
			Priority:  pb.PacketPriority(priority),
			EventId:   uint32(eventID),
			Gps: &pb.GPS{
				Longitude:  float64(longitude) / PRECISION,
				Latitude:   float64(latitude) / PRECISION,
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
	slices.SortFunc(points, func(a, b *pb.AVLData) bool {
		return a.Timestamp < b.Timestamp
	})
	return points, nil
}

func parseCodec8eIOElements(reader *bytes.Buffer) (elements []*pb.IOElement, err error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	//total id (N of Total ID)
	totalElements := binary.BigEndian.Uint16(reader.Next(2))
	logger.Info("salaaaaaaaaaaaaaaaaam4",
		zap.Any("totalElements:", totalElements),
	)
	//n1 , n2 , n4 , n8
	for stage := 1; stage <= 4; stage++ {

		//total id in this stage  (N 1|2|4|8 of One Byte Io )
		stageElements := binary.BigEndian.Uint16(reader.Next(2))
		logger.Info("salaaaaaaaaaaaaaaaaam5",
			zap.Any("stageElements:", stageElements),
		)
		for elementIndex := uint16(0); elementIndex < stageElements; elementIndex++ {
			//var (
			//	elementValue *pb.Value
			//	elementID    uint16
			//)
			//var elementValueArray = []*pb.Value{}

			//logger.Info("salaaaaaaaaaaaaaaaaam5000",
			//	zap.Any("len:", binary.BigEndian.Uint16(reader.Next(2))),
			//)

			elementID := binary.BigEndian.Uint16(reader.Next(2))
			switch stage {
			case 1: // One byte IO Elements

				elementValue := parseNOneValue(reader, elementID)
				elements = append(elements, elementValue)
			//break
			case 2: // Two byte IO Elements
				elementValue := parseNTowValue(reader, elementID)
				elements = append(elements, elementValue)
			case 3: // Four byte IO Elements
				elementValue := parseNFourValue(reader, elementID)
				elements = append(elements, elementValue)
			case 4: // Eight byte IO Elements
				elementValue := parseNEightValue(reader, elementID)
				elements = elementValue
			}
		}
	}
	reader.Next(2) //nx
	logger.Info("salaaaaaaaaaaaaaaaaam9",
		zap.Any("nx:", reader.Next(2)),
	)
	if len(elements) != int(totalElements) {
		logger.Info("salaaaaaaaaaaaaaaaaam10",
			zap.Any("total:", totalElements),
			zap.Any("len:", len(elements)),
		)
		return nil, ErrInvalidElementLen
	}
	//slices.SortFunc(elements, func(a, b *pb.IOElement) bool {
	//	return a.ElementName < b.ElementName
	//})
	return elements, nil
}

func parseNOneValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string

	var elementIntValue float64
	elementIntValue = float64(int64(reader.Next(1)[0]))

	switch elementId {
	case 1:
		elementName = "Digital Input 1"
	case 2:
		elementName = "Digital Input 2"
	case 21:
		elementName = "GSM Signal"
	case 144:
		elementName = "SD Status"
	case 179:
		elementName = "Digital Output 1"
	case 180:
		elementName = "Digital Output 2"
	case 239:
		elementName = "Ignition"
	case 247:
		elementName = "Crash Detection"
	case 255:
		elementName = "Over Speeding"
	default:
		elementName = "default Value"
	}
	logger.Info("salaaaaaaaaaaaaaaaaam60",
		zap.Any("elementIntValue:", elementIntValue),
		zap.Any("elementId:", elementId),
		zap.Any("elementName:", elementName),
	)
	if values != nil {
		values.ElementName = elementName
		values.ElementValue = elementIntValue
	}
	return values
}
func parseNTowValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string

	var elementIntValue float64
	elementIntValue = float64(int64(binary.BigEndian.Uint16(reader.Next(2))))
	switch elementId {
	case 9:
		elementName = "Analog Input 1"
	case 10:
		elementName = "Analog Input 2"
	case 11:
		elementName = "Analog Input 3"
	case 66:
		elementName = "External Voltage"
	case 67:
		elementName = "Battery Voltage"
	case 70:
		elementName = "PCB Temperature"
	case 245:
		elementName = "Analog Input 4"
	default:
		elementName = "default"
	}

	logger.Info("salaaaaaaaaaaaaaaaaam60",
		zap.Any("elementIntValue:", elementIntValue),
		zap.Any("elementId:", elementId),
		zap.Any("elementName:", elementName),
	)
	if values != nil {
		values.ElementName = elementName
		values.ElementValue = elementIntValue
	}
	return values
}
func parseNFourValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string

	var elementIntValue float64
	elementIntValue = float64(int64(binary.BigEndian.Uint16(reader.Next(4))))
	if values != nil {
		values.ElementName = strconv.Itoa(int(elementId))
		values.ElementValue = elementIntValue
	}
	logger.Info("salaaaaaaaaaaaaaaaaam60",
		zap.Any("elementIntValue:", elementIntValue),
		zap.Any("elementId:", elementId),
		zap.Any("elementName:", elementName),
	)
	return values
}
func parseNEightValue(reader *bytes.Buffer, elementId uint16) (values []*pb.IOElement) {
	var elementItem pb.IOElement
	var eightbytes = reader.Next(8)
	var byte1 = eightbytes[0]
	var byte2 = eightbytes[1]
	//var byte3= eightbytes[2]
	var byte4 = eightbytes[3]
	var byte5 = eightbytes[4]
	var byte6 = eightbytes[5]
	var byte7 = eightbytes[6]
	var byte8 = eightbytes[7]
	//elementIntValue := float64(binary.BigEndian.Uint64(eightbytes))
	bitArray1 := ConvertByteToBitArray(eightbytes[0])
	bitArray2 := ConvertByteToBitArray(eightbytes[1])
	bitArray3 := ConvertByteToBitArray(eightbytes[2])
	bitArray4 := ConvertByteToBitArray(eightbytes[3])
	bitArray5 := ConvertByteToBitArray(eightbytes[4])
	bitArray6 := ConvertByteToBitArray(eightbytes[5])
	bitArray7 := ConvertByteToBitArray(eightbytes[6])
	bitArray8 := ConvertByteToBitArray(eightbytes[7])
	switch elementId {
	case 145:

		if eightbytes[1] == byte2 {
			var bytesArr []byte
			bytesArr[0] = eightbytes[0]
			bytesArr[1] = eightbytes[1]
			elementIntValue := float64(binary.BigEndian.Uint64(bytesArr))
			elementItem.ElementName = "Vehicle Speed"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var bytesArr []byte
			bytesArr[0] = eightbytes[0]
			bytesArr[1] = eightbytes[1]
			bytesArr[2] = eightbytes[2]
			bytesArr[3] = eightbytes[3]
			elementIntValue := float64(binary.BigEndian.Uint64(bytesArr))
			elementItem.ElementName = "EngineSpeed_RPM"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
		}
		if eightbytes[4] == byte5 {
			var bytesArr []byte
			bytesArr[0] = eightbytes[0]
			bytesArr[1] = eightbytes[1]
			bytesArr[2] = eightbytes[2]
			bytesArr[3] = eightbytes[3]
			bytesArr[4] = eightbytes[4]
			elementIntValue := float64(binary.BigEndian.Uint64(bytesArr))
			elementItem.ElementName = "Engine Coolant Temperature"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte6 {
			var bytesArr []byte
			bytesArr[0] = eightbytes[0]
			bytesArr[1] = eightbytes[1]
			bytesArr[2] = eightbytes[2]
			bytesArr[3] = eightbytes[3]
			bytesArr[4] = eightbytes[4]
			bytesArr[5] = eightbytes[5]
			elementIntValue := float64(binary.BigEndian.Uint64(bytesArr))
			elementItem.ElementName = "Fuel level in tank"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
		}

		if eightbytes[6] == byte7 {
			var bits = ConvertByteToBitArray(eightbytes[6])
			if isSameTwoArray(bits, bitArray7) {
				if bits[0] == bitArray7[0] {
					elementItem.ElementName = "CheckEngine "
					elementItem.ElementValue = 999
					values = append(values, &elementItem)
				}
				if bits[1] == bitArray7[1] {
					elementItem.ElementName = "AirConditionPressureSwitch1 "
					elementItem.ElementValue = 999
					values = append(values, &elementItem)
				}
				if bits[2] == bitArray7[2] {
					elementItem.ElementName = "AirConditionPressureSwitch2 "
					elementItem.ElementValue = 999
					values = append(values, &elementItem)
				}
				if bits[4] == bitArray7[4] {
					elementItem.ElementName = "GearShiftindicator "
					elementItem.ElementValue = 999
					values = append(values, &elementItem)
				}
				if bits[7] == bitArray7[7] {
					elementItem.ElementName = "DesiredGearValue "
					elementItem.ElementValue = 999
					values = append(values, &elementItem)
				}
			}

		}

	case 146:
		//	var b = reader.Next(1)[0]
		//	var bitArray = ConvertByteToBitArray(b)
		//
		//	elementItem.ElementName = "Condition immobilizer"
		//	elementItem.ElementValue = float64(bitArray[0,
		//	1, 2])
		//values = append(values, &elementItem)
		//
		//elementItem.ElementName = "BrakePedalStatus"
		//elementItem.ElementValue = float64(bitArray[3, 4])
		//values = append(values, &elementItem)
		//
		//elementItem.ElementName = "ClutchPedalStatus"
		//elementItem.ElementValue = float64(bitArray[5])
		//values = append(values, &elementItem)
		//
		//elementItem.ElementName = "GearEngagedStatus"
		//elementItem.ElementValue = float64(bitArray[6, 7])
		//values = append(values, &elementItem)
		elementItem.ElementName = "Condition immobilizer !"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ActualAccPedal"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "EngineThrottlePosition"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "IndicatedEngineTorque"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Engine Friction Torque"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "EngineActualTorque"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		//var b = reader.Next(1)[0]
		//var bitArray = ConvertByteToBitArray(b)
		//
		//elementItem.ElementName = "CruiseControlOn_Off"
		//elementItem.ElementValue = float64(bitArray[0])
		//
		//elementItem.ElementName = "SpeedLimiterOn_Off"
		//elementItem.ElementValue = float64(bitArray[1])
		//
		//elementItem.ElementName = "condition cruise control lamp"
		//elementItem.ElementValue = float64(bitArray[2])
		//
		//elementItem.ElementName = "EngineFuleCutOff"
		//elementItem.ElementValue = float64(bitArray[3])
		//
		//elementItem.ElementName = "Condition catalyst heating activated"
		//elementItem.ElementValue = float64(bitArray[4])
		//
		//elementItem.ElementName = "AC compressor status"
		//elementItem.ElementValue = float64(bitArray[5])
		//
		//elementItem.ElementName = "Condition main relay -----> Starter Relay"
		//elementItem.ElementValue = float64(bitArray[6])
		//
		//elementItem.ElementName = "Reserve"
		//elementItem.ElementValue = float64(bitArray[7])
		elementItem.ElementName = "CruiseControlOn_Off !"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Reserve"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 147:
		elementItem.ElementName = "distance"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(5)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ActualAccPedal"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Intake air temperature"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 148:
		elementItem.ElementName = "DesiredSpeed"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(2)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Oil temperature------>TCU"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Ambient air temperature"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Number of DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "EMS_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ABS_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "BCM_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 149:
		elementItem.ElementName = "ACU_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ESC_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ICN_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(2)))
		values = append(values, &elementItem)

		elementItem.ElementName = "EPS_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "CAS_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "FCM/FN_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "ICU_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Reserve_DTC"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 150:
		elementItem.ElementName = "Sensor1_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor1_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor2_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor2_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor3_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor3_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor4_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor4_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 151:
		elementItem.ElementName = "Sensor5_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor5_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor6_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor6_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor7_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor7_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor8_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor8_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 152:
		elementItem.ElementName = "Sensor9_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor9_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor10_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor10_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor11_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor11_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor12_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor12_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	case 153:
		elementItem.ElementName = "Sensor13_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor13_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor14_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor14_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor15_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor15_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor16_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor16_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)
	case 154:
		elementItem.ElementName = "Sensor17_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor17_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor18_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor18_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor19_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor19_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor20_low"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

		elementItem.ElementName = "Sensor20_high"
		elementItem.ElementValue = float64(binary.BigEndian.Uint32(reader.Next(1)))
		values = append(values, &elementItem)

	}
	return values
}

func ConvertByteToBitArray(b byte) []int {
	var byteString = string(b)
	num, _ := strconv.Atoi(byteString)
	b1 := num & 1
	b2 := num & 2
	b3 := num & 4
	b4 := num & 8
	b5 := num & 16
	b6 := num & 32
	b7 := num & 64
	b8 := num & 128
	return []int{b8, b7, b6, b5, b4, b3, b2, b1}
}
func isSameTwoArray(a []int, b []int) bool {
	return reflect.DeepEqual(a, b)
}
