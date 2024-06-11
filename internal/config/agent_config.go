package config

import (
	"flag"
	"github.com/Vidkin/metrics/internal/server_address"
	"github.com/caarlos0/env/v6"
)

const (
	DefaultAgentPollInterval   = 2
	DefaultAgentReportInterval = 10
)

type AgentConfig struct {
	ServerAddress  *server_address.ServerAddress
	ReportInterval int `env:"REPORT_INTERVAL"`
	PollInterval   int `env:"POLL_INTERVAL"`
}

func NewAgentConfig() *AgentConfig {
	var config AgentConfig
	config.ServerAddress = server_address.New()
	config.parseFlags()
	return &config
}

func (config *AgentConfig) parseFlags() {
	flag.Var(config.ServerAddress, "a", "Server address host:port")
	flag.IntVar(&config.ReportInterval, "r", DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar(&config.PollInterval, "p", DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.Parse()

	env.Parse(config.ServerAddress)

	if config.ServerAddress.Address == "" {
		config.ServerAddress.Address = config.ServerAddress.String()
	}
}
