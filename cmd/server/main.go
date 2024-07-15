package main

import (
	"github.com/Vidkin/metrics/app"
	"github.com/Vidkin/metrics/internal/config"
)

func main() {
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
