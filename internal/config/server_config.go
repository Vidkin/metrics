package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
)

// ServerConfig holds the configuration settings for the server.
//
// This struct contains various fields that define how the server operates,
// including its address, storage settings, logging preferences, and more.
// The fields can be populated from environment variables, allowing for
// flexible configuration without hardcoding values.
type ServerConfig struct {
	ServerAddress   *ServerAddress `json:"address"`
	LogLevel        string
	TrustedSubnet   string   `evn:"TRUSTED_SUBNET" json:"trusted_subnet"`
	ConfigPath      string   `env:"CONFIG"`
	FileStoragePath string   `env:"FILE_STORAGE_PATH" json:"store_file"`
	DatabaseDSN     string   `env:"DATABASE_DSN" json:"database_dsn"`
	Key             string   `env:"KEY" json:"hash_key"`
	CryptoKey       string   `env:"CRYPTO_KEY" json:"crypto_key"`
	StoreInterval   Interval `env:"STORE_INTERVAL" json:"store_interval"`
	Restore         bool     `env:"RESTORE" json:"restore"`
	RetryCount      int
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
	flag.StringVar(&config.ConfigPath, "c", "", "Path to json config file")
	flag.StringVar(&config.ConfigPath, "config", "", "Path to json config file")
	flag.StringVar(&config.LogLevel, "l", "info", "Log level")
	flag.IntVar((*int)(&config.StoreInterval), "i", 300, "Config store interval")
	flag.StringVar(&config.FileStoragePath, "f", "", "Metrics file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", "", "Database DSN")
	flag.StringVar(&config.Key, "k", "", "Hash key")
	flag.StringVar(&config.CryptoKey, "crypto-key", "", "Crypto key")
	flag.StringVar(&config.CryptoKey, "t", "", "Agent trusted subnet")
	flag.BoolVar(&config.Restore, "r", true, "Restore metrics on startup")
	flag.Parse()

	if config.ConfigPath != "" {
		if err := config.loadJSONConfig(config.ConfigPath); err != nil {
			logger.Log.Error("error parse json config file", zap.Error(err))
		}
	}

	err := env.Parse(config)
	if err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}

	return nil
}

func (config *ServerConfig) loadJSONConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var jsonServerConfig ServerConfig
	if err = json.Unmarshal(data, &jsonServerConfig); err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress = jsonServerConfig.ServerAddress
	}

	storeFilePassed := false
	dbDSNPassed := false
	cryptoKeyPassed := false
	storeIntervalPassed := false
	restorePassed := false
	hashKeyPassed := false
	trustedSubnetPassed := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--f", "-f":
			storeFilePassed = true
		case "--i", "-i":
			storeIntervalPassed = true
		case "--d", "-d":
			dbDSNPassed = true
		case "--crypto-key", "-crypto-key":
			cryptoKeyPassed = true
		case "--r", "-r":
			restorePassed = true
		case "--k", "-k":
			hashKeyPassed = true
		case "--t", "-t":
			trustedSubnetPassed = true
		}
	}

	if !trustedSubnetPassed {
		config.TrustedSubnet = jsonServerConfig.TrustedSubnet
	}

	if !dbDSNPassed {
		config.DatabaseDSN = jsonServerConfig.DatabaseDSN
	}

	if !storeFilePassed {
		config.FileStoragePath = jsonServerConfig.FileStoragePath
	}

	if !cryptoKeyPassed {
		config.CryptoKey = jsonServerConfig.CryptoKey
	}

	if !storeIntervalPassed {
		config.StoreInterval = jsonServerConfig.StoreInterval
	}

	if !restorePassed {
		config.Restore = jsonServerConfig.Restore
	}

	if !hashKeyPassed {
		config.Key = jsonServerConfig.Key
	}

	return nil
}
