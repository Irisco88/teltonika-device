package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	pb "github.com/irisco88/protos/gen/device/v1"
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
			elementItem.ElementName = "Vehicle Speed"
			elementItem.ElementValue = elementIntValue
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
			elementItem.ElementName = "Engine Coolant Temperature"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 48) / (143.5 + 48)
			elementItem.ColorValue = "#065535"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := (float64(byte5)) * 0.390625
			elementItem.ElementName = "Fuel level in tank"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
		if eightbytes[6] == byte1 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "CheckEngine "
			elementItem0.ElementValue = float64(getBit(byte6, 0))
			elementItem0.NormalValue = float64(getBit(byte6, 0))
			elementItem0.ColorValue = "#ff80ed"
			values = append(values, &elementItem0)
			var elementItem1 pb.IOElement
			elementItem1.ElementName = "AirConditionPressureSwitch1 "
			elementItem1.ElementValue = float64(getBit(byte6, 1))
			elementItem1.NormalValue = float64(getBit(byte6, 1))
			elementItem1.ColorValue = "#198ba3"
			values = append(values, &elementItem1)
			var elementItem2 pb.IOElement
			elementItem2.ElementName = "AirConditionPressureSwitch2 "
			elementItem2.ElementValue = float64(getBit(byte6, 2))
			elementItem2.NormalValue = float64(getBit(byte6, 2))
			elementItem2.ColorValue = "#ae0e52"
			values = append(values, &elementItem2)
			//34
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearShiftindicator "
			elementItem3.ElementValue = float64(int((byte6 & 0x18) >> 3))
			elementItem3.NormalValue = 1000
			elementItem3.ColorValue = ""
			values = append(values, &elementItem3)
			//567
			var elementItem4 pb.IOElement
			elementItem4.ElementName = "DesiredGearValue "
			elementItem4.ElementValue = float64(int((byte6 & 0xe0) >> 5))
			elementItem4.NormalValue = 1000
			elementItem4.ColorValue = ""
			values = append(values, &elementItem4)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{eightbytes[7], 0, 0, 0, 0, 0, 0, 0}))
			elementIntValue := float64(byte7)
			elementItem.ElementName = "Vehicle Type"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)
		}
	case 146:
		if eightbytes[0] == byte7 {
			var elementItem0 pb.IOElement
			elementItem0.ElementName = "Condition immobilizer"
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
			elementItem2.ElementName = "ClutchPedalStatus "
			elementItem2.ElementValue = float64(getBit(byte0, 5))
			elementItem2.NormalValue = float64(getBit(byte0, 5))
			elementItem2.ColorValue = "#282a36"
			values = append(values, &elementItem2)
			//67
			var elementItem3 pb.IOElement
			elementItem3.ElementName = "GearEngagedStatus "
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
			elementItem.ElementName = "Engine Friction Torque"
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
			elementItem0.ElementName = "CruiseControlOn_Off "
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
			elementItem2.ElementName = "condition cruise control lamp "
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
			elementItem4.ElementName = "Condition catalyst heating activated"
			elementItem4.ElementValue = float64(getBit(byte6, 4))
			elementItem4.NormalValue = float64(getBit(byte6, 4))
			elementItem4.ColorValue = "#00FF00"
			values = append(values, &elementItem4)

			var elementItem5 pb.IOElement
			elementItem5.ElementName = "AC compressor status"
			elementItem5.ElementValue = float64(getBit(byte6, 5))
			elementItem5.NormalValue = float64(getBit(byte6, 5))
			elementItem5.ColorValue = "#FF0000"
			values = append(values, &elementItem5)

			var elementItem6 pb.IOElement
			elementItem6.ElementName = "Condition main relay(Starter Relay)"
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
			elementItem0.ElementName = "Reserve "
			elementItem0.ElementValue = float64(getBit(byte7, 0))
			elementItem0.NormalValue = 1000
			elementItem0.ColorValue = ""
			values = append(values, &elementItem0)

			var elementItem1 pb.IOElement
			elementItem1.ElementName = "Reserve "
			elementItem1.ElementValue = float64(getBit(byte7, 1))
			elementItem1.NormalValue = 1000
			elementItem1.ColorValue = ""
			values = append(values, &elementItem1)

			var elementItem2 pb.IOElement
			elementItem2.ElementName = "Reserve "
			elementItem2.ElementValue = float64(getBit(byte7, 2))
			elementItem2.NormalValue = 1000
			elementItem2.ColorValue = ""
			values = append(values, &elementItem2)

			var elementItem3 pb.IOElement
			elementItem3.ElementName = "Reserve "
			elementItem3.ElementValue = float64(getBit(byte7, 3))
			elementItem3.NormalValue = 1000
			elementItem3.ColorValue = ""
			values = append(values, &elementItem3)

			var elementItem4 pb.IOElement
			elementItem4.ElementName = "Reserve "
			elementItem4.ElementValue = float64(getBit(byte7, 4))
			elementItem4.NormalValue = 1000
			elementItem4.ColorValue = ""
			values = append(values, &elementItem4)

			var elementItem5 pb.IOElement
			elementItem5.ElementName = "Reserve "
			elementItem5.ElementValue = float64(getBit(byte7, 5))
			elementItem5.NormalValue = 1000
			elementItem5.ColorValue = ""
			values = append(values, &elementItem5)

			var elementItem6 pb.IOElement
			elementItem6.ElementName = "Reserve "
			elementItem6.ElementValue = float64(getBit(byte7, 6))
			elementItem6.NormalValue = 1000
			elementItem6.ColorValue = ""
			values = append(values, &elementItem6)

			var elementItem7 pb.IOElement
			elementItem7.ElementName = "Reserve "
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
			elementItem.ElementName = "Intake air temperature"
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
			elementItem.ElementName = "Oil temperature(TCU)"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 40) / (214 + 40)
			elementItem.ColorValue = "#0000FF"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := ((float64(byte3)) * 0.5) - 40
			elementItem.ElementName = "Ambient air temperature"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue + 40) / (86.5 + 40)
			elementItem.ColorValue = "#008080"
			values = append(values, &elementItem)

		}
		if eightbytes[4] == byte3 {
			var elementItem pb.IOElement
			//elementIntValue := float64(binary.BigEndian.Uint64([]byte{0, 0, 0, eightbytes[4], 0, 0, 0, 0}))
			elementIntValue := float64(byte4)
			elementItem.ElementName = "Number of DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
			values = append(values, &elementItem)

		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(byte5)
			elementItem.ElementName = "EMS_DTC"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = 1000
			elementItem.ColorValue = ""
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
			elementItem.ColorValue = ""
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
			elementItem.ElementName = "Sensor1"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 0.7
			elementItem.ColorValue = "#008000"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor2"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 1) / 0.7
			elementItem.ColorValue = "#808000"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor3"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 2) / 0.7
			elementItem.ColorValue = "#800000"
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor4"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 3) / 0.7
			elementItem.ColorValue = "#398112"
			values = append(values, &elementItem)
		}
	case 151:
		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementItem.ElementName = "Sensor5"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 4) / 0.7
			elementItem.ColorValue = "#12815E"
			values = append(values, &elementItem)
		}
		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor6"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 5) / 0.7
			elementItem.ColorValue = "#125781"
			values = append(values, &elementItem)
		}
		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor7"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 6) / 0.7
			elementItem.ColorValue = "#7E1281"
			values = append(values, &elementItem)
		}
		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor8"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#811241"
			values = append(values, &elementItem)
		}
	case 152:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementItem.ElementName = "Sensor9"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = elementIntValue / 0.7
			elementItem.ColorValue = "#817C12"
			values = append(values, &elementItem)
		}

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor10"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 1) / 0.7
			elementItem.ColorValue = "#F4E60E"
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor11"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 2) / 0.7
			elementItem.ColorValue = "#0E99F4"
			values = append(values, &elementItem)
		}

		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor12"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 3) / 0.7
			elementItem.ColorValue = "#F40EED"
			values = append(values, &elementItem)
		}

	case 153:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementItem.ElementName = "Sensor13"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 4) / 0.7
			elementItem.ColorValue = "#FF6C00"
			values = append(values, &elementItem)

		}

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor14"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 5) / 0.7
			elementItem.ColorValue = "#00FF55"
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor15"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 6) / 0.7
			elementItem.ColorValue = "#9B00FF"
			values = append(values, &elementItem)
		}

		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor16"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#FF008F"
			values = append(values, &elementItem)
		}

	case 154:

		if eightbytes[1] == byte6 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte1, byte0}))
			elementItem.ElementName = "Sensor17"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#51022E"
			values = append(values, &elementItem)
		}

		if eightbytes[3] == byte4 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte3, byte2}))
			elementItem.ElementName = "Sensor18"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#02513A"
			values = append(values, &elementItem)
		}

		if eightbytes[5] == byte2 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte5, byte4}))
			elementItem.ElementName = "Sensor19"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#512B02"
			values = append(values, &elementItem)
		}

		if eightbytes[7] == byte0 {
			var elementItem pb.IOElement
			elementIntValue := float64(binary.BigEndian.Uint16([]byte{byte7, byte6}))
			elementItem.ElementName = "Sensor20"
			elementItem.ElementValue = elementIntValue
			elementItem.NormalValue = (elementIntValue - 7) / 0.7
			elementItem.ColorValue = "#A41B9E"
			values = append(values, &elementItem)
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

func getBit(byteValue byte, bitPosition uint) int {
	// Shift the bit to the rightmost position
	shiftedBit := byteValue >> bitPosition
	// Use bitwise AND with 1 to extract the rightmost bit
	result := int(shiftedBit & 1)
	return result
}
