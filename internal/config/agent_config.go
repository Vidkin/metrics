package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
)

// Constants for default agent intervals.
//
// DefaultAgentPollInterval specifies the default interval at which the agent
// polls for updates. The default value is set to 2 seconds.
//
// DefaultAgentReportInterval specifies the default interval at which the agent
// reports metrics to the server. The default value is set to 10 seconds.
const (
	DefaultAgentPollInterval   = 2
	DefaultAgentReportInterval = 10
)

// AgentConfig holds the configuration settings for the agent.
//
// This struct is used to manage various parameters that control the behavior
// of the agent, including server connection details, polling intervals, rate
// limits, and logging levels. The configuration can be populated from command-line
// flags and environment variables, allowing for flexible deployment and management
// of the agent's settings.
type AgentConfig struct {
	ServerAddress  *ServerAddress `json:"address"`
	ConfigPath     string         `env:"CONFIG"`
	Key            string         `env:"KEY" json:"hash_key"`
	CryptoKey      string         `env:"CRYPTO_KEY" json:"crypto_key"`
	LogLevel       string
	ReportInterval Interval `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval   Interval `env:"POLL_INTERVAL" json:"poll_interval"`
	UseGRPC        bool     `env:"USE_GRPC" json:"use_grpc"`
	RateLimit      int      `env:"RATE_LIMIT" json:"rate_limit"`
}

// NewAgentConfig initializes a new AgentConfig instance with default values
// and parses command-line flags and environment variables to populate its fields.
//
// Returns:
// - A pointer to an `AgentConfig` instance containing the configuration settings.
// - An error if there was an issue during initialization or parsing; otherwise, nil.
func NewAgentConfig() (*AgentConfig, error) {
	var config AgentConfig
	config.ServerAddress = NewServerAddress()
	config.LogLevel = "info"
	err := config.parseFlags()
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *AgentConfig) parseFlags() error {
	flag.Var(config.ServerAddress, "a", "Server address host:port")
	flag.StringVar(&config.ConfigPath, "c", "", "Path to json config file")
	flag.StringVar(&config.ConfigPath, "config", "", "Path to json config file")
	flag.IntVar((*int)(&config.ReportInterval), "r", DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar((*int)(&config.PollInterval), "p", DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.IntVar(&config.RateLimit, "l", 5, "Rate limit")
	flag.StringVar(&config.Key, "k", "", "Hash key")
	flag.BoolVar(&config.UseGRPC, "g", true, "Use gRPC")
	flag.StringVar(&config.CryptoKey, "crypto-key", "", "Crypto key")
	flag.Parse()

	if config.ConfigPath != "" {
		if err := config.loadJSONConfig(config.ConfigPath); err != nil {
			logger.Log.Error("error parse json config file", zap.Error(err))
		}
	}

	err := env.Parse(config.ServerAddress)
	if err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}

	return nil
}

func (config *AgentConfig) loadJSONConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var jsonAgentConfig AgentConfig
	if err = json.Unmarshal(data, &jsonAgentConfig); err != nil {
		return err
	}

	if config.ServerAddress.Address == "" {
		config.ServerAddress = jsonAgentConfig.ServerAddress
	}

	cryptoKeyPassed := false
	reportIntervalPassed := false
	pollIntervalPassed := false
	hashKeyPassed := false
	rateLimitPassed := false
	useGRPCPassed := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--p", "-p":
			pollIntervalPassed = true
		case "--crypto-key", "-crypto-key":
			cryptoKeyPassed = true
		case "--r", "-r":
			reportIntervalPassed = true
		case "--l", "-l":
			rateLimitPassed = true
		case "--k", "-k":
			hashKeyPassed = true
		case "--g", "-g":
			useGRPCPassed = true
		}
	}

	if !cryptoKeyPassed {
		config.CryptoKey = jsonAgentConfig.CryptoKey
	}

	if !useGRPCPassed {
		config.UseGRPC = jsonAgentConfig.UseGRPC
	}

	if !reportIntervalPassed {
		config.ReportInterval = jsonAgentConfig.ReportInterval
	}

	if !pollIntervalPassed {
		config.PollInterval = jsonAgentConfig.PollInterval
	}

	if !rateLimitPassed {
		config.RateLimit = jsonAgentConfig.RateLimit
	}

	if !hashKeyPassed {
		config.Key = jsonAgentConfig.Key
	}

	return nil
}
