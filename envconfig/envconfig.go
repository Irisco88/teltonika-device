package envconfig

import (
	"github.com/caarlos0/env/v6"
)

type DeviceServiceEnvConfig struct {
	PostgresDB   string `env:"POSTGRES_DATABASE_URL,notEmpty"`
	ClickHouseDB string `env:"CLICKHOUSE_DATABASE_URL,notEmpty"`
	NatsConn     string `env:"NATS"`
	Host         string `env:"HOST" envDefault:"0.0.0.0"`
	Port         string `env:"PORT" envDefault:"5000"`
}

func ReadDeviceServiceEnv() (*DeviceServiceEnvConfig, error) {
	cfg := &DeviceServiceEnvConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
