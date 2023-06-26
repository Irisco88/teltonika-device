package clickhouse

import (
	"context"
	"net"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	pb "github.com/openfms/protos/gen/go/device/v1"
)

//go:generate mockgen -source=$GOFILE -destination=mock_db/conn.go -package=$GOPACKAG
type AVLDBConn interface {
	GetConn() driver.Conn
	SaveAvlPoints(ctx context.Context, points []*pb.AVLData) error
}

var _ AVLDBConn = &AVLDataBase{}

type AVLDataBase struct {
	ClickhouseConn driver.Conn
}

func (adb *AVLDataBase) GetConn() driver.Conn {
	return adb.ClickhouseConn
}

func ConnectAvlDB(databaseURL string) (*AVLDataBase, error) {
	opts, err := clickhouse.ParseDSN(databaseURL)
	if err != nil {
		return nil, err
	}
	opts.DialContext = func(ctx context.Context, addr string) (net.Conn, error) {
		//dialCount++
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	}
	opts.Compression = &clickhouse.Compression{
		Method: clickhouse.CompressionLZ4,
	}
	opts.DialTimeout = time.Second * 30
	opts.MaxOpenConns = 5
	opts.MaxIdleConns = 5
	opts.ConnMaxLifetime = time.Duration(10) * time.Minute
	opts.ConnOpenStrategy = clickhouse.ConnOpenInOrder

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	if e := conn.Ping(context.Background()); e != nil {
		return nil, e
	}
	return &AVLDataBase{
		ClickhouseConn: conn,
	}, nil
}
