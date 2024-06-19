package config

import (
	"flag"
	"github.com/Vidkin/metrics/internal/serveraddress"
	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	ServerAddress *serveraddress.ServerAddress
	LogLevel      string
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
	flag.Parse()

	err := env.Parse(config.ServerAddress)
	if err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}

	return nil
}
