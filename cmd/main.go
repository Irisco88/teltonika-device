package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	avldb "github.com/packetify/teltonika-device/db/clickhouse"
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
	HostAddress     string
	PortNumber      uint
	NatsAddr        string
	AVLDBClickhouse string
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
						DefaultText: "5000",
						Aliases:     []string{"p"},
						Destination: &PortNumber,
						EnvVars:     []string{"PORT"},
					},
					&cli.StringFlag{
						Name:        "nats",
						Usage:       "nats Address",
						Value:       "127.0.0.1:4222",
						DefaultText: "127.0.0.1:4222",
						Destination: &NatsAddr,
						EnvVars:     []string{"NATS"},
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "avldb",
						Usage:       "avldb clickhouse url",
						Value:       "clickhouse://admin:password@127.0.0.1:9423/default?dial_timeout=200ms",
						DefaultText: "clickhouse://admin:password@127.0.0.1:9423/default?dial_timeout=200ms",
						Destination: &AVLDBClickhouse,
						EnvVars:     []string{"AVLDB_CLICKHOUSE"},
						Required:    true,
					},
				},
				Action: func(ctx *cli.Context) error {
					listenAddr := net.JoinHostPort(HostAddress, fmt.Sprintf("%d", PortNumber))
					natsCon, err := nats.Connect(NatsAddr)
					if err != nil {
						return err
					}
					avlClickhouseDB, err := avldb.ConnectAvlDB(AVLDBClickhouse)
					if err != nil {
						return err
					}

					s := server.NewServer(listenAddr, logger, natsCon, avlClickhouseDB)
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
