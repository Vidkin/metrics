package main

import (
	"flag"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/caarlos0/env/v6"
	"github.com/go-resty/resty/v2"
	"runtime"
)

var ServerAddr = new(ServerAddress)

func main() {
	ServerAddr.Host = internal.DefaultServerAddress
	ServerAddr.Port = internal.DefaultServerPort

	flag.Var(ServerAddr, "a", "Server address host:port")
	flag.IntVar(&ServerAddr.ReportInterval, "r", internal.DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar(&ServerAddr.PollInterval, "p", internal.DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.Parse()

	env.Parse(ServerAddr)

	var memoryStorage = storage.New()
	memStats := &runtime.MemStats{}
	client := resty.New()

	Poll(client, memoryStorage, memStats)
}
