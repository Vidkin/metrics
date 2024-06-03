package main

import (
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"net/http"
)

func main() {
	var memStorage = storage.New()
	err := http.ListenAndServe(":8080", handlers.MetricsRouter(memStorage))

	if err != nil {
		panic(err)
	}
}
