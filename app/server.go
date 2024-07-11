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

type ServerApp struct {
	config     *config.ServerConfig
	srv        *http.Server
	repository router.Repository
}

func NewServerApp(cfg *config.ServerConfig) (*ServerApp, error) {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return nil, err
	}
	repo, err := repository.NewRepository(cfg)
	if err != nil {
		return nil, err
	}
	chiRouter := chi.NewRouter()
	metricRouter := router.NewMetricRouter(chiRouter, repo, cfg)

	srv := &http.Server{
		Addr:    cfg.ServerAddress.Address,
		Handler: metricRouter.Router,
	}
	return &ServerApp{
		config:     cfg,
		srv:        srv,
		repository: repo,
	}, nil
}

func (a *ServerApp) Serve() {
	if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Log.Fatal("listen and serve fatal error", zap.Error(err))
	}
}

func (a *ServerApp) DumpToFile() error {
	if dumper, ok := a.repository.(router.Dumper); ok {
		for i := 0; i <= a.config.RetryCount; i++ {
			err := dumper.FullDump()
			if err != nil {
				var pathErr *os.PathError
				if errors.As(err, &pathErr) && i != a.config.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
				logger.Log.Info("error saving metrics", zap.Error(err))
			}
			break
		}
	}
	return errors.New("provided Repository does not implement Dumper")
}

func (a *ServerApp) IntervalDump() {
	if a.config.StoreInterval > 0 {
		for {
			if err := a.DumpToFile(); err != nil {
				logger.Log.Info("error interval dump", zap.Error(err))
			}
			time.Sleep(time.Duration(a.config.StoreInterval) * time.Second)
		}
	}
}

func (a *ServerApp) Run() {
	logger.Log.Info("running server", zap.String("address", a.config.ServerAddress.Address))

	go a.Serve()
	go a.IntervalDump()

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

	logger.Log.Info("dump metrics before exit")
	if _, ok := a.repository.(*repository.FileStorage); ok {
		if err := a.DumpToFile(); err != nil {
			logger.Log.Info("error dump metrics before exit", zap.Error(err))
		}
	}
	router.Close(a.repository)
}
