package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
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
	ServerAddress  *ServerAddress
	Key            string `env:"KEY"`
	LogLevel       string
	ReportInterval int `env:"REPORT_INTERVAL"`
	PollInterval   int `env:"POLL_INTERVAL"`
	RateLimit      int `env:"RATE_LIMIT"`
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
	flag.IntVar(&config.ReportInterval, "r", DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar(&config.PollInterval, "p", DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.IntVar(&config.RateLimit, "l", 5, "Rate limit")
	flag.StringVar(&config.Key, "k", "", "Hash key")
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
