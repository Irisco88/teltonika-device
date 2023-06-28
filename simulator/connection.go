package simulator

import (
	"fmt"
	"github.com/openfms/teltonika-device/parser"
	"golang.org/x/exp/slices"
	"net"
)

func (td *TrackerDevice) AuthenticateIMEI(clientConn net.Conn, imei string) error {
	imeiBytes, err := parser.EncodeIMEIToHex(imei)
	if err != nil {
		return fmt.Errorf("failed to encode imei: %v", err.Error())
	}
	_, err = clientConn.Write(imeiBytes)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err.Error())
	}
	buf := make([]byte, 2048)
	_, err = clientConn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read authenticate response: %v", err.Error())
	}
	if !slices.Equal(buf[:1], []byte{1}) {
		return fmt.Errorf("auth not accepted")
	}
	return nil
}

func (td *TrackerDevice) SendPoints(clientConn net.Conn, points []*parser.AVLData) error {
	packetBytes, err := parser.MakeCodec8Packet(points)
	if err != nil {
		return fmt.Errorf("failed to make datapacket: %v", err.Error())
	}
	_, err = clientConn.Write(packetBytes)
	if err != nil {
		return fmt.Errorf("failed to send: %v", err.Error())
	}
	buf := make([]byte, 2048)
	_, err = clientConn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read send point response: %v", err.Error())
	}
	if !slices.Equal(buf[:4], []byte{0, 0, 0, uint8(len(points))}) {
		return fmt.Errorf("sent points are not acceptable")
	}
	return nil
}
