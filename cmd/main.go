package main

import (
	"fmt"
	"github.com/packetify/teltonika-device/server"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	HostAddress string
	PortNumber  uint
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("create new logger failed:%v\n", err)
	}

	app := &cli.App{
		Name:  "server",
		Usage: "teltonika tcp server",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "starts server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "host",
						Usage:       "host address",
						Value:       "0.0.0.0",
						DefaultText: "0.0.0.0",
						Destination: &HostAddress,
						EnvVars:     []string{"HOST"},
					},
					&cli.UintFlag{
						Name:        "port",
						Usage:       "server port number",
						Value:       5000,
						DefaultText: "50000",
						Aliases:     []string{"p"},
						Destination: &PortNumber,
						EnvVars:     []string{"PORT"},
					},
				},
				Action: func(ctx *cli.Context) error {
					listenAddr := net.JoinHostPort(HostAddress, fmt.Sprintf("%d", PortNumber))

					s := server.NewServer(listenAddr, logger)
					go s.Start()

					sigs := make(chan os.Signal, 1)
					signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
					<-sigs
					s.Stop()
					return nil
				},
			},
		},
	}

	if e := app.Run(os.Args); e != nil {
		logger.Error("failed to run app", zap.Error(e))
	}

}
