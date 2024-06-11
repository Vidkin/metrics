package main

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metricworker"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/go-resty/resty/v2"
	"runtime"
)

func main() {
	agentConfig := config.NewAgentConfig()

	var memoryStorage = repository.New()
	memStats := &runtime.MemStats{}
	client := resty.New()

	metricworker.Poll(client, memoryStorage, memStats, agentConfig)
}
