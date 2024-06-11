package main

import (
	"flag"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/caarlos0/env/v6"
	"github.com/go-resty/resty/v2"
	"runtime"
)

const (
	DefaultAgentPollInterval   = 2
	DefaultAgentReportInterval = 10

	DefaultServerAddress = "localhost"
	DefaultServerPort    = 8080
)

var ServerAddr = new(ServerAddress)

func main() {
	ServerAddr.Host = DefaultServerAddress
	ServerAddr.Port = DefaultServerPort

	flag.Var(ServerAddr, "a", "Server address host:port")
	flag.IntVar(&ServerAddr.ReportInterval, "r", DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar(&ServerAddr.PollInterval, "p", DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.Parse()

	env.Parse(ServerAddr)

	var memoryStorage = repository.New()
	memStats := &runtime.MemStats{}
	client := resty.New()

	Poll(client, memoryStorage, memStats)
}
