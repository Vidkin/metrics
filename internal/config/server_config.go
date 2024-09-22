package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// ServerConfig holds the configuration settings for the server.
//
// This struct contains various fields that define how the server operates,
// including its address, storage settings, logging preferences, and more.
// The fields can be populated from environment variables, allowing for
// flexible configuration without hardcoding values.
type ServerConfig struct {
	ServerAddress   *ServerAddress
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	RetryCount      int
	LogLevel        string
}

// NewServerConfig initializes a new ServerConfig instance with default values
// and parses command-line flags and environment variables to populate its fields
//
// Returns:
// - A pointer to the newly created and initialized ServerConfig instance.
// - An error if the configuration parsing fails; otherwise, nil.
func NewServerConfig() (*ServerConfig, error) {
	var config ServerConfig
	config.ServerAddress = NewServerAddress()
	config.RetryCount = 3
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
	flag.StringVar(&config.FileStoragePath, "f", "", "Metrics file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", "", "Database DSN")
	flag.StringVar(&config.Key, "k", "", "Hash key")
	flag.BoolVar(&config.Restore, "r", true, "Restore metrics on startup")
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
