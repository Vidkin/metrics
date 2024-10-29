package main

import (
	"context"
	"fmt"
	"os/signal"
	"path"
	"runtime"
	"sync"
	"syscall"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Vidkin/metrics/internal/agent"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/proto"
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

	var mw *agent.MetricWorker
	if !agentConfig.UseGRPC {
		client := resty.New()
		if agentConfig.CryptoKey != "" {
			client.SetRootCertificate(path.Join(agentConfig.CryptoKey, "cert.pem"))
		}
		client.SetDoNotParseResponse(true)
		mw = agent.New(memoryStorage, memStats, client, nil, agentConfig)
	} else {
		conn, err := grpc.NewClient(agentConfig.ServerAddress.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Log.Fatal("error create grpc conn", zap.Error(err))
		}
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				logger.Log.Error("error close grpc conn", zap.Error(err))
			}
		}(conn)
		clientGRPC := proto.NewMetricsClient(conn)
		mw = agent.New(memoryStorage, memStats, nil, clientGRPC, agentConfig)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		mw.Poll(ctx)
	}()

	wg.Wait()
}
