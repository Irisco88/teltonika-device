package parser

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/exp/constraints"
	"math"
	"strconv"
	"time"
)

func streamToInt32(data []byte) (int32, error) {
	var y int32
	err := binary.Read(bytes.NewReader(data), binary.BigEndian, &y)
	if y>>31 == 1 {
		y *= -1
	}
	return y, err
}

func streamToNumber[T constraints.Integer | constraints.Float](data []byte) (T, error) {
	var result T
	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &result); err != nil {
		return *new(T), err
	}
	return result, nil
}

func numberToStream(value any) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func twosComplement(input int32) int32 {
	mask := int32(math.Pow(2, 31))
	return -(input & mask) + (input &^ mask)
}

func streamToTime(data []byte) (int64, error) {
	milliSecond, err := streamToNumber[int64](data)
	seconds := int64(float64(milliSecond) / 1000.0)
	nanoseconds := int64(milliSecond % 1000)

	return time.Unix(seconds, nanoseconds).Unix(), err
}

func DecodeIMEI(data []byte) (string, error) {
	if len(data) < 2 {
		return "", errors.New("invalid imei bytes length")
	}
	imeiLenHex := hex.EncodeToString(data[0:2])
	imeiLength, err := strconv.ParseInt(imeiLenHex, 16, 64)
	if err != nil {
		return "", fmt.Errorf("decode imei len error %s", err.Error())
	}
	imeiHex := hex.EncodeToString(data[2:])
	imeiBytes, err := hex.DecodeString(imeiHex)
	if err != nil {
		return "", fmt.Errorf("decode error %s", err.Error())
	}
	imei := string(imeiBytes)
	if len(imei) != int(imeiLength) {
		return "", fmt.Errorf("invalid imei length")
	}
	return imei, nil
}

func calculateCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF) // Initial CRC value

	for _, b := range data {
		crc ^= uint16(b)

		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}
