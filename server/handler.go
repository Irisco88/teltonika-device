package server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	pb "github.com/irisco88/protos/gen/device/v1"
	"github.com/irisco88/teltonika-device/parser"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
)

func (ts *TeltonikaServer) HandleConnection(conn net.Conn) {
	defer conn.Close()
	defer ts.wg.Done()
	authenticated := false
	var imei string
	for {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 2048)

		// Read the incoming connection into the buffer.
		size, err := conn.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				ts.log.Error("read failed", zap.Error(err))
			}
			return
		}
		if !authenticated {
			imei, err = parser.DecodeIMEI(buf[:size])
			if err != nil {
				ts.log.Error("decode imei failed", zap.Error(err))
				return
			}
			ts.log.Info("Data received",
				zap.String("ip", conn.RemoteAddr().String()),
				zap.Int("size", size),
				zap.String("imei", imei),
			)
			ts.ResponseAcceptIMEI(conn)
			authenticated = true
			continue
		}
		ctx := context.Background()

		go func() {
			if rawDataErr := ts.avlDB.SaveRawData(ctx, imei, hex.EncodeToString(buf)); rawDataErr != nil {
				ts.log.Error("save raw data failed", zap.Error(rawDataErr))
			} else {
				ts.log.Info("rawData:",
					zap.Any("raw:", hex.EncodeToString(buf)),
				)
			}
		}()

		points, err := parser.ParsePacket(buf, imei)
		if err != nil {
			ts.log.Error("Error while parsing data",
				zap.Error(err),
				zap.String("imei", imei),
			)
			return
		}
		//	go func() {
		ts.LogPoints(points, buf)
		ts.PublishLastPoint(imei, points)
		if e := ts.avlDB.SaveAvlPoints(ctx, points); e != nil {
			ts.log.Error("failed to save avl points", zap.Error(e))
		}
		//}()
		ts.ResponseAcceptDataPack(conn, len(points))
	}
}

func (ts *TeltonikaServer) PublishLastPoint(imei string, points []*pb.AVLData) {
	subject := fmt.Sprintf("device.lastpoint.%s", imei)
	lastPointByte, err := proto.Marshal(points[len(points)-1])
	if err != nil {
		ts.log.Error("marshal last point failed", zap.Error(err))
		return
	}
	if e := ts.natsConn.Publish(subject, lastPointByte); e != nil {
		ts.log.Error("publish last point failed", zap.Error(e))
	}
}

func (ts *TeltonikaServer) LogPoints(points []*pb.AVLData, buff []byte) {
	for _, p := range points {
		ts.log.Info("new packet",
			zap.String("Priority", p.Priority.String()),
			zap.String("IMEI", p.GetImei()),
			//zap.Uint64("Timestamp", p.GetTimestamp()),
			zap.Any("Gps", p.GetGps()),
			zap.Any("IOElements", p.GetIoElements()),
			zap.Any("***************", buff),
		)
	}
}

func (ts *TeltonikaServer) ResponseAcceptIMEI(conn net.Conn) {
	_, err := conn.Write([]byte{1})
	if err != nil {
		ts.log.Error("response accept imei failed", zap.Error(err))
	}
}
func (ts *TeltonikaServer) ResponseAcceptDataPack(conn net.Conn, pointLen int) {
	_, err := conn.Write([]byte{0, 0, 0, uint8(pointLen)})
	if err != nil {
		ts.log.Error("response accept avl data failed", zap.Error(err))
	}
}

func (ts *TeltonikaServer) ResponseDecline(conn net.Conn) {
	_, err := conn.Write([]byte{0})
	if err != nil {
		ts.log.Error("response decline ailed", zap.Error(err))
	}
}
