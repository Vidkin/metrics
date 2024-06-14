package main

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/repository"
	"net/http"
)

func main() {
	serverConfig := config.NewServerConfig()
	memStorage := repository.New()
	metricRouter := handlers.NewMetricRouter(memStorage)

	err := http.ListenAndServe(serverConfig.ServerAddress.Address, metricRouter.Router)

	if err != nil {
		panic(err)
	}
}
