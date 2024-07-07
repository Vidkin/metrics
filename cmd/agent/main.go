package main

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric/worker"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/go-resty/resty/v2"
	"runtime"
)

func main() {
	agentConfig, err := config.NewAgentConfig()
	if err != nil {
		panic(err)
	}
	if err := logger.Initialize(agentConfig.LogLevel); err != nil {
		panic(err)
	}
	memoryStorage := repository.NewFileStorage("")
	memStats := &runtime.MemStats{}
	client := resty.New()
	client.SetDoNotParseResponse(true)
	mw := worker.New(memoryStorage, memStats, client, agentConfig)

	mw.Poll()
}
