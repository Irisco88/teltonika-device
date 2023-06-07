package server

import (
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"math/rand"
	"net"
	"testing"
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
