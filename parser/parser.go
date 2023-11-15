package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	pb "github.com/irisco88/protos/gen/device/v1"
	"go.uber.org/zap"
	"strconv"
	"time"
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
	//logger.Info("salaaaaaaaaaaaaaaaaam111",
	//	zap.Any("crc:", crc),
	//	zap.Any("header:", header),
	//	zap.Any("calculatedCRC:", calculatedCRC),
	//)
	return points, nil
}
func convertToDate(epochTimestamp int64) string {
	// Check if the epoch timestamp is in seconds, convert to milliseconds if needed
	if epochTimestamp <= 9999999999 {
		epochTimestamp *= 1000
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	logger.Info("time**************************************************1",
		zap.Any("time2:", epochTimestamp),
	)
	// Convert epoch timestamp to time.Time
	timeValue := time.Unix(0, epochTimestamp*int64(time.Millisecond))
	logger.Info("time**************************************************2",
		zap.Any("time2:", timeValue),
	)
	// Format time in a desired layout
	dateString := timeValue.Format("2006-01-02 15:04:05 MST")
	logger.Info("time**************************************************3",
		zap.Any("time2:", dateString),
	)
	return dateString
}
func parseCodec8EPacket(reader *bytes.Buffer, header *Header, imei string) ([]*pb.AVLData, error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	points := make([]*pb.AVLData, header.NumberOfData)
	for i := uint8(0); i < header.NumberOfData; i++ {
		timestamps := binary.BigEndian.Uint64(reader.Next(8))
		timestamp := convertToDate(int64(timestamps))
		logger.Info("time**************************************************",
			zap.Any("time:", convertToDate(int64(timestamps))),
			zap.Any("time2:", timestamp),
		)
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
	//slices.SortFunc(points, func(a, b *pb.AVLData) bool {
	//	return a.Timestamp < b.Timestamp
	//})
	return points, nil
}
func parseCodec8eIOElements(reader *bytes.Buffer) (elements []*pb.IOElement, err error) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	//total id (N of Total ID)
	totalElements := binary.BigEndian.Uint16(reader.Next(2))
	fmt.Println(totalElements)
	//logger.Info("salaaaaaaaaaaaaaaaaam4",
	//	zap.Any("totalElements:", totalElements),
	//)
	//n1 , n2 , n4 , n8
	for stage := 1; stage <= 4; stage++ {
		//total id in this stage  (N 1|2|4|8 of One Byte Io )
		stageElements := binary.BigEndian.Uint16(reader.Next(2))
		for elementIndex := uint16(0); elementIndex < stageElements; elementIndex++ {
			elementID := binary.BigEndian.Uint16(reader.Next(2))
			switch stage {
			case 1: // One byte IO Elements
				elementValue := parseNOneValue(reader, elementID)
				elements = append(elements, elementValue)
				//logger.Info("uuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuu1",
				//	zap.Any("elements:", elements),
				//)
			case 2: // Two byte IO Elements
				elementValue := parseNTowValue(reader, elementID)
				elements = append(elements, elementValue)
				//logger.Info("uuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuu2",
				//	zap.Any("elements:", elements),
				//)
			case 3: // Four byte IO Elements
				elementValue := parseNFourValue(reader, elementID)
				elements = append(elements, elementValue)
			case 4: // Eight byte IO Elements
				logger.Info("uuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuu3",
					zap.Any("elements:", "***"),
				)
				elementValue := parseNEightValue(reader, elementID)
				//elements = elementValue
				elements = append(elements, elementValue...)
			}
		}
	}
	reader.Next(2) //nx
	//logger.Info("salaaaaaaaaaaaaaaaaam_final",
	//	zap.Any("elements", elements),
	//)
	return elements, nil
}

func parseNOneValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string
	var elementIntValue float64
	elementIntValue = float64(int64(reader.Next(1)[0]))
	var value pb.IOElement
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
		elementName = strconv.Itoa(int(elementId))
	}
	//logger.Info("salaaaaaaaaaaaaaaaaam60_n1",
	//	zap.Any("elementIntValue_n1:", elementIntValue),
	//	zap.Any("elementId_n1:", elementId),
	//	zap.Any("elementName_n1:", elementName),
	//)
	value.ElementName = elementName
	value.ElementValue = elementIntValue
	return &value
}
func parseNTowValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string
	var value pb.IOElement
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
		elementName = strconv.Itoa(int(elementId))
	}
	//logger.Info("salaaaaaaaaaaaaaaaaam60_n2",
	//	zap.Any("elementIntValue_n2:", elementIntValue),
	//	zap.Any("elementId_n2:", elementId),
	//	zap.Any("elementName_n2:", elementName),
	//)
	value.ElementName = elementName
	value.ElementValue = elementIntValue
	return &value
}
func parseNFourValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var elementName string
	var elementIntValue int64
	var elementIntValues float64
	var value pb.IOElement
	elementIntValue = int64(binary.BigEndian.Uint32(reader.Next(4)))
	elementIntValues = float64(elementIntValue)
	elementName = strconv.Itoa(int(elementId))

	value.ElementName = elementName
	value.ElementValue = elementIntValues
	return &value
}
func parseNEightValue(reader *bytes.Buffer, elementId uint16) (value []*pb.IOElement) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	var eightbytes = reader.Next(8)
	var byte7 = eightbytes[0]
	var byte6 = eightbytes[1]
	var byte5 = eightbytes[2]
	var byte4 = eightbytes[3]
	var byte3 = eightbytes[4]
	var byte2 = eightbytes[5]
	var byte1 = eightbytes[6]
	var byte0 = eightbytes[7]

	var values []*pb.IOElement
	switch elementId {
	case 145:
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			//elementIntValue=elementIntValue*0.05625
			elementItem.ElementName = "Vehicle Speed"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("145-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint16([]byte{0, 0, 0, 0, eightbytes[3], eightbytes[2], 0, 0}))
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "EngineSpeed_RPM"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("145-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := float64(eightbytes[4])
			//elementIntValue=(elementIntValue* 0.75) - 48
			elementItem.ElementName = "Engine Coolant Temperature"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("145-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Fuel level in tank"
			//elementItem.ElementValue = elementIntValue * 0.390625
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("145-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "CheckEngine "
			elementItem0.ElementValue = float64(getBit(eightbytes[6], 0))
			values = append(values, &elementItem0)
			logger.Info("145-5_0",
				zap.Any("values:", elementItem0.ElementValue),
			)
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "AirConditionPressureSwitch1 "
			elementItem1.ElementValue = float64(getBit(eightbytes[6], 1))
			values = append(values, &elementItem1)
			logger.Info("145-5_1",
				zap.Any("values:", elementItem1.ElementValue),
			)
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "AirConditionPressureSwitch2 "
			elementItem2.ElementValue = float64(getBit(eightbytes[6], 2))
			values = append(values, &elementItem2)
			logger.Info("145-5_2",
				zap.Any("values:", elementItem2.ElementValue),
			)
			//34
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearShiftindicator "
			elementItem3.ElementValue = float64(int((eightbytes[6] & 0x18) >> 3))
			values = append(values, &elementItem3)
			logger.Info("145-5_3",
				zap.Any("values:", elementItem3.ElementValue),
			)
			//567
			var elementItem4 pb.IOElement
			elementItem4.ElementName = "DesiredGearValue "
			elementItem4.ElementValue = float64(int((eightbytes[6] & 0xe0) >> 5))
			values = append(values, &elementItem4)
			logger.Info("145-5_4",
				zap.Any("values:", elementItem4.ElementValue),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Vehicle Type"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("145-6",
				zap.Any("values:", values),
			)
		}
	case 146:
		if eightbytes[0] == byte7 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "Condition immobilizer"
			//012
			elementItem0.ElementValue = float64(int(eightbytes[0] & 0x07))
			values = append(values, &elementItem0)
			logger.Info("146-1",
				zap.Any("values:", values),
			)

			//34
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "BrakePedalStatus"
			elementItem1.ElementValue = float64(int((eightbytes[0] & 0x18) >> 3))
			values = append(values, &elementItem1)
			logger.Info("146-2",
				zap.Any("values:", values),
			)

			//5
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "ClutchPedalStatus "
			elementItem2.ElementValue = float64(getBit(eightbytes[0], 5))
			values = append(values, &elementItem2)
			logger.Info("146-3",
				zap.Any("values:", values),
			)

			//67
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearEngagedStatus "
			elementItem3.ElementValue = float64(int((eightbytes[0] & 0xC0) >> 5))
			values = append(values, &elementItem3)
			logger.Info("146-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "ActualAccPedal"
			//elementItem.ElementValue = elementIntValue*0.39063
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("146-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "EngineThrottlePosition"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("146-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "IndicatedEngineTorque"
			//elementItem.ElementValue = elementIntValue*0.39063
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("146-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Engine Friction Torque"
			//elementItem.ElementValue = elementIntValue *0.39063
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("146-8",
				zap.Any("values:", values),
			)

		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "EngineActualTorque"
			//elementItem.ElementValue = elementIntValue * 0.39063
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("146-9",
				zap.Any("values:", values),
			)

		}
		if eightbytes[6] == byte1 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "CruiseControlOn_Off "
			elementItem0.ElementValue = float64(getBit(eightbytes[6], 0))
			values = append(values, &elementItem0)
			logger.Info("146-10",
				zap.Any("values:", values),
			)

			var elementItem1 pb.IOElement
			elementItem1.ElementName = "SpeedLimiterOn_Off"
			elementItem1.ElementValue = float64(getBit(eightbytes[6], 1))
			values = append(values, &elementItem1)
			logger.Info("146-11",
				zap.Any("values:", values),
			)
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "condition cruise control lamp "
			elementItem2.ElementValue = float64(getBit(eightbytes[6], 2))
			values = append(values, &elementItem2)
			logger.Info("146-12",
				zap.Any("values:", values),
			)

			var elementItem3 pb.IOElement
			elementItem3.ElementName = "EngineFuleCutOff"
			elementItem3.ElementValue = float64(getBit(eightbytes[6], 3))
			values = append(values, &elementItem3)
			logger.Info("146-13",
				zap.Any("values:", values),
			)
			var elementItem4 pb.IOElement
			elementItem4.ElementName = "Condition catalyst heating activated"
			elementItem4.ElementValue = float64(getBit(eightbytes[6], 4))
			values = append(values, &elementItem4)
			logger.Info("146-14",
				zap.Any("values:", values),
			)
			var elementItem5 pb.IOElement
			elementItem5.ElementName = "AC compressor status"
			elementItem5.ElementValue = float64(getBit(eightbytes[6], 5))
			values = append(values, &elementItem5)
			logger.Info("146-15",
				zap.Any("values:", values),
			)
			var elementItem6 pb.IOElement
			elementItem6.ElementName = "Condition main relay(Starter Relay)"
			elementItem6.ElementValue = float64(getBit(eightbytes[6], 6))
			values = append(values, &elementItem6)
			logger.Info("146-16",
				zap.Any("values:", values),
			)
			var elementItem7 pb.IOElement
			elementItem7.ElementName = "Reserve"
			elementItem7.ElementValue = float64(getBit(eightbytes[6], 7))
			values = append(values, &elementItem7)
			logger.Info("146-17",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "Reserve "
			elementItem0.ElementValue = float64(getBit(eightbytes[7], 0))
			values = append(values, &elementItem0)
			logger.Info("146-18",
				zap.Any("values:", values),
			)

			var elementItem1 pb.IOElement
			elementItem1.ElementName = "Reserve "
			elementItem1.ElementValue = float64(getBit(eightbytes[7], 1))
			values = append(values, &elementItem1)
			logger.Info("146-19",
				zap.Any("values:", values),
			)
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "Reserve "
			elementItem2.ElementValue = float64(getBit(eightbytes[7], 2))
			values = append(values, &elementItem2)
			logger.Info("146-20",
				zap.Any("values:", values),
			)
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "Reserve "
			elementItem3.ElementValue = float64(getBit(eightbytes[7], 3))
			values = append(values, &elementItem3)
			logger.Info("146-21",
				zap.Any("values:", values),
			)
			var elementItem4 pb.IOElement
			elementItem4.ElementName = "Reserve "
			elementItem4.ElementValue = float64(getBit(eightbytes[7], 4))
			values = append(values, &elementItem4)
			logger.Info("146-22",
				zap.Any("values:", values),
			)
			var elementItem5 pb.IOElement
			elementItem5.ElementName = "Reserve "
			elementItem5.ElementValue = float64(getBit(eightbytes[7], 5))
			values = append(values, &elementItem5)
			logger.Info("146-23",
				zap.Any("values:", values),
			)
			var elementItem6 pb.IOElement
			elementItem6.ElementName = "Reserve "
			elementItem6.ElementValue = float64(getBit(eightbytes[7], 6))
			values = append(values, &elementItem6)
			logger.Info("146-24",
				zap.Any("values:", values),
			)
			var elementItem7 pb.IOElement
			elementItem7.ElementName = "Reserve "
			elementItem7.ElementValue = float64(getBit(eightbytes[7], 7))
			values = append(values, &elementItem7)
			logger.Info("146-25",
				zap.Any("values:", values),
			)
		}
	case 147:
		logger.Info("147-0",
			zap.Any("values:", values),
		)
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], eightbytes[4], eightbytes[3], eightbytes[2], eightbytes[1], eightbytes[0]}))
			elementIntValue := float64(binary.BigEndian.Uint32([]byte{byte2, byte3, byte4, byte5, byte6, byte7}))
			elementItem.ElementName = "distance"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("147-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "ActualAccPedal"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("147-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Intake air temperature"
			//elementItem.ElementValue = (elementIntValue * 0.75) - 48
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("147-3",
				zap.Any("values:", values),
			)
		}
	case 148:
		logger.Info("148-0",
			zap.Any("values:", values),
		)
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], eightbytes[0]}))
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte6, byte7}))
			elementItem.ElementName = "DesiredSpeed"
			//elementItem.ElementValue = elementIntValue*0.125
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("147-1",
				zap.Any("values:", values),
			)

		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Oil temperature(TCU)"
			//elementItem.ElementValue = elementIntValue-40
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-2",
				zap.Any("values:", values),
			)

		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Ambient air temperature"
			//elementItem.ElementValue = (elementIntValue* 0.5) - 40
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Number of DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "EMS_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "ABS_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "BCM_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("148-7",
				zap.Any("values:", values),
			)
		}
	case 149:
		logger.Info("149-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "ACU_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "ESC_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "ICN_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "EPS_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "CAS_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "FCM/FN_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "ICU_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Reserve_DTC"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("149-8",
				zap.Any("values:", values),
			)
		}
	case 150:
		logger.Info("150-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "Sensor1_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "Sensor1_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Sensor2_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Sensor2_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Sensor3_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Sensor3_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "Sensor4_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Sensor4_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("150-8",
				zap.Any("values:", values),
			)
		}
	case 151:
		logger.Info("151-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "Sensor5_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "Sensor5_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Sensor6_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Sensor6_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Sensor7_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Sensor7_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "Sensor8_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Sensor8_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("151-8",
				zap.Any("values:", values),
			)
		}
	case 152:
		logger.Info("152-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "Sensor9_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "Sensor9_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Sensor10_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Sensor10_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Sensor11_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Sensor11_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "Sensor12_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Sensor12_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("152-7",
				zap.Any("values:", values),
			)
		}

	case 153:
		logger.Info("153-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "Sensor13_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "Sensor13_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Sensor14_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Sensor14_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Sensor15_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Sensor15_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "Sensor16_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Sensor16_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("153-8",
				zap.Any("values:", values),
			)
		}

	case 154:
		logger.Info("154-0",
			zap.Any("values:", values),
		)
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, 0, eightbytes[0]}))
			elementIntValue := float64(eightbytes[0])
			elementItem.ElementName = "Sensor17_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("154-1",
				zap.Any("values:", values),
			)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := float64(eightbytes[1])
			elementItem.ElementName = "Sensor17_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("154-2",
				zap.Any("values:", values),
			)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, eightbytes[2], 0, 0}))
			elementIntValue := float64(eightbytes[2])
			elementItem.ElementName = "Sensor18_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("154-3",
				zap.Any("values:", values),
			)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, eightbytes[3], 0, 0, 0}))
			elementIntValue := float64(eightbytes[3])
			elementItem.ElementName = "Sensor18_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("154-4",
				zap.Any("values:", values),
			)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[4])
			elementItem.ElementName = "Sensor19_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)
			logger.Info("154-5",
				zap.Any("values:", values),
			)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, eightbytes[5], 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[5])
			elementItem.ElementName = "Sensor19_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)

			logger.Info("154-6",
				zap.Any("values:", values),
			)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, eightbytes[6], 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[6])
			elementItem.ElementName = "Sensor20_high"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)

			logger.Info("154-7",
				zap.Any("values:", values),
			)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(eightbytes[7])
			elementItem.ElementName = "Sensor20_low"
			elementItem.ElementValue = elementIntValue
			values = append(values, &elementItem)

			logger.Info("154-8",
				zap.Any("values:", values),
			)
		}

	default:
		var elementItem pb.IOElement
		elementItem.ElementName = strconv.Itoa(int(elementId))
		elementItem.ElementValue = 999
		values = append(values, &elementItem)

	}
	return values
}

//	func ConvertByteToBitArray(b byte) byte {
//		//var byteString = string(b)
//		//num, _ := strconv.Atoi(byteString)
//		var bitArray byte
//		for i := 0; i < 8; i++ {
//			bit := (b >> i) & 1
//			bitArray = append(bitArray, bit)
//		}
//		return bitArray
//	}
func ConvertByteToBitArray(byte byte) []int {
	bits := make([]int, 8)
	for i := 0; i < 8; i++ {
		bit := (byte >> i) & 1
		bits[i] = int(bit)
	}

	return Reverser(bits)
}

func Reverser(b []int) []int {
	bitsRver := make([]int, 8)
	bitsRver[0] = b[7]
	bitsRver[1] = b[6]
	bitsRver[2] = b[5]
	bitsRver[3] = b[4]
	bitsRver[4] = b[3]
	bitsRver[5] = b[2]
	bitsRver[6] = b[1]
	bitsRver[7] = b[0]
	return bitsRver
}

func getBit(byteValue byte, bitPosition uint) int {
	// Shift the bit to the rightmost position
	shiftedBit := byteValue >> bitPosition
	// Use bitwise AND with 1 to extract the rightmost bit
	result := int(shiftedBit & 1)
	return result
}
