package main

import (
	"fmt"

	"github.com/Vidkin/metrics/app"
	"github.com/Vidkin/metrics/internal/config"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	cfg, err := config.NewServerConfig()
	if err != nil {
		panic(err)
	}

	serverApp, err := app.NewServerApp(cfg)
	if err != nil {
		panic(err)
	}
	serverApp.Run()
}
