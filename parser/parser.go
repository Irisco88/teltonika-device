package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	pb "github.com/irisco88/protos/gen/device/v1"
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
func convertToDate(epochTimestamp int64) string {
	if epochTimestamp <= 9999999999 {
		epochTimestamp *= 1000
	}
	// Convert epoch timestamp to time.Time
	timeValue := time.Unix(0, epochTimestamp*int64(time.Millisecond))
	tehranLocation, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		fmt.Println("Error loading Tehran location:", err)
	}
	// Convert to Tehran local time
	tehranLocalTime := timeValue.In(tehranLocation)
	// Format time in a desired layout
	dateString := tehranLocalTime.Format("2006-01-02 15:04:05")
	return dateString
}
func parseCodec8EPacket(reader *bytes.Buffer, header *Header, imei string) ([]*pb.AVLData, error) {
	points := make([]*pb.AVLData, header.NumberOfData)
	for i := uint8(0); i < header.NumberOfData; i++ {
		timestamps := binary.BigEndian.Uint64(reader.Next(8))
		timestamp := convertToDate(int64(timestamps))
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
	return points, nil
}
func parseCodec8eIOElements(reader *bytes.Buffer) (elements []*pb.IOElement, err error) {
	//total id (N of Total ID)
	totalElements := binary.BigEndian.Uint16(reader.Next(2))
	fmt.Println(totalElements)
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
			case 2: // Two byte IO Elements
				elementValue := parseNTowValue(reader, elementID)
				elements = append(elements, elementValue)
			case 3: // Four byte IO Elements
				elementValue := parseNFourValue(reader, elementID)
				elements = append(elements, elementValue)
			case 4: // Eight byte IO Elements
				elementValue := parseNEightValue(reader, elementID)
				//elements = elementValue
				elements = append(elements, elementValue...)
			}
		}
	}
	reader.Next(2) //nx
	return elements, nil
}

func parseNOneValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	var elementName string
	var elementIntValue float64
	elementIntValue = float64(int64(reader.Next(1)[0]))
	var value pb.IOElement
	switch elementId {
	case 1:
		elementName = "DigitalInput1"
	case 2:
		elementName = "DigitalInput2"
	case 21:
		elementName = "GSMSignal"
	case 144:
		elementName = "SDStatus"
	case 179:
		elementName = "DigitalOutput1"
	case 180:
		elementName = "DigitalOutput2"
	case 239:
		elementName = "Ignition"
	case 247:
		elementName = "CrashDetection"
	case 255:
		elementName = "OverSpeeding"
	default:
		elementName = strconv.Itoa(int(elementId))
	}
	value.ElementName = elementName
	value.ElementValue = elementIntValue
	return &value
}
func parseNTowValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
	var elementName string
	var value pb.IOElement
	var elementIntValue float64
	elementIntValue = float64(int64(binary.BigEndian.Uint16(reader.Next(2))))
	switch elementId {
	case 9:
		elementName = "AnalogInput1"
	case 10:
		elementName = "AnalogInput2"
	case 11:
		elementName = "AnalogInput3"
	case 66:
		elementName = "ExternalVoltage"
	case 67:
		elementName = "BatteryVoltage"
	case 70:
		elementName = "PCBTemperature"
	case 245:
		elementName = "AnalogInput4"
	default:
		elementName = strconv.Itoa(int(elementId))
	}
	value.ElementName = elementName
	value.ElementValue = elementIntValue
	return &value
}
func parseNFourValue(reader *bytes.Buffer, elementId uint16) (values *pb.IOElement) {
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
			elementIntValue := (float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))) * 0.05625
			elementItem.ElementName = "VehicleSpeed"
			if elementIntValue > 200 {
				elementItem.ElementValue = 200
			} else {
				elementItem.ElementValue = elementIntValue
			}
			elementItem.NormalValue = elementIntValue / 8189
			elementItem.ColorValue = "#a09db2"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "EngineSpeed_RPM"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 8160
			elementItem.ColorValue = "#008080"
			values = append(values, &elementItem)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := ((float64(byte4)) * 0.75) - 48
			elementItem.ElementName = "EngineCoolantTemperature"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 48) / (143.5 + 48)
			elementItem.ColorValue = "#065535"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte5)) * 0.390625
			elementItem.ElementName = "FuelLevelinTank"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[6] == byte1 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "CheckEngine"
			elementItem0.ElementValue = float64(getBit(byte6, 0))
			elementItem0.NormalValue = float64(getBit(byte6, 0))
			elementItem0.ColorValue = "#ff80ed"
			values = append(values, &elementItem0)
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "AirConditionPressureSwitch1"
			elementItem1.ElementValue = float64(getBit(byte6, 1))
			elementItem1.NormalValue = float64(getBit(byte6, 1))
			elementItem1.ColorValue = "#198ba3"
			values = append(values, &elementItem1)
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "AirConditionPressureSwitch2"
			elementItem2.ElementValue = float64(getBit(byte6, 2))
			elementItem2.NormalValue = float64(getBit(byte6, 2))
			elementItem2.ColorValue = "#ae0e52"
			values = append(values, &elementItem2)
			//34
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearShiftindicator"
			elementItem3.ElementValue = float64(int((byte6 & 0x18) >> 3))
			elementItem3.NormalValue = 1000
			elementItem3.ColorValue = ""
			values = append(values, &elementItem3)
			//567
			var elementItem4 pb.IOElement
			elementItem4.ElementName = "DesiredGearValue"
			elementItem4.ElementValue = float64(int((byte6 & 0xe0) >> 5))
			elementItem4.NormalValue = 1000
			elementItem4.ColorValue = ""
			values = append(values, &elementItem4)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(byte7)
			elementItem.ElementName = "VehicleType"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
	case 146:
		if eightbytes[0] == byte7 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "ConditionImmobilizer"
			//012
			elementItem0.ElementValue = float64(int(byte0 & 0x07))
			elementItem0.NormalValue = 1000
			elementItem0.ColorValue = ""
			values = append(values, &elementItem0)
			//34
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "BrakePedalStatus"
			elementItem1.ElementValue = float64(int((byte0 & 0x18) >> 3))
			elementItem1.NormalValue = ((float64(int((byte0 & 0x18) >> 3))) - 1) / 2
			elementItem1.ColorValue = "#7bcf7d"
			values = append(values, &elementItem1)
			//5
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "ClutchPedalStatus"
			elementItem2.ElementValue = float64(getBit(byte0, 5))
			elementItem2.NormalValue = float64(getBit(byte0, 5))
			elementItem2.ColorValue = "#282a36"
			values = append(values, &elementItem2)
			//67
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearEngagedStatus"
			elementItem3.ElementValue = float64(int((byte0 & 0xC0) >> 5))
			elementItem3.NormalValue = float64(int((byte0 & 0xC0) >> 5))
			elementItem3.ColorValue = "#c70d0f"
			values = append(values, &elementItem3)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, 0, 0, 0, eightbytes[1], 0}))
			elementIntValue := (float64(byte1)) * 0.39063
			elementItem.ElementName = "ActualAccPedal"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.6094
			elementItem.ColorValue = "#006ab5"
			values = append(values, &elementItem)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte2)) * 0.39063
			elementItem.ElementName = "EngineThrottlePosition"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.2
			elementItem.ColorValue = "#DFFF00"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte3)) * 0.39063
			elementItem.ElementName = "IndicatedEngineTorque"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.6094
			elementItem.ColorValue = "#FFBF00"
			values = append(values, &elementItem)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte4)) * 0.39063
			elementItem.ElementName = "EngineFrictionTorque"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.6094
			elementItem.ColorValue = "#FF7F50"
			values = append(values, &elementItem)

		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte5)) * 0.39063
			elementItem.ElementName = "EngineActualTorque"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.6094
			elementItem.ColorValue = "#DE3163"
			values = append(values, &elementItem)
		}
		if eightbytes[6] == byte1 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "CruiseControlOn_Off"
			elementItem0.ElementValue = float64(getBit(byte6, 0))
			elementItem0.NormalValue = 1000
			elementItem0.ColorValue = ""
			values = append(values, &elementItem0)
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "SpeedLimiterOn_Off"
			elementItem1.ElementValue = float64(getBit(byte6, 1))
			elementItem1.NormalValue = 1000
			elementItem1.ColorValue = ""
			values = append(values, &elementItem1)

			var elementItem2 pb.IOElement
			elementItem2.ElementName = "conditionCruisControlLamp"
			elementItem2.ElementValue = float64(getBit(byte6, 2))
			elementItem2.NormalValue = 1000
			elementItem2.ColorValue = ""
			values = append(values, &elementItem2)

			var elementItem3 pb.IOElement
			elementItem3.ElementName = "EngineFuleCutOff"
			elementItem3.ElementValue = float64(getBit(byte6, 3))
			elementItem3.NormalValue = float64(getBit(byte6, 3))
			elementItem3.ColorValue = "#0000FF"
			values = append(values, &elementItem3)

			var elementItem4 pb.IOElement
			elementItem4.ElementName = "ConditionCatalystHeatingActivated"
			elementItem4.ElementValue = float64(getBit(byte6, 4))
			elementItem4.NormalValue = float64(getBit(byte6, 4))
			elementItem4.ColorValue = "#00FF00"
			values = append(values, &elementItem4)

			var elementItem5 pb.IOElement
			elementItem5.ElementName = "ACCompressorStatus"
			elementItem5.ElementValue = float64(getBit(byte6, 5))
			elementItem5.NormalValue = float64(getBit(byte6, 5))
			elementItem5.ColorValue = "#FF0000"
			values = append(values, &elementItem5)

			var elementItem6 pb.IOElement
			elementItem6.ElementName = "ConditionMainRelay"
			elementItem6.ElementValue = float64(getBit(byte6, 6))
			elementItem6.NormalValue = float64(getBit(byte6, 6))
			elementItem6.ColorValue = "#800080"
			values = append(values, &elementItem6)

			var elementItem7 pb.IOElement
			elementItem7.ElementName = "Reserve"
			elementItem7.ElementValue = float64(getBit(byte6, 7))
			elementItem7.NormalValue = 1000
			elementItem7.ColorValue = ""
			values = append(values, &elementItem7)

		}
		if eightbytes[7] == byte0 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "TCU_GearShiftPosition"
			//012
			elementItem0.ElementValue = float64(int(byte0 & 0x0F))
			elementItem0.NormalValue = 1000
			elementItem0.ColorValue = ""
			values = append(values, &elementItem0)

			var elementItem4 pb.IOElement
			elementItem4.ElementName = "Reserve"
			elementItem4.ElementValue = float64(getBit(byte7, 4))
			elementItem4.NormalValue = 1000
			elementItem4.ColorValue = ""
			values = append(values, &elementItem4)

			var elementItem5 pb.IOElement
			elementItem5.ElementName = "Reserve"
			elementItem5.ElementValue = float64(getBit(byte7, 5))
			elementItem5.NormalValue = 1000
			elementItem5.ColorValue = ""
			values = append(values, &elementItem5)

			var elementItem6 pb.IOElement
			elementItem6.ElementName = "Reserve"
			elementItem6.ElementValue = float64(getBit(byte7, 6))
			elementItem6.NormalValue = 1000
			elementItem6.ColorValue = ""
			values = append(values, &elementItem6)

			var elementItem7 pb.IOElement
			elementItem7.ElementName = "Reserve"
			elementItem7.ElementValue = float64(getBit(byte7, 7))
			elementItem7.NormalValue = 1000
			elementItem7.ColorValue = ""
			values = append(values, &elementItem7)

		}
	case 147:

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint32([]byte{byte3, byte2, byte1, byte0}))
			elementItem.ElementName = "distance"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte4)
			elementItem.ElementName = "Reserve"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte5)
			elementItem.ElementName = "Reserve"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte6)) * 0.39063
			elementItem.ElementName = "VirtualAccPedal"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 99.2
			elementItem.ColorValue = "#FF00FF"
			values = append(values, &elementItem)

		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := ((float64(byte7)) * 0.75) - 48
			elementItem.ElementName = "IntakeAirTemperature"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 48) / (143.5 + 48)
			elementItem.ColorValue = "#000080"
			values = append(values, &elementItem)

		}
	case 148:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := (float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))) * 0.125
			elementItem.ElementName = "DesiredSpeed"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte2)) - 40
			elementItem.ElementName = "OilTemperature(TCU)"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 40) / (214 + 40)
			elementItem.ColorValue = "#0000FF"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := ((float64(byte3)) * 0.5) - 40
			elementItem.ElementName = "AmbientAirTemperature"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 40) / (86.5 + 40)
			elementItem.ColorValue = "#008080"
			values = append(values, &elementItem)

		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := (float64(binary.BigEndian.Uint16([]byte{byte5, byte4})))
			elementItem.ElementName = "EMS_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			// elementItem.ColorValue =""
			val, desc := findEMSMap(elementIntValue)
			elementItem.ColorValue = val + "_" + desc
			values = append(values, &elementItem)
		}

		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte6)
			elementItem.ElementName = "ABS_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte7)
			elementItem.ElementName = "BCM_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			// elementItem.ColorValue = ""
			val, desc := findBCMMap(elementIntValue)
			elementItem.ColorValue = val + "_" + desc
			values = append(values, &elementItem)
		}
	case 149:
		if eightbytes[0] == byte7 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte0)
			elementItem.ElementName = "ACU_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte1)
			elementItem.ElementName = "ESC_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[2] == byte5 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte2)
			elementItem.ElementName = "ICN_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte3)
			elementItem.ElementName = "EPS_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte4)
			elementItem.ElementName = "CAS_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte5)
			elementItem.ElementName = "FCM/FN_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[6] == byte1 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte6)
			elementItem.ElementName = "ICU_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte7)
			elementItem.ElementName = "Reserve_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
	case 150:
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor1"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#008000"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor2"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#808000"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor3"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#800000"
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor4"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#398112"
			values = append(values, &elementItem)
		}
	case 151:
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor5"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#12815E"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor6"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#125781"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementIntValues := elementIntValue / 10
			elementItem.ElementName = "Sensor7"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 1370
			elementItem.ColorValue = "#7E1281"
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor8"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 65535
			elementItem.ColorValue = "#811241"
			values = append(values, &elementItem)
		}
	case 152:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementItem.ElementName = "Sensor9"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 65535
			elementItem.ColorValue = "#817C12"
			values = append(values, &elementItem)
		}

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor10"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 65535
			elementItem.ColorValue = "#F4E60E"
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor11"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 65535
			elementItem.ColorValue = "#0E99F4"
			values = append(values, &elementItem)
		}

		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor12"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 10
			elementItem.ColorValue = "#F40EED"
			values = append(values, &elementItem)
		}

	case 153:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor13"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 10
			elementItem.ColorValue = "#FF6C00"
			values = append(values, &elementItem)

		}

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor14"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 10
			elementItem.ColorValue = "#00FF55"
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor15"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 10
			elementItem.ColorValue = "#9B00FF"
			values = append(values, &elementItem)
		}

		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementIntValues := (elementIntValue / 100) - 10
			elementItem.ElementName = "Sensor16"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = (elementIntValues + 10) / 20
			elementItem.ColorValue = "#FF008F"
			values = append(values, &elementItem)
		}

	case 154:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementIntValues := (elementIntValue / 100) - 10
			elementItem.ElementName = "Sensor17"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = (elementIntValues + 10) / 20
			elementItem.ColorValue = "#51022E"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor18"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 20
			elementItem.ColorValue = "#02513A"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementIntValues := elementIntValue / 100
			elementItem.ElementName = "Sensor19"
			elementItem.ElementValue = elementIntValues
			elementItem.NormalValue = elementIntValues / 20
			elementItem.ColorValue = "#512B02"
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			//var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			//elementItem.ElementName = "Sensor20"
			//elementItem.ElementValue = elementIntValue
			//elementItem.NormalValue = 1
			//elementItem.ColorValue = "#A41B9E"
			//values = append(values, &elementItem)
		}
	default:
		var elementItem pb.IOElement
		elementItem.ElementName = strconv.Itoa(int(elementId))
		elementItem.ElementValue = 999
		elementItem.NormalValue = 1000
		elementItem.ColorValue = ""
		values = append(values, &elementItem)

	}
	return values
}
func findBCMMap(num float64) (string, string) {
	dataMap := map[float64]struct {
		Value string
		Desc  string
	}{
		1:   {"B1000-1A", "BCM:Fuel Gauge Sensor SCG"},
		2:   {"B1000-1B", "BCM:Fuel Gauge Sensor OL"},
		3:   {"B1001-1A", "BCM:Ambient Temperature Sensor SCG"},
		4:   {"B1001-1B", "BCM:Ambient Temperature Sensor OL"},
		5:   {"B1002-1A", "BCM:Evaporator Sensor SCG"},
		6:   {"B1002-1B", "BCM:Evaporator Sensor OL"},
		7:   {"B1003-1B", "BCM:Front Wiper Zero Position OL"},
		8:   {"B1100-15", "BCM:Reverse Lamp OL or SCVBAT"},
		9:   {"B1100-11", "BCM:Reverse Lamp SCG"},
		10:  {"B1101-15", "BCM:DRL OL or SCVBAT"},
		11:  {"B1101-11", "BCM:DRL SCG"},
		12:  {"B1102-11", "BCM:Roof Lamp SCG"},
		13:  {"B1103-15", "BCM:Rear Fog Lamp OL or SCVBAT"},
		14:  {"B1103-11", "BCM:Rear Fog Lamp SCG"},
		15:  {"B1104-15", "BCM:Trunk Lid Actuator OL or SCVBAT"},
		16:  {"B1104-11", "BCM:Trunk Lid Actuator SCG"},
		17:  {"B1105-15", "BCM:Right Indicator Lamp OL or SCVBAT"},
		18:  {"B1105-11", "BCM:Right Indicator Lamp SCG"},
		19:  {"B1106-15", "BCM:Left Indicator Lamp OL or SCVBAT"},
		20:  {"B1106-11", "BCM:Left Indicator Lamp SCG"},
		21:  {"B1107-15", "BCM:Right Brake Lamp OL or SCVBAT"},
		22:  {"B1107-11", "BCM:Right Brake Lamp SCG"},
		23:  {"B1108-15", "BCM:Left Brake Lamp OL or SCVBAT"},
		24:  {"B1108-11", "BCM:Left Brake Lamp SCG"},
		25:  {"B1109-15", "BCM:Right Side Lamp OL or SCVBAT"},
		26:  {"B1109-11", "BCM:Right Side Lamp SCG"},
		27:  {"B110A-15", "BCM:Left Side Lamp OL or SCVBAT"},
		28:  {"B110A-11", "BCM:Left Side Lamp SCG"},
		29:  {"B110B-15", "BCM:Right Dipped Lamp OL or SCVBAT"},
		30:  {"B110B-11", "BCM:Right Dipped Lamp SCG"},
		31:  {"B110C-15", "BCM:Left Dipped Lamp OL or SCVBAT"},
		32:  {"B110C-11", "BCM:Left Dipped Lamp SCG"},
		33:  {"B110D-15", "BCM:Backlight OL or SCVBAT"},
		34:  {"B110D-11", "BCM:Backlight SCG"},
		35:  {"B1120-01", "BCM:Screen-heater Relay fault"},
		36:  {"B1121-01", "BCM:Mirror Fold Relay fault"},
		37:  {"B1122-01", "BCM:Mirror Unfold Relay fault"},
		38:  {"B1123-01", "BCM:Reserved"},
		39:  {"B1124-01", "BCM:Front Window Winder Power Relay fault"},
		40:  {"B1125-01", "BCM:Rear Window Winder Power Relay fault"},
		41:  {"B1126-01", "BCM:Front Wiper Control Relay fault"},
		42:  {"B1127-01", "BCM:Front Wiper Speed Relay fault"},
		43:  {"B1128-01", "BCM:Reserved"},
		44:  {"B1129-01", "BCM:Compressor Clutch Relay fault"},
		45:  {"B112A-01", "BCM:Front Fog Lamps Relay fault"},
		46:  {"B112B-01", "BCM:Main Lamps Relay fault"},
		47:  {"B112C-01", "BCM:Reserved"},
		48:  {"B112F-01", "BCM:Sunroof Relay fault"},
		49:  {"B1200-51", "BCM:Unit not programmed"},
		50:  {"B1201-55", "BCM:Missing configuration"},
		51:  {"B1202-54", "BCM:Remote not learnt"},
		52:  {"B1202-56", "BCM:Erroneous configuration received from diagnostic tool."},
		53:  {"U1F0A-88", "BCM:CAN/LS NERR fault / BUSOFF fault"},
		54:  {"U1F0F-88", "BCM:CAN/LS NBCM Mute"},
		55:  {"U1F10-87", "BCM:ICN Absent"},
		56:  {"U1F11-87", "BCM:AIRBAG Absent"},
		57:  {"U1F12-87", "BCM:MMS Absent"},
		58:  {"U1F13-87", "BCM:PEPS Absent"},
		59:  {"U1F2A-88", "BCM:CAN/HS BUSOFF fault"},
		60:  {"U1F2F-88", "BCM:CAN/HS NBCM Mute"},
		61:  {"U1F30-87", "BCM:EMS Absent"},
		62:  {"U1F31-87", "BCM:ESC Absent"},
		63:  {"U1F32-87", "BCM:EPS Absent"},
		64:  {"U1F33-87", "BCM:TCU Absent"},
		65:  {"U1F34-87", "BCM:ICU Absent"},
		66:  {"U1F50-87", "BCM:PAS Absent"},
		67:  {"U1F51-87", "BCM:Alternator Absent"},
		68:  {"U1F52-87", "BCM:RLS Absent"},
		69:  {"B10010A", "BCM:Wiper Zero Position/ Incoherency, not plausible"},
		70:  {"B10020A", "BCM:Rear Wiper Zero Position/ Incoherency, not plausible"},
		71:  {"B110001", "BCM:Horn Relay / Open Load"},
		72:  {"B110003", "BCM:Horn Relay / Short Circuit to Vbat"},
		73:  {"B111002", "BCM:LH Indicator Lamps / Short Circuit to Ground"},
		74:  {"B111005", "BCM:LH Indicator Lamps / Open Load or Short Circuit to Battery"},
		75:  {"B111102", "BCM:RH Indicator Lamps / Short Circuit to Ground"},
		76:  {"B111105", "BCM:RH Indicator Lamps / Open Load or Short Circuit to Battery"},
		77:  {"B112001", "BCM:Wiper High Speed /Low Speed Relay  / Open Load"},
		78:  {"B112003", "BCM:Wiper High Speed /Low Speed Relay /  Short Circuit to Vbat"},
		79:  {"B112101", "BCM:Wiper On/Off Relay / Open Load"},
		80:  {"B112103", "BCM:Wiper On/Off Relay / Short Circuit to Vbat"},
		81:  {"B112201", "BCM:Front Wash Pump Relay / Open Load"},
		82:  {"B112203", "BCM:Front Wash Pump Relay / Short Circuit to Vbat"},
		83:  {"B112301", "BCM:Rear Wash Pump /Open Load"},
		84:  {"B112303", "BCM:Rear Wash Pump/Short Circuit to Vbat"},
		85:  {"B113002", "BCM:Side Lamps /Short Circuit to Ground"},
		86:  {"B113005", "BCM:Side Lamps / Open Load or Short Circuit to Battery"},
		87:  {"B114002", "BCM:Stop Lamps / Short Circuit to Ground"},
		88:  {"B114005", "BCM:Stop Lamps/ Open Load or Short Circuit to Battery"},
		89:  {"B115002", "BCM:Reverse Lamps / Short Circuit to Ground"},
		90:  {"B115005", "BCM:Reverse Lamps / Open Load or Short Circuit to Battery"},
		91:  {"B116002", "BCM:Rear Fog Lamps /Short Circuit to Ground"},
		92:  {"B116005", "BCM:Rear Fog Lamps/Open Load or Short Circuit to Battery"},
		93:  {"B117001", "BCM:Front Fog Lamps / Open Load"},
		94:  {"B117003", "BCM:Front Fog Lamps / Short Circuit to Vbat"},
		95:  {"B118001", "BCM:Main Lamps / Open Load"},
		96:  {"B118003", "BCM:Main Lamps / Short Circuit to Vbat"},
		97:  {"B119001", "BCM:Dipped Lamps /Open Load"},
		98:  {"B119003", "BCM:Dipped Lamps /Short Circuit to Vbat"},
		99:  {"B11A001", "BCM:Screen Heater /Open Load"},
		100: {"B11A002", "BCM:Screen Heater /Short Circuit to Ground"},
		101: {"B11B001", "BCM:Roof Lamps /Open Load"},
		102: {"B11B002", "BCM:Roof Lamps /Short Circuit to Ground"},
		103: {"B11C001", "BCM:Veco Output / Open Load"},
		104: {"B11C002", "BCM:Veco Output / Short Circuit to Ground"},
		105: {"B11D001", "BCM:Rear Wiper(P6LHB Only) / Open Load"},
		106: {"B11D002", "BCM:Rear Wiper(P6LHB Only) / Short Circuit to Ground"},
		107: {"B11E001", "BCM:A/C Compressor / Open Load"},
		108: {"B11E003", "BCM:A/C Compressor / Short Circuit to Vbat"},
		109: {"B11F001", "BCM:Day Light / Open Load"},
		110: {"B11F002", "BCM:Day Light / Short Circuit to Ground"},
		111: {"B11F101", "BCM:Hazard LED / Open Load"},
		112: {"B11F102", "BCM:Hazard LED / Short Circuit to Ground"},
		113: {"B11F201", "BCM:Rear Window Power Feed / Open Load"},
		114: {"B11F202", "BCM:Rear Window Power Feed / Short Circuit to Ground"},
		115: {"B11F301", "BCM:Trunk lid open relay / Open Load"},
		116: {"B11F302", "BCM:Trunk lid open relay / Short Circuit to Ground"},
		117: {"U10010D", "BCM:CAN/LS Communication Bus OFF/Network Fault"},
		118: {"U100A0D", "BCM:Reserve./Network Fault"},
		119: {"U100B0D", "BCM:CAN/LS CLU absent/Network Fault"},
		120: {"U100C0D", "BCM:CAN/LS DCN absent/Network Fault"},
		121: {"U100D0D", "BCM:CAN/LS MFD absent/Network Fault"},
		122: {"U100E0D", "BCM:CAN/LS MMS absent/Network Fault"},
		123: {"U10130D", "BCM:CAN/LS CBM mute/Network Fault"},
		124: {"U10140D", "BCM:CAN/LS NERR (CAN physical error)/Network Fault"},
		125: {"U10150D", "BCM:CAN/LS HVAC (CAN physical error)/Network Fault"},
		126: {"U10210D", "BCM:CAN/HS Communication Bus OFF/Network Fault"},
		127: {"U10280D", "BCM:CAN/HS ABS absent/Network Fault"},
		128: {"U102A0D", "BCM:CAN/HS EMS absent/Network Fault"},
		129: {"U102B0D", "BCM:CAN/HS CLU absent/Network Fault"},
		130: {"U102C0D", "BCM:CAN/HS EPS absent/Network Fault"},
		131: {"U102D0D", "BCM:CAN/HS TCU absent/Network Fault"},
		132: {"U102E0D", "BCM:CAN/HS SAS absent/Network Fault"},
		133: {"U102F0D", "BCM:CAN/HS ICU absent/Network Fault"},
		134: {"U10330D", "BCM:CAN/HS CBM mute/Network Fault"},
		135: {"U10360D", "BCM:CAN/HS ACU absent/Network Fault"},
		136: {"U10370D", "BCM:CAN/HS PEPS absent/Network Fault"},
		137: {"U10380D", "BCM:CAN/HS TPMS absent/Network Fault"},
		138: {"B10041A", "BCM:Alternator CHARGING"},
		139: {"B10041B", "BCM:Ambient Temperature Sensor SCG"},
		140: {"B10041C", "BCM:Ambient Temperature Sensor OL"},
		141: {"B10041D", "BCM:Front Wiper Zero Position OL"},
		142: {"B10041E", "BCM:Fuel Gauge Sensor SCG"},
		143: {"B10041F", "BCM:Fuel Gauge Sensor OL"},
		144: {"B110015", "BCM:Reverse Lamp OL or SCVBAT"},
		145: {"B110011", "BCM:Reverse Lamp SCG"},
		146: {"B110115", "BCM:Right Main Lamp OL or SCVBAT"},
		147: {"B110111", "BCM:Right Main Lamp SCG"},
		148: {"B110215", "BCM:Left Main Lamp OL or SCVBAT"},
		149: {"B110211", "BCM:Left Main Lamp SCG"},
		150: {"B110315", "BCM:Rear Fog Lamp OL or SCVBAT"},
		151: {"B110311", "BCM:Rear Fog Lamp SCG"},
		152: {"B110415", "BCM:AC Compressor Clutch OL"},
		153: {"B110411", "BCM:AC Compressor Clutch SC"},
		154: {"B110515", "BCM:Right Indicator Lamp OL or SCVBAT"},
		155: {"B110511", "BCM:Right Indicator Lamp SCG"},
		156: {"B110615", "BCM:Left Indicator Lamp OL or SCVBAT"},
		157: {"B110611", "BCM:Left Indicator Lamp SCG"},
		158: {"B110715", "BCM:Brake Lamp OL or SCVBAT"},
		159: {"B110711", "BCM:Brake Lamp SCG"},
		160: {"B110B15", "BCM:Right Dipped Lamp OL or SCVBAT"},
		161: {"B110B11", "BCM:Right Dipped Lamp SCG"},
		162: {"B110C15", "BCM:Left Dipped Lamp OL or SCVBAT"},
		163: {"B110C11", "BCM:Left Dipped Lamp SCG"},
		164: {"B110D15", "BCM:RH Turning (Corner) Lamp OL or SCVBAT"},
		165: {"B110D11", "BCM:RH Turning (Corner) Lamp SCG"},
		166: {"B110E15", "BCM:LH Turning (Corner) Lamp OL or SCVBAT"},
		167: {"B110E11", "BCM:LH Turning (Corner) Lamp SCG"},
		168: {"B110F15", "BCM:Trunk lamp OL or SCVBAT"},
		169: {"B110F11", "BCM:Trunk lamp SCG"},
		170: {"B110D10", "BCM:Unit not programmed (If applicable)"},
		171: {"B110D12", "BCM:Missing configuration"},
		172: {"B110D13", "BCM:Erroneous configuration received from diagnostic tool*"},
		173: {"U1F0A88", "BCM:CAN/LS NERR fault / BUSOFF fault"},
		174: {"U1F1087", "BCM:ATC Absent"},
		175: {"U1F1187", "BCM:MMS Absent"},
		176: {"U1F1287", "BCM:CAS Absent"},
		177: {"U1F1387", "BCM:ACU Absent"},
		178: {"U1F1487", "BCM:ESCL Absent"},
		179: {"U1F1587", "BCM:HVAC Absent"},
		180: {"U1F1687", "BCM:PLG Absent"},
		181: {"U1F2A88", "BCM:CAN/HS BUSOFF fault"},
		182: {"U1F3087", "BCM:EMS Absent"},
		183: {"U1F3187", "BCM:ABS Absent"},
		184: {"U1F3287", "BCM:EPS Absent"},
		185: {"U1F3387", "BCM:TCU Absent"},
		186: {"U1F3487", "BCM:BSD Absent"},
		187: {"U1F3587", "BCM:EGS Absent"},
		188: {"U1F3687", "BCM:WCM Absent"},
		189: {"U1F5087", "BCM:RPAS Absent"},
		190: {"U1F5187", "BCM:FPAS Absent"},
		191: {"U1F5287", "BCM:RLS Absent"},
		192: {"U1F5387", "BCM:RATL Absent"},
		193: {"U1F5487", "BCM:FATL Absent"},
		194: {"U1F5587", "BCM:REAR DSM"},
	}
	if data, ok := dataMap[num]; ok {
		return data.Value, data.Desc
	}

	return "", ""
}

