package simulator

import (
	"fmt"
	"github.com/openfms/teltonika-device/parser"
	"log"
	"net"
	"os"
	"sync"
)

type TrackerDevice struct {
	serverAddr, imei string
	conn             net.Conn
	wg               sync.WaitGroup
	log              *log.Logger
}

type TrackerInterface interface {
	Connect() error
	Stop()
	AuthenticateIMEI(clientConn net.Conn, imei string) error
	SendPoints(clientConn net.Conn, points []*parser.AVLData) error
	SendRandomPoints()
}

var (
	_ TrackerInterface = &TrackerDevice{}
)

func NewTrackerDevice(serverAddr, imei string, logger *log.Logger) *TrackerDevice {
	return &TrackerDevice{
		serverAddr: serverAddr,
		imei:       imei,
		wg:         sync.WaitGroup{},
		log:        logger,
	}
}

func (td *TrackerDevice) Connect() error {
	conn, err := net.Dial("tcp", td.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to dial server: %v\n", err.Error())
	}
	td.conn = conn
	return nil
}

func (td *TrackerDevice) Stop() {
	td.wg.Wait()
	td.conn.Close()
	td.log.Println("stop tracker simulator...")
	os.Exit(0)
}
