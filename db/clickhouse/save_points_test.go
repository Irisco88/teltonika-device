package clickhouse

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/openfms/protos/gen/go/device/v1"
	"gotest.tools/v3/assert"
)

func NewConnTest(t *testing.T) AVLDBConn {
	avlDB, err := ConnectAvlDB(os.Getenv("AVLDB_CLICKHOUSE"))
	assert.NilError(t, err)
	return avlDB
}
func TestAVLDataBase_SaveAvlPoints(t *testing.T) {
	dbConn := NewConnTest(t)
	tests := map[string]struct {
		errWant error
		points  []*pb.AVLData
		ctx     func() context.Context
	}{
		"success": {
			errWant: nil,
			points: []*pb.AVLData{
				{
					Imei:      "457845652414565",
					Timestamp: uint64(time.Now().UnixMilli()),
					Priority:  pb.PacketPriority_PACKET_PRIORITY_HIGH,
					EventId:   47,
					Gps: &pb.GPS{
						Longitude:  25.451,
						Latitude:   31.654,
						Altitude:   451,
						Angle:      45,
						Speed:      362,
						Satellites: 23,
					},
					IoElements: []*pb.IOElement{
						{ElementId: 87, Value: 23205},
						{ElementId: 2, Value: 785},
					},
				},
				{
					Imei:      "564123654789541",
					Timestamp: uint64(time.Now().UnixMilli()),
					Priority:  pb.PacketPriority_PACKET_PRIORITY_LOW,
					EventId:   12,
					Gps: &pb.GPS{
						Longitude:  28.451,
						Latitude:   16.654,
						Altitude:   12,
						Angle:      34,
						Speed:      93,
						Satellites: 13,
					},
					IoElements: []*pb.IOElement{
						{ElementId: 1, Value: 125},
						{ElementId: 3, Value: 56},
					},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if test.ctx != nil {
				ctx = test.ctx()
			}
			err := dbConn.SaveAvlPoints(ctx, test.points)
			if test.errWant != nil {
				assert.ErrorIs(t, err, test.errWant)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
