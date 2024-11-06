// Package app provides an implementation of a server app for handling metrics.
package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	protoAPI "github.com/Vidkin/metrics/internal/proto"
	"github.com/Vidkin/metrics/internal/repository/storage"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/pkg/interceptors"
	"github.com/Vidkin/metrics/proto"
)

type ServerApp struct {
	config     *config.ServerConfig
	httpSrv    *http.Server
	gRPCServer *grpc.Server
	repository router.Repository
}

func NewServerApp(cfg *config.ServerConfig) (*ServerApp, error) {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return nil, err
	}
	repo, err := router.NewRepository(cfg)
	if err != nil {
		return nil, err
	}

	serverApp := &ServerApp{
		config:     cfg,
		repository: repo,
	}

	if cfg.UseGRPC {
		var s *grpc.Server
		s = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				interceptors.LoggingInterceptor,
				interceptors.TrustedSubnetInterceptor(cfg.TrustedSubnet),
				interceptors.HashInterceptor(cfg.Key)))
		proto.RegisterMetricsServer(s, &protoAPI.MetricsServer{
			Repository:    repo,
			LastStoreTime: time.Now(),
			StoreInterval: (int)(cfg.StoreInterval),
			RetryCount:    cfg.RetryCount,
		})
		serverApp.gRPCServer = s
	} else {
		chiRouter := chi.NewRouter()
		metricRouter := router.NewMetricRouter(chiRouter, repo, cfg)
		serverApp.httpSrv = &http.Server{
			Addr:    cfg.ServerAddress.Address,
			Handler: metricRouter.Router,
		}
	}

	return serverApp, nil
}

func (a *ServerApp) Serve() {
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			logger.Log.Error("error start pprof endpoint", zap.Error(err))
		}
	}()
	if a.config.UseGRPC {
		listen, err := net.Listen("tcp", a.config.ServerAddress.Address)
		if err != nil {
			logger.Log.Fatal("listen gRPC server fatal error", zap.Error(err))
		}
		// получаем запрос gRPC
		if err := a.gRPCServer.Serve(listen); err != nil {
			logger.Log.Fatal("serve gRPC server fatal error", zap.Error(err))
		}
	} else {
		if a.config.CryptoKey != "" {
			if err := a.httpSrv.ListenAndServeTLS(path.Join(a.config.CryptoKey, "cert.pem"), path.Join(a.config.CryptoKey, "privateKey.pem")); err != nil && err != http.ErrServerClosed {
				logger.Log.Fatal("listen and serve tls fatal error", zap.Error(err))
			}
		} else {
			if err := a.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Log.Fatal("listen and serve fatal error", zap.Error(err))
			}
		}
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
			} else {
				return nil
			}
		}
	}
	return errors.New("provided Repository does not implement Dumper")
}

func (a *ServerApp) Run() {
	logger.Log.Info("running server", zap.String("address", a.config.ServerAddress.Address))

	go a.Serve()
	if a.config.StoreInterval > 0 {
		ticker := time.NewTicker(time.Duration(a.config.StoreInterval) * time.Second)
		go func() {
			for range ticker.C {
				if err := a.DumpToFile(); err != nil {
					logger.Log.Info("error interval dump", zap.Error(err))
				}
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	defer a.Stop()
}

func (a *ServerApp) Stop() {
	logger.Log.Info("stop server", zap.String("address", a.config.ServerAddress.Address))
	// Создаем контекст с таймаутом для корректного завершения сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем сервер, ожидая завершения текущих обработчиков
	if a.gRPCServer != nil {
		a.gRPCServer.GracefulStop()
	}
	if a.httpSrv != nil {
		if err := a.httpSrv.Shutdown(ctx); err != nil {
			logger.Log.Info("shutdown error", zap.Error(err))
		}
	}

	logger.Log.Info("dump metrics before exit")
	if _, ok := a.repository.(*storage.FileStorage); ok {
		if err := a.DumpToFile(); err != nil {
			logger.Log.Info("error dump metrics before exit", zap.Error(err))
		}
	}
	err := router.Close(a.repository)
	if err != nil {
		logger.Log.Info("error close repository before exit", zap.Error(err))
	}
}
