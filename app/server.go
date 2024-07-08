package app

import (
	"context"
	"errors"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	RetryCount = 3
)

type ServerApp struct {
	config     *config.ServerConfig
	srv        *http.Server
	repository router.Repository
}

func initRepository(serverConfig *config.ServerConfig) (router.Repository, error) {
	if serverConfig.DatabaseDSN != "" {
		return repository.NewPostgresStorage(serverConfig.DatabaseDSN)
	}

	if serverConfig.FileStoragePath != "" {
		fileStorage := repository.NewFileStorage(serverConfig.FileStoragePath)
		if serverConfig.Restore {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			for i := 0; i <= RetryCount; i++ {
				err := fileStorage.Load(ctx)
				if err != nil {
					var pathErr *os.PathError
					if errors.As(err, &pathErr) && i != RetryCount {
						logger.Log.Info("repository connection error", zap.Error(err))
						time.Sleep(time.Duration(1+i*2) * time.Second)
						continue
					}
					logger.Log.Info("error load saved metrics", zap.Error(err))
				}
				break
			}
		}
		return fileStorage, nil
	}

	memStorage := repository.NewMemoryStorage()
	return memStorage, nil
}

func NewServerApp() (*ServerApp, error) {
	serverConfig, err := config.NewServerConfig()
	if err != nil {
		return nil, err
	}

	if err := logger.Initialize(serverConfig.LogLevel); err != nil {
		return nil, err
	}
	repo, err := initRepository(serverConfig)
	if err != nil {
		return nil, err
	}
	chiRouter := chi.NewRouter()
	metricRouter := router.NewMetricRouter(chiRouter, repo, serverConfig)

	srv := &http.Server{
		Addr:    serverConfig.ServerAddress.Address,
		Handler: metricRouter.Router,
	}
	return &ServerApp{
		config:     serverConfig,
		srv:        srv,
		repository: repo,
	}, nil
}

func (a *ServerApp) Run() {
	logger.Log.Info("running server", zap.String("address", a.config.ServerAddress.Address))

	if t, ok := a.repository.(*repository.FileStorage); ok && a.config.StoreInterval > 0 {
		go func() {
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				for i := 0; i <= RetryCount; i++ {
					err := t.Save(ctx)
					if err != nil {
						var pathErr *os.PathError
						if errors.As(err, &pathErr) && i != RetryCount {
							logger.Log.Info("repository connection error", zap.Error(err))
							time.Sleep(time.Duration(1+i*2) * time.Second)
							continue
						}
						logger.Log.Info("error saving metrics", zap.Error(err))
					}
					break
				}
				time.Sleep(time.Duration(a.config.StoreInterval) * time.Second)
				cancel()
			}
		}()
	}
	go func() {
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("listen and serve fatal error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	defer a.Stop()
}

func (a *ServerApp) Stop() {
	logger.Log.Info("stop server", zap.String("address", a.config.ServerAddress.Address))
	// Создаем контекст с таймаутом для корректного завершения сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем сервер, ожидая завершения текущих обработчиков
	if err := a.srv.Shutdown(ctx); err != nil {
		logger.Log.Info("shutdown error", zap.Error(err))
	}

	logger.Log.Info("save metrics before exit")
	if t, ok := a.repository.(*repository.FileStorage); ok {
		for i := 0; i <= RetryCount; i++ {
			err := t.Save(ctx)
			if err != nil {
				var pathErr *os.PathError
				if errors.As(err, &pathErr) && i != RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
				logger.Log.Info("error saving metrics", zap.Error(err))
			}
			break
		}
	}
	a.repository.Close()
}
