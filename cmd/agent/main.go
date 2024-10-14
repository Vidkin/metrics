package main

import (
	"fmt"
	"path"
	"runtime"

	"github.com/go-resty/resty/v2"

	"github.com/Vidkin/metrics/internal/agent"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/router"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	agentConfig, err := config.NewAgentConfig()
	if err != nil {
		panic(err)
	}
	if err = logger.Initialize(agentConfig.LogLevel); err != nil {
		panic(err)
	}
	memoryStorage := router.NewFileStorage("")
	memStats := &runtime.MemStats{}
	client := resty.New()
	if agentConfig.CryptoKey != "" {
		client.SetRootCertificate(path.Join(agentConfig.CryptoKey, "cert.pem"))
	}
	client.SetDoNotParseResponse(true)
	mw := agent.New(memoryStorage, memStats, client, agentConfig)

	mw.Poll()
}
