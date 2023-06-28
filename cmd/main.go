package main

import (
	"fmt"
	"github.com/openfms/teltonika-device/simulator"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	avldb "github.com/openfms/teltonika-device/db/clickhouse"
	"github.com/openfms/teltonika-device/server"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	HostAddress     string
	PortNumber      uint
	NatsAddr        string
	AVLDBClickhouse string

	SimulatorHostAddr string
	TrackerIMEI       string
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("create new logger failed:%v\n", err)
	}
	randomIMEI := generateRandomIMEI()
	app := &cli.App{
		Name:  "teltonikasrv",
		Usage: "teltonika tcp server",
		Commands: []*cli.Command{
			{
				Name:  "server",
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
			{
				Name:  "simulator",
				Usage: "starts teltonika simulator",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "host",
						Usage:       "simulator host address",
						Destination: &SimulatorHostAddr,
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "imei",
						Usage:       "device imei",
						Value:       randomIMEI,
						DefaultText: randomIMEI,
						Destination: &TrackerIMEI,
						Required:    false,
					},
				},
				Action: func(ctx *cli.Context) error {
					teltonikaSimulator := simulator.NewTrackerDevice(SimulatorHostAddr, TrackerIMEI, log.Default())
					if e := teltonikaSimulator.Connect(); e != nil {
						return e
					}
					go teltonikaSimulator.SendRandomPoints()

					sigs := make(chan os.Signal, 1)
					signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
					<-sigs
					teltonikaSimulator.Stop()
					return nil
				},
			},
		},
	}

	if e := app.Run(os.Args); e != nil {
		logger.Error("failed to run app", zap.Error(e))
	}

}

func generateRandomIMEI() string {
	// Seed the random number generator
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Generate a random IMEI
	imei := "35"

	// Generate 13 random digits
	for i := 0; i < 13; i++ {
		digit := randomizer.Intn(10)
		imei += strconv.Itoa(digit)
	}

	return imei
}
