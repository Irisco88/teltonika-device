package parser

import (
	"encoding/hex"
	"github.com/packetify/teltonika-device/proto/pb"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
	"testing"
)

func TestParsePacket(t *testing.T) {
	tests := map[string]struct {
		imei       string
		dataString string
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
							ElementId: 17,
							Value:     29,
						},
						{
							ElementId: 16,
							Value:     22949000,
						},
						{
							ElementId: 11,
							Value:     893700218,
						},
						{
							ElementId: 14,
							Value:     500686954,
						},
					},
					EventId: 1,
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dBytes, err := hex.DecodeString(test.dataString)
			assert.NilError(t, err)
			points, err := ParsePacket(dBytes, test.imei)
			assert.NilError(t, err)
			assert.Equal(t, len(points), len(test.expected))
			assert.DeepEqual(t, points, test.expected, protocmp.Transform())
		})
	}
}
