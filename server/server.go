package server

import (
	"net"
	"sync"

	"github.com/nats-io/nats.go"
	avldb "github.com/irisco88/teltonika-device/db/clickhouse"
	"go.uber.org/zap"
)

type Empty struct{}

type TeltonikaServer struct {
	listenAddr string
	ln         net.Listener
	quitChan   chan Empty
	wg         sync.WaitGroup
	log        *zap.Logger
	natsConn   *nats.Conn
	avlDB      avldb.AVLDBConn
}

const PRECISION = 10000000.0

type TcpServerInterface interface {
	Start()
	Stop()
	AcceptConnections()
	HandleConnection(conn net.Conn)
}

var (
	_ TcpServerInterface = &TeltonikaServer{}
)

func NewServer(listenAddr string,
	logger *zap.Logger,
	natsConn *nats.Conn,
	avlDB avldb.AVLDBConn) TcpServerInterface {
	return &TeltonikaServer{
		listenAddr: listenAddr,
		quitChan:   make(chan Empty),
		wg:         sync.WaitGroup{},
		log:        logger,
		natsConn:   natsConn,
		avlDB:      avlDB,
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

	go ts.AcceptConnections()
	ts.log.Info("server started",
		zap.String("ListenAddress", ts.listenAddr),
	)
	<-ts.quitChan
}

func (ts *TeltonikaServer) AcceptConnections() {
	for {
		conn, err := ts.ln.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return
			}
			ts.log.Error("failed to accept connection", zap.Error(err))
			continue
		}
		ts.log.Info("new Connection to the server", zap.String("Address", conn.RemoteAddr().String()))
		ts.wg.Add(1)
		go ts.HandleConnection(conn)
	}
}

func (ts *TeltonikaServer) Stop() {
	ts.log.Info("stopping server")
	// Close the listener to stop accepting new connections
	if ts.ln != nil {
		ts.ln.Close()
	}
	close(ts.quitChan)
	ts.wg.Wait()
}
