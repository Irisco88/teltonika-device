package parser

import (
	"gotest.tools/v3/assert"
	"testing"
	"time"
)

func TestEncodeAVLData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		point := &AVLData{
			Timestamp:  uint64(time.Now().UnixMilli()),
			Priority:   priorityHigh,
			Longitude:  -31.867449,
			Latitude:   135.303686,
			Altitude:   451,
			Angle:      654,
			Satellites: 20,
			Speed:      457,
			EventID:    451,
			IOElements: []*IOElement{
				{ID: 1, Value: uint32(500)},
				{ID: 2, Value: true},
				{ID: 3, Value: uint8(54)},
			},
		}
		pointBytes, err := MakeCodec8Packet([]*AVLData{point})
		assert.NilError(t, err)
		avlData, err := ParsePacket(pointBytes, "547865412456987452")
		assert.NilError(t, err)
		t.Log(avlData)
	})

}
