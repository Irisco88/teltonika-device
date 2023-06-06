package server

import (
	"github.com/packetify/teltonika-device/server/parser"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"
	"net"
	"testing"
	"time"
)

func TestSendData(t *testing.T) {
	nowTime := uint64(time.Now().UnixMilli())

	tests := map[string]struct {
		imei   string
		points []*parser.AVLData
	}{
		"success": {
			imei: "356478954125698",
			points: []*parser.AVLData{
				{
					Timestamp:  nowTime,
					Priority:   parser.PriorityPanic,
					Longitude:  -31.867449,
					Latitude:   135.303686,
					Altitude:   27,
					Angle:      112,
					Satellites: 32,
					Speed:      120,
					EventID:    36,
					IOElements: []*parser.IOElement{
						{ID: 1, Value: uint32(500)},
						{ID: 2, Value: true},
						{ID: 3, Value: uint8(54)},
					},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			clientConn, serverConn := net.Pipe()

			logger, _ := zap.NewDevelopment()
			server := NewServer(serverConn.LocalAddr().String(), logger).(*TeltonikaServer)
			go func() {
				server.wg.Add(1)
				server.HandleConnection(serverConn)
			}()

			imeiBytes, err := parser.EncodeIMEIToHex(test.imei)
			assert.NilError(t, err)
			_, err = clientConn.Write(imeiBytes)
			assert.NilError(t, err)
			buf := make([]byte, 2048)
			_, err = clientConn.Read(buf)
			assert.NilError(t, err)
			assert.DeepEqual(t, buf[:1], []byte{1})
			packetBytes, err := parser.MakeCodec8Packet(test.points)
			assert.NilError(t, err)
			_, err = clientConn.Write(packetBytes)
			assert.NilError(t, err)
			_, err = clientConn.Read(buf)
			assert.NilError(t, err)
			assert.DeepEqual(t, buf[:4], []byte{0, 0, 0, uint8(len(test.points))})
		})
	}
}
