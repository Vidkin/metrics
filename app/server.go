package app

import (
	"context"
	"github.com/Vidkin/metrics/internal/api/handler"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/repository"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServerApp struct {
	config     *config.ServerConfig
	srv        *http.Server
	repository handler.Repository
}

func NewServerApp() (*ServerApp, error) {
	serverConfig, err := config.NewServerConfig()
	if err != nil {
		return nil, err
	}

	if err := logger.Initialize(serverConfig.LogLevel); err != nil {
		return nil, err
	}

	memStorage := repository.NewMemoryStorage(serverConfig.FileStoragePath)
	if serverConfig.Restore {
		if err := memStorage.Load(); err != nil {
			logger.Log.Info("error load saved metrics", zap.Error(err))
		}
	}
	metricRouter := handler.NewMetricRouter(memStorage, serverConfig.StoreInterval)
	srv := &http.Server{
		Addr:    serverConfig.ServerAddress.Address,
		Handler: metricRouter.Router,
	}
	return &ServerApp{
		config:     serverConfig,
		srv:        srv,
		repository: metricRouter.Repository,
	}, nil
}

func (app *ServerApp) Run() {
	logger.Log.Info("running server", zap.String("address", app.config.ServerAddress.Address))
	if app.config.StoreInterval > 0 {
		go func() {
			for {
				err := app.repository.Save()
				if err != nil {
					logger.Log.Info("error saving metrics", zap.Error(err))
				}
				time.Sleep(time.Duration(app.config.StoreInterval) * time.Second)
			}
		}()
	}
	go func() {
		if err := app.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("listen and serve fatal error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	defer app.Stop()
}

func (app *ServerApp) Stop() {
	logger.Log.Info("stop server", zap.String("address", app.config.ServerAddress.Address))
	// Создаем контекст с таймаутом для корректного завершения сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем сервер, ожидая завершения текущих обработчиков
	if err := app.srv.Shutdown(ctx); err != nil {
		logger.Log.Info("shutdown error", zap.Error(err))
	}

	logger.Log.Info("save metrics before exit")
	err := app.repository.Save()
	if err != nil {
		logger.Log.Info("error saving metrics", zap.Error(err))
	}
}
