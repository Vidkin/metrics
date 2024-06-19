package main

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/repository"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	serverConfig, err := config.NewServerConfig()
	if err != nil {
		return err
	}

	if err := logger.Initialize(serverConfig.LogLevel); err != nil {
		return err
	}

	memStorage := repository.New()
	metricRouter := handlers.NewMetricRouter(memStorage)

	logger.Log.Info("Running server", zap.String("address", serverConfig.ServerAddress.Address))
	return http.ListenAndServe(serverConfig.ServerAddress.Address, metricRouter.Router)
}
