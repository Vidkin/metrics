package main

import (
	"runtime"

	"github.com/go-resty/resty/v2"

	"github.com/Vidkin/metrics/internal/agent"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/router"
)

func main() {
	agentConfig, err := config.NewAgentConfig()
	if err != nil {
		panic(err)
	}
	if err := logger.Initialize(agentConfig.LogLevel); err != nil {
		panic(err)
	}
	memoryStorage := router.NewFileStorage("")
	memStats := &runtime.MemStats{}
	client := resty.New()
	client.SetDoNotParseResponse(true)
	mw := agent.New(memoryStorage, memStats, client, agentConfig)

	mw.Poll()
}
