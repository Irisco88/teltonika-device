package server

import (
	"context"
	"github.com/openfms/teltonika-device/parser"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/openfms/teltonika-device/db/clickhouse/mock_db"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

func TestSendData(t *testing.T) {
	nowTime := uint64(time.Now().UnixMilli())
	natsServer := RunNatsServerOnPort(0)
	defer natsServer.Shutdown()
	tests := map[string]struct {
		imei       string
		points     []*parser.AVLData
		MockDB     func(ctx context.Context, dbConn *mockdb.MockAVLDBConn)
		logEntry   []*zapcore.Entry
		logContext []map[string]interface{}
		errWant    error
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

			ctrl := gomock.NewController(t)
			dbConn := mockdb.NewMockAVLDBConn(ctrl)

			observerlog, out := observer.New(zap.InfoLevel)
			logger := zap.New(observerlog)

			natsClient := NewNatsConnection(t, natsServer.ClientURL())
			server := NewServer(serverConn.LocalAddr().String(), logger, natsClient, dbConn).(*TeltonikaServer)
			go func() {
				server.wg.Add(1)
				server.HandleConnection(serverConn)
			}()
			ImeiAuthenticate(t, clientConn, test.imei)
			SendPoints(t, clientConn, test.points)
			logs := out.TakeAll()
			assert.Assert(t, len(logs) == len(test.logEntry))
			for i, log := range logs {
				assert.Assert(t, log.Level == test.logEntry[i].Level)
				assert.Equal(t, log.Message, test.logEntry[i].Message)
				assert.DeepEqual(t, log.ContextMap(), test.logContext[i], protocmp.Transform())
			}
		})
	}
}
