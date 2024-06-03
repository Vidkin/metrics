package main

import (
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func main() {
	metricsRouter := chi.NewRouter()
	var memStorage = storage.New()

	metricsRouter.Route("/", func(r chi.Router) {
		metricsRouter.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetMetricValueHandler(memStorage))
		})
		metricsRouter.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandler(memStorage))
		})
	})

	err := http.ListenAndServe(":8080", metricsRouter)

	if err != nil {
		panic(err)
	}
}
