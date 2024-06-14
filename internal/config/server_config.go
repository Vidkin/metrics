package config

import (
	"flag"
	"github.com/Vidkin/metrics/internal/serveraddress"
	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	ServerAddress *serveraddress.ServerAddress
}

func NewServerConfig() *ServerConfig {
	var config ServerConfig
	config.ServerAddress = serveraddress.New()
	config.parseFlags()
	return &config
}

func (config *ServerConfig) parseFlags() {
	flag.Var(config.ServerAddress, "a", "Net address host:port")
	flag.Parse()

	env.Parse(config.ServerAddress)

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}
}
