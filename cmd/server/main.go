package main

import (
	"github.com/Vidkin/metrics/internal/api/handler"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/repository"
	"go.uber.org/zap"
	"net/http"
	"time"
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

	memStorage := repository.NewMemoryStorage(serverConfig.FileStoragePath)
	defer memStorage.Save()

	if serverConfig.Restore {
		if err := memStorage.Load(); err != nil {
			logger.Log.Info("error load saved metrics", zap.Error(err))
		}
	}
	metricRouter := handler.NewMetricRouter(memStorage, serverConfig.StoreInterval)

	logger.Log.Info("Running server", zap.String("address", serverConfig.ServerAddress.Address))

	if serverConfig.StoreInterval > 0 {
		go func() {
			for {
				err = metricRouter.Repository.Save()
				if err != nil {
					logger.Log.Info("error saving metrics", zap.Error(err))
				}
				time.Sleep(time.Duration(serverConfig.StoreInterval) * time.Second)
			}
		}()
	}
	return http.ListenAndServe(serverConfig.ServerAddress.Address, metricRouter.Router)
}
