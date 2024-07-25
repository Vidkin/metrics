package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

const (
	DefaultAgentPollInterval   = 2
	DefaultAgentReportInterval = 10
)

type AgentConfig struct {
	ServerAddress  *ServerAddress
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	LogLevel       string
}

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
	flag.IntVar(&config.RateLimit, "l", 0, "Rate limit")
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
