package config

import (
	"flag"
	"github.com/Vidkin/metrics/internal/serveraddress"
	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	ServerAddress   *serveraddress.ServerAddress
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	LogLevel        string
}

func NewServerConfig() (*ServerConfig, error) {
	var config ServerConfig
	config.ServerAddress = serveraddress.New()
	err := config.parseFlags()
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *ServerConfig) parseFlags() error {
	flag.Var(config.ServerAddress, "a", "Net address host:port")
	flag.StringVar(&config.LogLevel, "l", "info", "Log level")
	flag.IntVar(&config.StoreInterval, "i", 300, "Config store interval")
	flag.StringVar(&config.FileStoragePath, "f", "/tmp/metrics-db.json", "Config file storage path")
	flag.BoolVar(&config.Restore, "r", true, "Restore config on startup")
	flag.Parse()

	err := env.Parse(config)
	if err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}

	return nil
}
