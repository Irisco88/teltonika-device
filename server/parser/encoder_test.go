package parser

import (
	"testing"
	"time"

	"github.com/openfms/teltonika-device/proto/pb"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

func TestEncodeAVLData(t *testing.T) {
	nowTime := uint64(time.Now().UnixMilli())
	twoHourLater := uint64(time.Now().Add(time.Hour * 2).UnixMilli())
	tests := map[string]struct {
		points       []*AVLData
		wantedPoints []*pb.AVLData
		errWant      error
		imei         string
	}{
		"success": {
			imei: "547865412456987452",
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
			wantedPoints: []*pb.AVLData{
				{
					Imei:      "547865412456987452",
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
		"success multiple points": {
			imei: "547865412456987452",
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
				{
					Timestamp:  twoHourLater,
					Priority:   PriorityLow,
					Longitude:  -20.867449,
					Latitude:   60.303786,
					Altitude:   36,
					Angle:      12,
					Satellites: 5,
					Speed:      99,
					EventID:    57,
					IOElements: []*IOElement{
						{ID: 1, Value: uint32(951)},
						{ID: 39, Value: uint16(742)},
						{ID: 7, Value: true},
						{ID: 24, Value: uint64(69874)},
					},
				},
			},
			wantedPoints: []*pb.AVLData{
				{
					Imei:      "547865412456987452",
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
				{
					Imei:      "547865412456987452",
					Timestamp: twoHourLater,
					Priority:  pb.PacketPriority_PACKET_PRIORITY_LOW,
					EventId:   57,
					Gps: &pb.GPS{
						Latitude:   60.303786,
						Longitude:  -20.867449,
						Speed:      99,
						Altitude:   36,
						Satellites: 5,
						Angle:      12,
					},
					IoElements: []*pb.IOElement{
						{ElementId: 1, Value: 951},
						{ElementId: 7, Value: 1},
						{ElementId: 24, Value: 69874},
						{ElementId: 39, Value: 742},
					},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			pointBytes, err := MakeCodec8Packet(test.points)
			assert.NilError(t, err)
			avlData, err := ParsePacket(pointBytes, test.imei)
			assert.NilError(t, err)
			assert.Equal(t, len(avlData), len(test.wantedPoints))
			assert.DeepEqual(t, avlData, test.wantedPoints, protocmp.Transform())
		})
	}
}
