package server

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/openfms/teltonika-device/server/parser"
	"gotest.tools/v3/assert"
)

func generateRandomHostPort() string {
	port := rand.Intn(65535-1024) + 1024
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}

// RunNatsServerOnPort will run a nats server on the given port.
func RunNatsServerOnPort(port int) *server.Server {
	opts := natstest.DefaultTestOptions
	opts.Port = port
	return RunNatsServerWithOptions(&opts)
}

// RunNatsServerWithOptions will run a server with the given options.
func RunNatsServerWithOptions(opts *server.Options) *server.Server {
	return natstest.RunServer(opts)
}

func NewNatsConnection(t *testing.T, url string) *nats.Conn {
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("Failed to create default connection: %v\n", err)
	}
	return nc
}

func ImeiAuthenticate(t *testing.T, clientConn net.Conn, imei string) {
	imeiBytes, err := parser.EncodeIMEIToHex(imei)
	assert.NilError(t, err)
	_, err = clientConn.Write(imeiBytes)
	assert.NilError(t, err)
	buf := make([]byte, 2048)
	_, err = clientConn.Read(buf)
	assert.NilError(t, err)
	assert.DeepEqual(t, buf[:1], []byte{1})
}

func SendPoints(t *testing.T, clientConn net.Conn, points []*parser.AVLData) {
	packetBytes, err := parser.MakeCodec8Packet(points)
	assert.NilError(t, err)
	_, err = clientConn.Write(packetBytes)
	assert.NilError(t, err)
	buf := make([]byte, 2048)
	_, err = clientConn.Read(buf)
	assert.NilError(t, err)
	assert.DeepEqual(t, buf[:4], []byte{0, 0, 0, uint8(len(points))})
}