func findEMSMap(num float64) (string, string) {
	dataMap := map[float64]struct {
		Value string
		Desc  string
	}{
		1:   {"P2123", "EMS:Throttle/Pedal Position Sensor must be added/ Accelerator pedal 1st potentiometer: The first potentiometer voltage is exceeded upper plausible limit. It may be short circuit to power supply."},
		2:   {"P2128", "EMS:Throttle/Pedal Position Sensor must be added/ Monitor the acceleratio pedal position sensor 2# voltage signal, if it is above the limit, it is determined to be faulty"},
		3:   {"P2122", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the acceleration pedal position sensor 1# voltage signal, if it is below the limit, it is determined to be faulty"},
		4:   {"P2127", "EMS:Throttle/Pedal Position Sensor must be added/ Monitor the acceleration pedal position sensor 2# voltage signal, if it is below the limit, it is determined to be faulty"},
		5:   {"P2138", "EMS:Throttle/Pedal Position Sensor must be added/ Monitor the absolute value of the difference between PPS1 and PPS2. If it is greater than the limit, it is determined to be faulty"},
		6:   {"P2135", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the absolute value of the difference between TPS1 and TPS2. If it is greater than the limit, it is determined to be faulty"},
		7:   {"P0123", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the throttle position sensor 1# voltage signal, above the limit,it is determined to be faulty"},
		8:   {"P0122", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the throttle position sensor 1# voltage signal, if it is below the limit, it is determined to be faulty"},
		9:   {"P0223", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the throttle position sensor 2# voltage signal, if it is above the limit, it is determined to be faulty"},
		10:  {"P0222", "EMS: Throttle/Pedal Position Sensor must be added/Monitor the throttle position sensor 2# voltage signal, if it is below the limit, it is determined to be faulty"},
		11:  {"P2104", "EMS:ETC Diagnosis/ Engine forced idle Control"},
		12:  {"P2105", "EMS:ETC Diagnosis/ Forced engine shutdown"},
		13:  {"P2106", "EMS: ETC Diagnosis/Engine performance limit"},
		14:  {"P2110", "EMS: ETC Diagnosis/Throttle out of control"},
		15:  {"P2119", "EMS: ETC Diagnosis/Throttle Actuator \"A\" Control Throttle Body Range/Performance"},
		16:  {"P2101", "EMS: ETC Diagnosis/Monitor the difference between the ETC target position and the actual position. If the limit is exceeded,it is determined to be faulty"},
		17:  {"P0340", "EMS: Crankshaft & Camshaft position sensor(must be isolated)/Monitor the change of the camshaft signal after the diagnosis is enabled"},
		18:  {"P0341", "EMS: Crankshaft & Camshaft position sensor(must be isolated)/Monitor the number of teeth of the camshaft signal after the diagnosis is enabled"},
		19:  {"P0335", "EMS: Crankshaft & Camshaft position sensor(must be isolated)/Monitor the change in engine speed after diagnostics are enabled"},
		20:  {"P0315", "EMS: Crankshaft & Camshaft position sensor(must be isolated)/Crankshaft Position System Variation Not Learned"},
		21:  {"P0336", "EMS: Crankshaft & Camshaft position sensor(must be isolated)/After the diagnosis is enabled, if the crankshaft position error tooth number is greater than the limit value, it is determined to be faulty"},
		22:  {"P0262", "EMS:Gasoline Injector/ Hardware circuit check"},
		23:  {"P0261", "EMS: Gasoline Injector/Hardware circuit check"},
		24:  {"P0268", "EMS: Gasoline Injector/Hardware circuit check"},
		25:  {"P0267", "EMS: Gasoline Injector/Hardware circuit check"},
		26:  {"P0271", "EMS: Gasoline Injector/Hardware circuit check"},
		27:  {"P0270", "EMS: Gasoline Injector/Hardware circuit check"},
		28:  {"P0265", "EMS: Gasoline Injector/Hardware circuit check"},
		29:  {"P0264", "EMS: Gasoline Injector/Hardware circuit check"},
		30:  {"P2300", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		31:  {"P2303", "EMS:Ignition Coil Primary Control Circuit Malfunction / Hardware circuit check"},
		32:  {"P2306", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		33:  {"P2309", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		34:  {"P2301", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		35:  {"P2304", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		36:  {"P2307", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		37:  {"P2310", "EMS: Ignition Coil Primary Control Circuit Malfunction /Hardware circuit check"},
		38:  {"P0300", "EMS:misfire/ When a cylinder has a fire, its crankshaft rotation speed is slowed down."},
		39:  {"P0301", "EMS: misfire/ When a cylinder has a misfire, its crankshaft rotation speed is slowed. If the calibration limit is exceeded and 90% of the detected misfire comes from the first cylinder, it is determined that the first cylinder is misfired"},
		40:  {"P0302", "EMS: misfire/ When a cylinder has a misfire, its crankshaft rotation speed is slowed. If the calibration limit is exceeded and 90% of the detected misfire comes from the second cylinder, it is determined that the second cylinder is misfired"},
		41:  {"P0303", "EMS: misfire/ When a cylinder has a misfire, its crankshaft rotation speed is slowed. If the calibration limit is exceeded and 90% of the detected misfire comes from the third cylinder, it is determined that the third cylinder is misfired"},
		42:  {"P0304", "EMS: misfire/ When a cylinder has a misfire, its crankshaft rotation speed is slowed. If the calibration limit is exceeded and 90% of the detected misfire comes from the fourth cylinder, it is determined that the fourth cylinder is misfired"},
		43:  {"P0602", "EMS: Control Module Programming Error"},
		44:  {"P0604", "EMS: Internal Control Module Random Access Memory (RAM) Error"},
		45:  {"P0606", "EMS: Control Module Processor"},
		46:  {"P0651", "EMS: Monitor the sensor reference voltage percentage signal, and judge the fault if it is not within the limit"},
		47:  {"P0641", "EMS: Monitor the sensor reference voltage percentage signal, and judge the fault if it is not within the limit"},
		48:  {"P0107", "EMS: Manifold Absolute pressure(MAP)/ Manifold pressure sensor: pressure voltage is below lower plausible threshold .it may be due to short circuit to ground."},
		49:  {"P0108", "EMS: Manifold Absolute pressure(MAP)/ Manifold pressure sensor: sensor voltage is exceeded upper plausible threshold .it may be due to signal interruption or short circuit to power supply."},
		50:  {"P023D", "EMS: Manifold Absolute pressure(MAP)/ After the diagnosis is enabled, the difference between the pressure value of the intake pressure sensor and the pressure value of the boost pressure sensor is monitored. If the difference is greater than the maximum limit or below the minimum limit, the fault is determined"},
		51:  {"P0112", "EMS: Intake Air Temperature Sensor /Intake manifold air temperature sensor temperature sensor: The sensor voltage is below lower plausible value. It may be due to short circuit to ground."},
		52:  {"P0113", "EMS: Intake Air Temperature Sensor /Intake manifold air temperature sensor: The sensor voltage is exceeded upper plausible value. It may be due to signal interruption, ground cable interruption or short circuit to power supply"},
		53:  {"P0116", "EMS:Coolant Sensor/ If the temperature of the coolant is less than the limit value of the temperature of the power-on coolant, it is determined to be faulty / If the temperature change of the coolant is less than the limit within a certain period of time after the cold start, it is determined to be faulty"},
		54:  {"P0117", "EMS: Coolant Sensor/ Monitor the coolant temperature sensor signal voltage. When the voltage is lower than the limit, it is determined to be faulty"},
		55:  {"P0118", "EMS: Coolant Sensor/ Monitor the coolant temperature sensor signal voltage. When the voltage is higher than the limit, it is determined to be faulty"},
		56:  {"P0125", "EMS: Coolant Sensor/Monitor the time of Coolant Temperature for Closed Loop Fuel Control , when the time is above the limit, it is determined to be faulty"},
		57:  {"P0128", "EMS: Coolant Sensor/Monitor the time when the engine coolant temperature reaches the fault diagnosis limit of the thermostat,When the time is above the limit, it is determined to be faulty"},
		58:  {"P050C", "EMS: Coolant Sensor/ Monitor Cold Start Engine Coolant Temperature, when the value is above the limit, it is determined to be faulty"},
		59:  {"P0037", "EMS: Downstream O2 Sensor must be added / Hardware circuit check"},
		60:  {"P0038", "EMS: Downstream O2 Sensor must be added / Hardware circuit check"},
		61:  {"P00D2", "EMS: Downstream O2 Sensor must be added / If the downstream oxygen sensor heating current is below the limit, it is judged to be faulty"},
		62:  {"P0139", "EMS: Downstream O2 Sensor must be added / Monitor the response time of the downstream oxygen sensor in the deceleration and fuel cut condition. When the filtered response time exceeds the limit, it is determined to be faulty"},
		63:  {"P0136", "EMS: Downstream O2 Sensor must be added / Monitor the downstream oxygen sensor signal voltage.When the voltage signal is between the calibration limits,it is judged to be faulty"},
		64:  {"P0137", "EMS: Downstream O2 Sensor must be added / Monitor the downstream oxygen sensor signal voltage, and when the voltage signal is below the limit, it is determined to be faulty"},
		65:  {"P0138", "EMS: Downstream O2 Sensor must be added / Monitor the downstream oxygen sensor signal voltage, and when the voltage signal is above the limit, it is determined to be faulty"},
		66:  {"P2270", "EMS: Downstream O2 Sensor must be added / The downstream oxygen sensor voltage value is less than the limit when the power is enriched"},
		67:  {"P2271", "EMS: Downstream O2 Sensor must be added /  The downstream oxygen sensor voltage value is greater than the limit when the oil is decelerated"},
		68:  {"P2096", "EMS: Downstream O2 Sensor must be added / Based on the fuel closed-loop control, the downstream oxygen sensor correction value, if greater than the limit, it is determined to be faulty"},
		69:  {"P2097", "EMS: Downstream O2 Sensor must be added /  Based on the fuel closed-loop control, the downstream oxygen sensor correction value, if less than the limit, it is determined to be faulty"},
		70:  {"P0031", "EMS: O2 Sensor Up /Hardware circuit check"},
		71:  {"P0032", "EMS: O2 Sensor Up /Hardware circuit check"},
		72:  {"P00D1", "EMS: O2 Sensor Up /O2 Sensor Up /If the heating current of the upstream oxygen sensor is lower than the limit value,it is determined to be faulty"},
		73:  {"P0130", "EMS: O2 Sensor Up /Monitoring the signal voltage of the upstream oxygen sensor, when the voltage signal is between the calibration limit,it is determined to be faulty"},
		74:  {"P0131", "EMS: O2 Sensor Up /Monitoring the signal voltage of the upstream oxygen sensor, when the voltage signal is lower than the limit, it is determined to be faulty"},
		75:  {"P0132", "EMS: O2 Sensor Up /Monitor the upstream oxygen sensor signal voltage, and when the voltage signal is above the limit, it is determined to be faulty"},
		76:  {"P0133", "EMS: O2 Sensor Up /Monitor the average response time of the upstream oxygen sensor from lean to rich and from rich to lean. When the response time exceeds the limit at the same time, it is determined to be faulty "},
		77:  {"P0134", "EMS: O2 Sensor Up /Monitor the number of times of upstream oxygen sensor conversion, and when the number of conversions is lower than the limit, it is determined to be faulty"},
		78:  {"P2A00", "EMS: O2 Sensor Up /Monitor the front oxygen sensor ready flag. If the time is not ready exceeds the limit, it is determined to be faulty / Monitor the time when the upstream oxygen sensor enters the ready state under cold start. If the time exceeds the limit, it is determined to be faulty"},
		79:  {"P2195", "EMS: O2 Sensor Up /When the voltage of the upstream oxygen sensor is less than the limit when the power is enriched, it is determined to be faulty"},
		80:  {"P2196", "EMS: O2 Sensor Up /When the voltage of the upstream oxygen sensor is greater than the limit when the oil is decelerated, it is determined to be faulty"},
		81:  {"P0627", "EMS: FUEL PUMPRELAY / Hardware circuit check"},
		82:  {"U0121", "EMS:CAN Communication Failure/ Lost Communication With Anti-Lock Brake System (ABS) Control Module"},
		83:  {"U0001", "EMS: CAN Communication Failure/Monitoring communication loss between the ECU and all other nodes on the vehicle CAN bus"},
		84:  {"U0073", "EMS: CAN Communication Failure/After the CAN Bus off is detected, after a certain counting time, the bus shutdown fault is reported."},
		85:  {"U0101", "EMS: CAN Communication Failure/Lost Communication with TCM"},
		86:  {"U0167", "EMS: CAN Communication Failure/Lost Communication With Vehicle Immobilizer Control Module"},
		87:  {"P0564", "EMS: Cruise Control must be omitted / Cruise Control Multi-Function Input \"A\" Circuit"},
		88:  {"P0565", "EMS: Cruise Control must be omitted / Cruise Control \"On\" Signal"},
		89:  {"P0567", "EMS: Cruise Control must be omitted / Cruise Control \"Resume\" Signal"},
		90:  {"P0568", "EMS: Cruise Control must be omitted / Cruise Control “Set” Signal"},
		91:  {"P0504", "EMS: Brake Pedal /Brake Switch \"A\"/\"B\" Correlation"},
		92:  {"P0571", "EMS: Brake Pedal /Brake Switch \"A\" Circuit"},
		93:  {"P0324", "EMS: Knock/Combustion Vibration Control System /Monitor the original signal filter value of the knock sensor"},
		94:  {"P0325", "EMS: Knock/Combustion Vibration Control System /Monitoring software-filtered knock sensor output signal integral value"},
		95:  {"P0562", "EMS: System Voltage / Monitor system voltage, if it is below the limit, it is determined to be faulty"},
		96:  {"P0563", "EMS: System Voltage / Monitor system voltage, if it is above the limit, it is determined to be faulty"},
		97:  {"P0480", "EMS: Cooling Fan / Hardware circuit check"},
		98:  {"P0481", "EMS: Cooling Fan / Hardware circuit check"},
		99:  {"P0482", "EMS: Cooling Fan / Hardware circuit check"},
		100: {"P0420", "EMS: Catalyst /When the diagnosis is enabled, the target air-fuel ratio is forcibly changed, and the upstream and downstream oxygen sensor signals are observed to calculate the oxygen storage time; if the filtered oxygen storage time is lower than the fault limit, the catalyst conversion efficiency is judged to be low"},
		101: {"P0461", "EMS: Fuel Level Sensor/ After the diagnosis is enabled, the difference between the maximum and minimum values of the monitored oil level signal is monitored. If the difference is less than the limit value, it is determined to be faulty"},
		102: {"P0462", "EMS: Fuel Level Sensor /Monitor the original voltage of the tank level sensor, less than the limit, it is determined to be faulty"},
		103: {"P0463", "EMS: Fuel Level Sensor /Monitor the original voltage of the tank level sensor, above the limit, it is determined to be faulty"},
		104: {"P0458", "EMS: System Purge / Hardware circuit check"},
		105: {"P0459", "EMS: System Purge / Hardware circuit check"},
		106: {"P0633", "EMS: Immobilizer / According to immo prototal"},
		107: {"U0426", "EMS: Immobilizer / Invalid Data Received From Vehicle Immobilizer Control Module"},
		108: {"P0685", "EMS: ECM/PCM Power Relay Control Circuit/Open / Hardware circuit check"},
		109: {"P2177", "EMS: Fuel System / Monitor Fuel condition in non - idle condition"},
		110: {"P2178", "EMS: Fuel System / Monitor Fuel condition in non - idle condition"},
		111: {"P2187", "EMS: Fuel System / Fuel condition in idle condition"},
		112: {"P2188", "EMS: Fuel System / Fuel condition in idle condition"},
		113: {"P0011", "EMS: INTKVVT / \"A\" Camshaft Position - Timing Over-Advanced or System Performance Bank 1"},
		114: {"P0077", "EMS: INTKVVT / Intake Valve Control Solenoid Circuit High Bank 1"},
		115: {"P0076", "EMS: INTKVVT / Intake Valve Control Solenoid Circuit Low OR Open Bank 1"},
		116: {"P0076", "EMS: INTKVVT / Intake Valve Control Solenoid Circuit Low OR Open Bank 1"},
		117: {"P000A", "EMS: INTKVVT / \"A\" Camshaft Position Slow Response Bank1"},
		118: {"P0026", "EMS: INTKVVT /Intake Valve Control Solenoid Circuit Range/Performance Bank 1"},
		119: {"P0016", "EMS: INTKVVT / Crankshaft Position – Camshaft Position Correlation (Bank 1 Sensor A)"},
		120: {"P0341", "EMS: INTKVVT / Camshaft Position Sensor “A” Circuit Range/Performance Bank 1"},
		121: {"P0014", "EMS: EXHVVT / \"B\" Camshaft Position - Timing Over-Advanced or System Performance Bank 1"},
		122: {"P0080", "EMS: EXHVVT / Exhaust Valve Control Solenoid Circuit High Bank 1"},
		123: {"P0079", "EMS: EXHVVT / Exhaust Valve Control Solenoid Circuit Low OR open Bank 1"},
		124: {"P0079", "EMS: EXHVVT / Exhaust Valve Control Solenoid Circuit Low OR open Bank 1"},
		125: {"P000B", "EMS: EXHVVT / \"B\" Camshaft Position Slow Response Bank1"},
		126: {"P0027", "EMS: EXHVVT / Exhaust Valve Control Solenoid Circuit Range/Performance Bank 1"},
		127: {"P0017", "EMS: EXHVVT / Crankshaft Position - Camshaft Position Correlation (Bank 1 Sensor B)"},
		128: {"P0366", "EMS: EXHVVT / Camshaft Position Sensor “B” Circuit Range/Performance Bank 1"},
		129: {"P2565", "EMS: Electronic waste gate / The waste gate sensor signal is above the limit"},
		130: {"P2564", "EMS: Electronic waste gate / The waste gate sensor signal is below the limit"},
		131: {"P2566", "EMS: Electronic waste gate / Waste gate sensor signal noise"},
		132: {"P2599", "EMS: Electronic waste gate / Waste gate rationality high"},
		133: {"P2598", "EMS: Electronic waste gate / Waste gate rationality low"},
		134: {"P0034", "EMS: pressure relief valve / Turbocharger/Supercharger Bypass Valve Control Circuit Low SCG"},
		135: {"P0035", "EMS: pressure relief valve / Turbocharger/Supercharger Bypass Valve Control Circuit High SCB"},
		136: {"P0238", "EMS: Boost P/T Sensor / Boost pressure sensor short circuit to high voltage"},
		137: {"P0237", "EMS: Boost P/T Sensor / Boost pressure sensor short circuit to low voltage"},
		138: {"P00CF", "EMS: Boost P/T Sensor / BSTP/AMP rationality"},
		139: {"P0098", "EMS: Boost P/T Sensor / Boost temperature sensor short circuit to high voltage"},
		140: {"P011B", "EMS: Boost P/T Sensor / Boost temperature sensor high rationality"},
		141: {"P011B", "EMS: Boost P/T Sensor / Boost temperature sensor low rationality"},
		142: {"P0097", "EMS: Boost P/T Sensor / Boost temperature sensor short circuit to low voltage"},
		143: {"P0629", "EMS:Brake booster Vaccum PumpRelay/ Brake booster vaccum pump relay: short circuit to power supply"},
		144: {"P0628", "EMS:Brake booster Vaccum PumpRelay/ Brake booster vaccum pump relay: short circuit to ground"},
		145: {"P0557", "EMS:Brake booster pressure sensor / Brake booster pressure sensor: voltage is high .short circuit to power supply"},
		146: {"P0558", "EMS:Brake booster pressure sensor / Brake booster pressure sensor: voltage is low . short circuit to ground or signal interruption"},
		147: {"P0556", "EMS:Brake booster pressure sensor / Brake booster pressure: the pressure is out of range"},
		148: {"P0559", "EMS:Brake booster pressure sensor / Brake booster pressure: lekage in brake booster or improper signal of brake booster pressure sensor"},
		149: {"P0700", "EMS:TCU error detected. TCU send a fault signal to ECU"},
		150: {"U0122", "EMS:CAN Communication Failure/ ESC node fault: No message received from ESC"},
		151: {"U0140", "EMS:CAN Communication Failure/CCN node fault: No message received from CCN (Instrument Panel Cluster)"},
		152: {"P0105", "EMS:Manifold Absolute pressure(MAP) /Manifold pressure: not plausible just after start. Stuck"},
		153: {"P0106", "EMS:Manifold Absolute pressure(MAP)  / Manifold pressure: Air pressure is not in the proper range"},
		154: {"P0073", "EMS:Ambient temperature sensor / Ambient temperature sensor: The sensor voltage is exceeded upper plausible value. It may be due to signal interruption, ground cable interruption or short circuit to power supply."},
		155: {"P0072", "EMS:Ambient temperature sensor / Ambient temperature sensor: The sensor voltage is exceeded upper plausible value. It may be due to signal interruption, ground cable interruption or short circuit to power supply."},
		156: {"P0121", "EMS:Throttle/Pedal PositionSensor/ Throttle valve 1st potentiometer: signal is not in proper range of the model."},
		157: {"P0221", "EMS:Throttle/Pedal PositionSensor/ Throttle valve 1st potentiometer: signal is not in proper range of the model."},
		158: {"P2102", "EMS:ETC Diagnosis / Throttle actuator control: throttle power stage short circuit."},
		159: {"P2103", "EMS:ETC Diagnosis / Throttle actuator control: throttle power stage is over heated or over current."},
		160: {"P2100", "EMS:ETC Diagnosis / ETC power stage: the throttle power stage open load."},
		161: {"P2108", "EMS:ETC Diagnosis / Spring check: Error in the return spring check"},
		162: {"P2118", "EMS:ETC Diagnosis / ETC control range: the throttle controller is exceeded permisible value for a short time."},
		163: {"P2172", "EMS:ETC Diagnosis /ETC UMA re-learning (for throttle adaptation, after ignition On, should be wait for 30 sec. till parameter counter for learning time for a learning step=11)"},
		164: {"P2173", "EMS::ETC Diagnosis / ETC adaptation abort because of environmental conditions(Max) (for throttle adaptation, after ignition On, should be wait for 30 sec. till parameter counter for learning time for a learning step=11)"},
		165: {"P2174", "EMS:ETC Diagnosis / ETC adaptation abort because of environmental conditions (Min) (for throttle adaptation, after ignition On, should be wait for 30 sec. till parameter counter for learning time for a learning step=11)"},
		166: {"P2176", "EMS: ETC Diagnosis /ETC failure during UMA learning,Throttle adaptation abort because of environmental conditions (for throttle adaptation, after ignition On, should be wait for 30 sec. till parameter counter for learning time for a learning step=11)"},
		167: {"P0320", "EMS:Cranckshaft position sensor /Reference mark (crankshaft sensor) : frequent correction by plus one tooth"},
		168: {"P0323", "EMS:Cranckshaft position sensor /Reference mark (crankshaft sensor) : frequent correction by minus one tooth"},
		169: {"P0322", "EMS:Cranckshaft position sensor /Reference mark (crankshaft sensor): reference mark is not found"},
		170: {"P0321", "EMS:Cranckshaft position sensor /Reference mark (crankshaft sensor): Frequent loss of the reference mark"},
		171: {"P0343", "EMS:Crankshaft/Camshaft /Camshaft sensor: No camshaft signal edge present, level on the input is high. May short circuit to power supply or signal interruption"},
		172: {"P0342", "EMS:Crankshaft/Camshaft /Camshaft sensor: No camshaft signal edge present, level on the input is low. May short circuit to ground"},
		173: {"P0012", "EMS:Crankshaft/Camshaft / Alignment between camshaft and crankshaft (< -10deg). camshaft signal is too retarded. It may be due to wrong timing, improper trigger wheel position, CVVT damage or any reason to make CVVT not fix at lock position."},
		174: {"P0727", "EMS:Crankshaft/Camshaft /  Cranckshaft sensor error. no signal from cranckshaft received. check RPM sensor or harness."},
		175: {"P0726", "EMS:Crankshaft/Camshaft /  Cranckshaft sensor error. Signal from RPM sensor is faulty.Check RPM sensor,harness, target wheel gap."},
		176: {"P2089", "EMS:CVVT / Camshaft control valve (CVVT): signal short circuit to power supply"},
		177: {"P2088", "EMS:CVVT / Camshaft control valve (CVVT): signal short circuit to ground"},
		178: {"P0010", "EMS:CVVT / Camshaft control valve (CVVT): signal cable interruption"},
		179: {"P0201", "EMS:Gasoline Injector /Gasoline injector /cyl1: interruption of signal cable"},
		180: {"P0203", "EMS:Gasoline Injector /Gasoline injector /cyl3: interruption of signal cable"},
		181: {"P0202", "EMS:Gasoline Injector /Gasoline injector /cyl2: interruption of signal cable"},
		182: {"P0573", "EMS:Brake switch / Brake pedal switch:both signals are true while acceleration"},
		183: {"P0572", "EMS:Brake switch / Brake pedal switch: one or both signals are false while deceleration"},
		184: {"P0704", "EMS:Clutch Switch Diagnosis / Clutch pedal switch signal: the fault of clutch switch is detected if there is no command from switch while gear changing.it may be due to weak connection or improper position of switch."},
		185: {"P0577", "EMS:Cruise Control Lever /Cruise control: Lever voltage is not in the range: SCG or SCB or OL., Lever voltage not in specified range"},
		186: {"P063D", "EMS:Generator PWM Signal/ Generator PWM signal: Generator PWM signal is exceeded 99 percent for a long time., Generator PWM signal exceeded 99%"},
		187: {"P063C", "EMS:Generator PWM signal: Generator PWM signal is below 0.5 percent for a specified time., Generator PWM signal below 0.5% for specified time"},
		188: {"P0327", "EMS:Knock Sensor/ Knock sensor: the engine reference noise is below lower plausible threshold. It may be due to sensor malfunction, improper mounting torque of sensor, weak connection of connector or interconnector and mechanical problem of engine systems., Engine reference noise below lower plausible threshold"},
		189: {"P0560", "EMS:System Voltage/ Power supply: ECU voltage supply is very low (below threshold 5V). Main relay and battery to be checked., ECU voltage supply very low"},
		190: {"P0691", "EMS:Cooling Fan Low Relay/ Fan relay control circuit low speed malfunction: short circuit to ground, Cooling fan relay control circuit low speed malfunction: short circuit to ground"},
		191: {"P0692", "EMS:Cooling Fan Low Relay/Fan relay control circuit low speed malfunction: short circuit to power supply, Cooling fan relay control circuit low speed malfunction: short circuit to power supply"},
		192: {"P0484", "EMS:Cooling Fan Diagnostic /Engine fan relay: the ECU command for low speed fan but fan works with high speed, Cooling fan diagnostic: low speed command, but fan works with high speed"},
		193: {"P0485", "EMS:Cooling Fan Diagnostic /Engine fan relay: the ECU command for low or high speed fan but fan is not activated, Cooling fan diagnostic: low/high speed command, but fan not activated"},
		194: {"P0483", "EMS:Cooling Fan Diagnostic /Engine fan relay: the ECU command for high speed fan but fan works with low speed, Cooling fan diagnostic: high speed command, but fan works with low speed"},
		195: {"P0036", "EMS:Downstream O2 Sensor/ Lambda sensor heater downstream catalyst power stage: signal interruption, Downstream O2 sensor heater signal interruption"},
		196: {"P0054", "EMS:Downstream O2 Sensor/ Lambda sensor heating downstream catalyst (Heater resistance) .The heater of LSF is unable to provide sufficient heating, Downstream O2 sensor heating insufficient"},
		197: {"P2232", "EMS:Downstream O2 Sensor/O2 sensor downstream: After LSF heater switch on, LSF output voltage increase more than 2 volt, may be due to coupling between heater and signal of LSF., Downstream O2 sensor voltage increase after heater switch on"},
		198: {"P0030", "EMS:O2 Sensor Up / Lambda sensor heater upstrem catalyst power stage: signal interruption, Upstream O2 sensor heater signal interruption"},
		199: {"P0053", "EMS:O2 Sensor Up / Lambda sensor heating upstrem catalyst. Even maximum duty cycle of the heater cannot heat the sensor properly (probably due to aging). Sensor resistance is high, Upstream O2 sensor heating insufficient"},
		200: {"P2231", "EMS:O2 Sensor Up / O2 sensor upstream: After LSF heater switch on, LSF output voltage increase more than 2 volt, may be due to coupling between heater and signal of LSF., Upstream O2 sensor voltage increase after heater switch on"},
		201: {"P0444", "EMS:Evaporative Emission Control System-Purge Control Valve Malfunction /Canister purge valve power stage: signal interruption, Evaporative Emission Control System-Purge Control Valve signal interruption"},
		202: {"P062F", "EMS:ECU Self Test/ECU EEPROM, ECU self-test EEPROM error"},
		203: {"P0605", "EMS:ECU Self Test/ECM Monitoring: Internal Control Module Read Only Memory (ROM) Error. Exchange of the control unit, ECM monitoring ROM error, exchange of control unit required"},
		204: {"P061A", "EMS:ECU Self Test / ECM Monitoring: Torque comparison from function monitoring. Exchange of the control unit., Torque comparison error from function monitoring, exchange of control unit required"},
		205: {"P061C", "EMS:ECU Self Test / ECM Monitoring: Plausibility check of the internal engine-speed calculation. Check engine speed sensor and wiring harness, Exchange of the control unit if necessary, Plausibility check error of internal engine-speed calculation, check engine speed sensor and wiring harness"},
		206: {"P061D", "EMS:ECU Self Test / ECM Monitoring: Monitoring the load signal. Check TMAP sensor and wiring harness, Exchange of the control unit if neccessary., Monitoring error of the load signal, check TMAP sensor and wiring harness, exchange of control unit if necessary"},
		207: {"P061B", "EMS:ECU Self Test / ECM Monitoring: Comparison of the two engine-internal load signals for consistency. Exchange of the control unit, Comparison error of two engine-internal load signals, exchange of control unit required"},
		208: {"P061E", "EMS:ECU Self Test / ECM Monitoring:Plausibility check of the ignition timing. Exchange of the control unit, Plausibility check error of ignition timing, exchange of control unit required"},
		209: {"P060A", "EMS:ECU Self Test / ECM Monitoring: Monitoring the fault reactions of function surveillance and other functions that lead to a reduction in the performance. Exchange of the control unit, Monitoring error of fault reactions, exchange of control unit required"},
		210: {"P060E", "EMS:ECU Self Test / ECM Monitoring:Monitoring the lower throttle-valve limit. Exchange of the control unit, Monitoring error of lower throttle-valve limit, exchange of control unit required"},
		211: {"P060E", "EMS:ECU Self Test / ECM Monitoring:Plausibility check of the ignition timing. Exchange of the control unit, Plausibility check error of ignition timing, exchange of control unit required"},
		212: {"P0601", "EMS:ECU Self Test / ECM Monitoring: Monitoring the variant coding. Checking the variant criteria by a check sum. Exchange of the control unit, Monitoring error of variant coding, exchange of control unit required"},
		213: {"P060C", "EMS:ECU Self Test / ECM Monitoring: Fault reaction demand from function monitoring: (safety fuel cut-off). Exchange of the control unit, Fault reaction demand error from function monitoring (safety fuel cut-off), exchange of control unit required"},
		214: {"P060D", "EMS:ECU Self Test / ECM Monitoring:Surveillance of the pedal sensor. Monitoring is by a comparison of both potentiometer voltages.Servicing: Check Pedal Module and wiring harness. Exchange of the control unit, Surveillance error of the pedal sensor, check pedal module and wiring harness, exchange of control unit required"},
		215: {"P0297", "EMS:vehicle speed Signal/ vehicle overspeed. Vehicle speed gets the max value, Vehicle speed signal indicates overspeed"},
		216: {"P0501", "EMS:vehicle speed Signal/ vehicle speed. Vehicle speed stuck, Vehicle speed signal indicates stuck"},
		217: {"P0500", "EMS:vehicle speed Signal/ Vehicle speed signal. CAN signal interruption or ABS fault report, Vehicle speed signal CAN interruption or ABS fault report"},
		218: {"P1636", "EMS:Immobilizer /Immobilizer: EMS key error. the received authentication result does not match, Immobilizer key error, authentication result mismatch"},
		219: {"P1637", "EMS:Immobilizer /Immobilizer:no response received from ICU/No authentication request was initiated, Immobilizer no response received from ICU"},
		220: {"P1623", "EMS:Immobilizer /Immobilizer: no SK stored in EMS and EMS is in virgin or neutral state, Immobilizer no SK stored in EMS and EMS in virgin or neutral state"},
		221: {"P0608", "EMS:ECU VSS supply/ECU VSS Output “A”, ECU VSS supply or output 'A' error"},
		222: {"P0609", "EMS:ECU VSS supply/ECU VSS Output “B”, ECU VSS supply or output 'B' error"},
		223: {"P0607", "EMS:ECU SPI BUS/Control Module Performance. ECU SPI BUS is faulty, ECU SPI BUS performance error"},
		224: {"P3352", "EMS:FSD CNG/Multiplicative mixture adaptation factor for CNG: Adaptation factor is exceeded upper plausible threshold (1.2). It may be due to CNG regulator malfunction or nonplausiblity of lambda sensor, intake manifold pressure sensor or rail pressure sensor, CNG adaptation factor exceeded upper plausible threshold"},
		225: {"P3353", "EMS:FSD CNG/Multiplicative mixture adaptation factor for CNG: Adaptation factor is below lower plausible threshold (0.8). It may be due to CNG regulator malfunction or nonplausiblity of lambda sensor, intake manifold pressure sensor or rail pressure sensor, CNG adaptation factor below lower plausible threshold"},
		226: {"P3350", "EMS:FSD CNG/Additive CNG mixture adaptation factor: Adaptation factor is exceeded upper plausible threshold (5%). It may be due to regulator malfunction, nonplausiblity of lambda sensor or intake manifold pressure sensor, air leakage, fuel leakage or exhaust leakage, CNG additive mixture adaptation factor exceeded upper plausible threshold"},
		227: {"P3351", "EMS:FSD CNG/Additive CNG mixture adaptation factor: Adaptation factor is below lower plausible threshold (-5%). It may be due to regulator malfunction, nonplausiblity of lambda sensor or intake manifold pressure sensor or fuel leakage, CNG additive mixture adaptation factor below lower plausible threshold"},
		228: {"P1898", "EMS:CNG/GASOLINE SWITCH OVER /Operation gasoline mode due to fault not possible: Fault is detected if engine doesn’t switch to gasoline after certain times (3 tries) due to not sufficient mixture, CNG to gasoline switch fault detected, insufficient mixture"},
		229: {"P189B", "EMS:CNG/GASOLINE SWITCH OVER /Start on gasoline mode due to fault not possible: Fault is detected if engine doesn’t start with gasoline after some tries, Start on gasoline mode fault detected, engine doesn't start with gasoline"},
		230: {"P189C", "EMS:CNG/GASOLINE SWITCH OVER /Operation CNG mode due to fault not possible: Fault is detected if engine doesn’t switch to CNG after certain times (3 tries) due to not sufficient mixture., CNG to gasoline switch fault detected, insufficient mixture"},
		231: {"P189F", "EMS:CNG/GASOLINE SWITCH OVER /Start on CNG mode due to fault not possible: Fault is detected if engine doesn’t start with CNG after some tries, Start on CNG mode fault detected, engine doesn't start with CNG"},
		232: {"P339A", "EMS:CNG shutoff valves /first tank solenoid valve: short circuit to power supply, CNG shutoff valves first tank solenoid valve short circuit to power supply"},
		233: {"P339B", "EMS:CNG shutoff valves /first tank solenoid valve: short circuit to ground,CNG shutoff valves first tank solenoid valve short circuit to ground"},
		234: {"P339C", "EMS:CNG shutoff valves /first tank solenoid valve: signal cable interruption,CNG shutoff valves first tank solenoid valve signal cable interruption"},
		235: {"P3380", "EMS:CNG shutoff valves /CNG regulator solenoid valve : short circuit to power supply, CNG shutoff valves CNG regulator solenoid valve short circuit to power supply"},
		236: {"P3381", "EMS:CNG shutoff valves /CNG regulator solenoid valve : short circuit to ground, CNG shutoff valves CNG regulator solenoid valve short circuit to ground"},
		237: {"P3379", "EMS:CNG shutoff valves /CNG regulator solenoid valve : signal cable interruption, CNG shutoff valves CNG regulator solenoid valve signal cable interruption"},
		238: {"P3383", "EMS:CNG Tank Pressure Sensor /CNG tank pressure sensor: sensor voltage is exceeded upper plausible value . short circuit to power supply or signal interruption,CNG tank pressure sensor voltage exceeded upper plausible value, short circuit or signal interruption"},
		239: {"P3384", "EMS:CNG Tank Pressure Sensor /CNG tank pressure sensor: sensor voltage is exceeded upper plausible value . short circuit to ground, CNG tank pressure sensor voltage exceeded upper plausible value, short circuit to ground"},
		240: {"P3366", "EMS:CNG Rail Pressure / CNG rail pressure system: the rail pressure is exceeded upper plausible threshold . It may be due to regulator malfunction (pressure increase Gradually) or wiring harness problem (pressure increase suddenly due to voltage change)"},
		241: {"P3367", "EMS:CNG Rail Pressure / CNG rail pressure system: the rail pressure is below lower plausible threshold while tank pressure is above. It may be due to regulator malfunction or wiring harness (pressure decrease suddenly due to voltage change)"},
		242: {"P3386", "EMS:CNG Rail Pressure / CNG rail pressure sensor: the rail pressure sensor voltage is exceeded upper plausible value .It may be short circuit to power supply or signal interruption."},
		243: {"P3387", "EMS:CNG Rail Pressure / CNG rail pressure sensor: the rail pressure sensor voltage is below lower plausible value .It may be short circuit to ground."},
		244: {"P336B", "EMS:CNG Rail Tempreture / CNG rail temperature sensor: the sensor voltage is exceeded upper plausible threshold. It may be due to signal or ground cable interruption or short circuit to power supply"},
		245: {"P336C", "EMS:CNG Rail Tempreture / CNG rail temperature sensor: the sensor voltage is below lower plausible threshold. It may be due to short circuit to ground"},
		246: {"P336E", "EMS:CNG Rail Tempreture / CNG rail temperature sensor: the rail temperature is more than maximum of coolant temperature and ambient temprature (10C) after cold start.It may be due to rail temprature sensor malfunction if coolant and ambient temprature sensors are valid"},
		247: {"P336D", "EMS:CNG Rail Tempreture / CNG rail temperature sensor: the engine temperature is increased by a certain value (70C) but rail temperature is not changed obviously. It may be due to sensor malfunction."},
		248: {"P3389", "EMS:CNG Injector/CNG injector /cyl1: signal short circuit to power supply"},
		249: {"P3389", "EMS:CNG Injector/CNG injector /cyl1: signal short circuit to ground"},
		250: {"P3388", "EMS:CNG Injector/CNG injector /cyl1: interruption of signal cable"},
		251: {"P3392", "EMS:CNG Injector/CNG injector /cyl3: signal short circuit to power supply"},
		252: {"P3393", "EMS:CNG Injector/CNG injector /cyl3: signal short circuit to ground"},
		253: {"P3391", "EMS:CNG Injector/CNG injector /cyl3: interruption of signal cable"},
		254: {"P3395", "EMS:CNG Injector/CNG injector /cyl4: signal short circuit to power supply"},
		255: {"P3396", "EMS:CNG Injector/CNG injector /cyl4: signal short circuit to ground"},
		256: {"P3394", "EMS:CNG Injector/CNG injector /cyl4: interruption of signal cable"},
		257: {"P3398", "EMS:CNG Injector/CNG injector /cyl2: signal short circuit to power supply"},
		258: {"P3399", "EMS:CNG Injector/CNG injector /cyl2: signal short circuit to ground"},
		259: {"P3397", "EMS:CNG Injector/CNG injector /cyl2: interruption of signal cable"},
		260: {"P3314", "EMS:CNG Leakage/CNG internal leakage through pressure regulator valve: tank pressure loss is lower than plausible value while running on CNG and shutoff valve is closed for a short time"},
		261: {"P3313", "EMS:CNG Leakage/CNG internal leakage through tank valve: tank pressure loss is lower than plausible value while running on CNG and shutoff valve is closed for a short time"},
		262: {"P3321", "EMS:CNG Leakage/CNG leak from Low pressure system:While SOVs closed, CNG rail mass reduction in a short time due to large leak. It also can occur due to wiring harness problem (pressure decrease suddenly since of voltage change)"},
		263: {"P3322", "EMS:CNG Leakage/CNG leak from Low pressure system: While SOVs closed, CNG rail mass reduction in a long time due to fine leak. It also can occur due to wiring harness problem (pressure decrease suddenly since of voltage change)"},
		264: {"P3325", "EMS:CNG Leakage/CNG leak from high pressure system: While running on CNG, CNG pressure loss is exceeded permisible value due to large leak (continous reduction).It also can occur due to wiring harness problem (pressure decrease suddenly since of voltage change) or CNG regulator solenoid valve."},
		265: {"P3326", "EMS:CNG Leakage/CNG fine leak from high pressure system: While running on CNG, CNG tank mass reduction is exceeded permisible value due to fine leak."},
		266: {"P1300", "EMS:Misfire which cause emission high(CNG) / Random/Multiple Cylinder Misfire Detected"},
		267: {"P1301", "EMS:Misfire which cause emission high(CNG) / Cylinder 1 Misfire Detected"},
		268: {"P1302", "EMS:Misfire which cause emission high(CNG) / Cylinder 2 Misfire Detected"},
		269: {"P1303", "EMS:Misfire which cause emission high(CNG) / Cylinder 3 Misfire Detected"},
		270: {"P1304", "EMS:Misfire which cause emission high(CNG) / Cylinder 4 Misfire Detected"},
		271: {"P0140", "EMS:Downstream O2 sensor / O2 sensor signal plausibility diagnosis (Sensor 2)"},
		272: {"P0642", "EMS:Sensor reference voltage / Sensor supply voltage low for ETC, PVS1"},
		273: {"P0643", "EMS:Sensor reference voltage / Sensor supply voltage high for ETC, PVS1"},
		274: {"P0652", "EMS:Sensor reference voltage / Sensor supply voltage low for MAP, PVS2"},
		275: {"P0653", "EMS:Sensor reference voltage / Sensor supply voltage HIGH for MAP,PVS2"},
		276: {"P0119", "EMS:Engine Coolant Temperature (TCO) /Plausibility check"},
		277: {"P0204", "EMS:Gasoline Injector /Electrical Check"},
		278: {"P1340", "EMS:Phase signal malfunction / Plausibility check"},
		279: {"P0693", "EMS:Cooling fan high relay / Electrical Check"},
		280: {"P0694", "EMS:Cooling fan high relay / Electrical Check"},
		281: {"P0171", "EMS:FSD / System Too Lean"},
		282: {"P0172", "EMS:FSD / System Too Rich"},
		283: {"P1610", "EMS:Immobilizer ECM configuration failure/Immobiliser ECM config failure"},
		284: {"P1611", "EMS:Security code input error / Wrong security code(Access code) of immobilizer entered to ECM"},
		285: {"P1612", "EMS:Timeout of No response from ICU /ICU message Timeout or No response from ICU to ECM"},
		286: {"P1613", "EMS:ICU response failure /Authentication failure detected by ICU"},
		287: {"P1614", "EMS:ECM & ICU encryption of authentication failed"},
		288: {"P063E", "EMS:ETC Diagnosis / TPS adaptation diagnosis"},
		289: {"P2120", "EMS:Throttle/ Pedal position sensor /Accelerator pedal disconnect"},
		290: {"P0552", "EMS:Power steering pressure switch /Diagnosis for power steering pressure switch for short circuit to ground"},
		291: {"P0566", "EMS:Cruise control switch / Cruise CANCEL switch stuck"},
		292: {"U0415", "EMS:CAN communication failure/Invalid signal from ABS"},
		293: {"P0217", "EMS:Engine Coolant Temperature performance / Engine Coolant Temperature Sensor 1 Circuit Range/Performance"},
		294: {"P0111", "EMS:Intake Air Temperature Sensor/Crcuit Range/PerformancePlausibility"},
		295: {"P2620", "EMS:ETC Diagnosis / Throttle Actuator Control"},
		296: {"P0690", "EMS:Power Relay/ System Voltage"},
		297: {"P0506", "EMS:Idle Speed / Too Low Idle speed"},
		298: {"P0326", "EMS:Knock Sensor Circuit Malfunction/ Knock Sensor 1 Circuit Range/Performance (Single sensor) - Oscillation check"},
		299: {"P0215", "EMS:Crash Detection/ Engine Shutoff Solenoid"},
		300: {"P1667", "EMS:Imobilizer / Circuit malfunction immobilizer"},
		301: {"P1656", "EMS:Imobilizer / Bitfail in last byte / before last byte"},
		302: {"P1630", "EMS:Imobilizer /TP (Transponder) virgin"},
		303: {"P1628", "EMS:Imobilizer/ Single wire error"},
		304: {"P0075", "EMS:VVT mechanical reference diagnosis/ VVT deactivated, camshaft in default position"},
		305: {"P0076", "EMS:VVT mechanical reference diagnosis/ VVT deactivated, camshaft in default position"},
		306: {"P0077", "EMS:VVT mechanical reference diagnosis/ VVT deactivated, camshaft in default position"},
		307: {"P0688", "EMS:Power Relay / ECM/PCM Power Relay Sense Circuit/Open"},
		308: {"P0525", "EMS:Cruise Control / Cruise Control Servo Control Circuit Range/Performance"},
		309: {"P0576", "EMS:Cruise Control / Cruise Control Input Circuit Low"},
		310: {"P0328", "EMS:Knock Sensor Circuit Malfunction / Knock Sensor 1 Circuit High"},
		311: {"P0170", "EMS:FSD/ Fuel Trim,Bank1 Malfunction"},
		312: {"P1629", "EMS:Immobilizer/ No challenge from immobiliser/Immobiliser communication:timeout"},
		313: {"P1621", "EMS:Immobilizer / Wrong PIN/Incorrect Immobilizer Key"},
		314: {"P1622", "EMS:Immobilizer / Wrong key/Auth. NOK, Key Learning process NOK"},
		315: {"P1624", "EMS:Immobilizer / Immo not used/Immo is disabled"},
		316: {"P0219", "EMS:Plausibiltity check of exceed maximum engine speed/Engine Overspeed Condition"},
		317: {"P1336", "EMS:ETC Torque Limitation level1 / Engine torque control Adaption at limit(ETC safety monitoring)"},
		318: {"P0507", "EMS:Idle Speed Control / Idle Air Control System RPM Higher Than Expected"},
	}
	if data, ok := dataMap[num]; ok {
		return data.Value, data.Desc
	}

	return "", ""
}

func getBit(byteValue byte, bitPosition uint) int {
	// Shift the bit to the rightmost position
	shiftedBit := byteValue >> bitPosition
	// Use bitwise AND with 1 to extract the rightmost bit
	result := int(shiftedBit & 1)
	return result
}
