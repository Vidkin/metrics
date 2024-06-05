package main

import (
	"flag"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/go-resty/resty/v2"
	"runtime"
)

var (
	ServerAddr     = new(ServerAddress)
	ReportInterval int
	PollInterval   int
)

func main() {
	ServerAddr.Host = internal.DefaultServerAddress
	ServerAddr.Port = internal.DefaultServerPort

	flag.Var(ServerAddr, "a", "Server address host:port")
	flag.IntVar(&ReportInterval, "r", internal.DefaultAgentReportInterval, "Agent report poll interval (sec)")
	flag.IntVar(&ReportInterval, "p", internal.DefaultAgentPollInterval, "Agent poll interval (sec)")
	flag.Parse()

	var memoryStorage = storage.New()
	memStats := &runtime.MemStats{}
	client := resty.New()

	Poll(client, memoryStorage, memStats)
}
