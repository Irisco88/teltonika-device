package simulator

import (
	"fmt"
	"github.com/openfms/teltonika-device/parser"
	"net"
)

func (td *TrackerDevice) AuthenticateIMEI(clientConn net.Conn, imei string) error {
	imeiBytes, err := parser.EncodeIMEIToHex(imei)
	if err != nil {
		return fmt.Errorf("failed to encode IMEI: %v", err)
	}

	_, err = clientConn.Write(imeiBytes)
	if err != nil {
		return fmt.Errorf("failed to send authentication request: %v", err)
	}

	buf := make([]byte, 1)
	_, err = clientConn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read authentication response: %v", err)
	}

	if buf[0] != 1 {
		return fmt.Errorf("authentication not accepted")
	}

	return nil
}

func (td *TrackerDevice) SendPoints(clientConn net.Conn, points []*parser.AVLData) error {
	packetBytes, err := parser.MakeCodec8Packet(points)
	if err != nil {
		return fmt.Errorf("failed to create data packet: %v", err)
	}

	_, err = clientConn.Write(packetBytes)
	if err != nil {
		return fmt.Errorf("failed to send data packet: %v", err)
	}

	buf := make([]byte, 4)
	_, err = clientConn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	expectedLength := uint8(len(points))
	if buf[3] != expectedLength {
		return fmt.Errorf("sent points are not acceptable")
	}

	return nil
}
