package server

import (
	"github.com/packetify/teltonika-device/proto/pb"
	"github.com/packetify/teltonika-device/server/parser"
	"go.uber.org/zap"
	"net"
	"sync"
)

type Empty struct{}

type TeltonikaServer struct {
	listenAddr string
	ln         net.Listener
	quitChan   chan Empty
	wg         sync.WaitGroup
	log        *zap.Logger
}

const PRECISION = 10000000.0

type TcpServerInterface interface {
	Start()
	Stop()
}

var (
	_ TcpServerInterface = &TeltonikaServer{}
)

func NewServer(listenAddr string, logger *zap.Logger) TcpServerInterface {
	return &TeltonikaServer{
		listenAddr: listenAddr,
		quitChan:   make(chan Empty),
		wg:         sync.WaitGroup{},
		log:        logger,
	}
}

func (ts *TeltonikaServer) Start() {
	ln, err := net.Listen("tcp", ts.listenAddr)
	if err != nil {
		ts.log.Error("failed to listen", zap.Error(err))
		return
	}
	defer ln.Close()
	ts.ln = ln

	go ts.acceptConnections()
	ts.log.Info("server started",
		zap.String("ListenAddress", ts.listenAddr),
	)
	<-ts.quitChan
}

func (ts *TeltonikaServer) acceptConnections() {
	for {
		conn, err := ts.ln.Accept()
		if err != nil {
			ts.log.Error("accept connection error", zap.Error(err))
			continue
		}
		ts.log.Info("new Connection to the server", zap.String("Address", conn.RemoteAddr().String()))
		ts.wg.Add(1)
		go ts.HandleConnection(conn)
	}
}

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
			ts.log.Error("read failed", zap.Error(err))
			break
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
			ts.ResponseAcceptMsg(conn)
			authenticated = true
			continue
		}
		points, err := parser.ParsePacket(buf, imei)
		if err != nil {
			ts.log.Error("Error while parsing data",
				zap.Error(err),
				zap.String("imei", imei),
			)
			break
		}
		go ts.LogPoints(points)
		conn.Write([]byte{0, 0, 0, uint8(len(points))})
	}
}

func (ts *TeltonikaServer) LogPoints(points []*pb.AVLData) {
	for _, p := range points {
		ts.log.Info("new packet",
			zap.String("IMEI", p.GetImei()),
			zap.Uint64("timestamp", p.GetTimestamp()),
			zap.Any("gps", p.GetGps()),
			zap.Any("IOElements", p.GetIoElements()),
		)
	}
}

func (ts *TeltonikaServer) ResponseAcceptMsg(conn net.Conn) {
	conn.Write([]byte{1})
}

func (ts *TeltonikaServer) ResponseDecline(conn net.Conn) {
	conn.Write([]byte{0})
}

func (ts *TeltonikaServer) Stop() {
	ts.wg.Wait()
	ts.quitChan <- Empty{}
	ts.log.Info("stop server")
}
