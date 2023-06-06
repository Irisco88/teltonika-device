package server

import (
	"fmt"
	"math/rand"
	"net"
)

func generateRandomHostPort() string {
	port := rand.Intn(65535-1024) + 1024
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}
