package parser

import (
	"encoding/hex"
	"testing"
	"time"

	pb "github.com/irisco88/protos/gen/device/v2"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

func TestParsePacket(t *testing.T) {
	nowTime := uint64(time.Now().UnixMilli())
	tests := map[string]struct {
		imei       string
		dataString string
		points     []*AVLData
		expected   []*pb.AVLData
	}{
		"success": {
			imei:       "546897541245687",
			dataString: `000000000000004A8E010000016B412CEE000100000000000000000000000000000000010005000100010100010011001D00010010015E2C880002000B000000003544C87A000E000000001DD7E06A00000100002994`,
			expected: []*pb.AVLData{
				{
					Imei:      "546897541245687",
					Timestamp: 1560166592000,
					Priority:  pb.PacketPriority_PACKET_PRIORITY_HIGH,
					Gps:       &pb.GPS{},
					IoElements: []*pb.IOElement{
						{
							ElementId: 1,
							Value:     1,
						},
						{
							ElementId: 11,
							Value:     893700218,
						},
						{
							ElementId: 14,
							Value:     500686954,
						},
						{
							ElementId: 16,
							Value:     22949000,
						},
						{
							ElementId: 17,
							Value:     29,
						},
					},
					EventId: 1,
				},
			},
		},
		"success points": {
			imei: "587414569874521",
			points: []*AVLData{
				{
					Timestamp:  nowTime,
					Priority:   PriorityHigh,
					Longitude:  -31.867449,
					Latitude:   135.303686,
					Altitude:   27,
					Angle:      112,
					Satellites: 32,
					Speed:      120,
					EventID:    36,
					IOElements: []*IOElement{
						{ID: 1, Value: uint32(500)},
						{ID: 2, Value: true},
						{ID: 3, Value: uint8(54)},
					},
				},
			},
			expected: []*pb.AVLData{
				{
					Imei:      "587414569874521",
					Timestamp: nowTime,
					Priority:  pb.PacketPriority_PACKET_PRIORITY_HIGH,
					EventId:   36,
					Gps: &pb.GPS{
						Latitude:   135.303686,
						Longitude:  -31.867449,
						Speed:      120,
						Altitude:   27,
						Satellites: 32,
						Angle:      112,
					},
					IoElements: []*pb.IOElement{
						{ElementId: 1, Value: 500},
						{ElementId: 2, Value: 1},
						{ElementId: 3, Value: 54},
					},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				e         error
				dataBytes []byte
			)

			if len(test.dataString) > 0 {
				dataBytes, e = hex.DecodeString(test.dataString)
				assert.NilError(t, e)
			} else {
				dataBytes, e = MakeCodec8Packet(test.points)
				assert.NilError(t, e)
			}
			points, err := ParsePacket(dataBytes, test.imei)
			assert.NilError(t, err)
			assert.Equal(t, len(points), len(test.expected))
			assert.DeepEqual(t, points, test.expected, protocmp.Transform())
		})
	}
}
