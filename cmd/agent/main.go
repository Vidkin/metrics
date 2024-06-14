package main

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metricworker"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/go-resty/resty/v2"
	"runtime"
)

func main() {
	agentConfig, err := config.NewAgentConfig()
	if err != nil {
		panic(err)
	}
	memoryStorage := repository.New()
	memStats := &runtime.MemStats{}
	client := resty.New()
	mw := metricworker.New(memoryStorage, memStats, client, agentConfig)

	mw.Poll()
}
