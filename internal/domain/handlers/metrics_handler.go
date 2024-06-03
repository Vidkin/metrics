package handlers

import (
	"fmt"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

func MetricsRouter(repository repository.Repository) chi.Router {
	metricsRouter := chi.NewRouter()
	metricsRouter.Route("/", func(r chi.Router) {
		r.Get("/", RootHandler(repository))
		metricsRouter.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", GetMetricValueHandler(repository))
		})
		metricsRouter.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", UpdateMetricHandler(repository))
		})
	})
	return metricsRouter
}

func RootHandler(repository repository.Repository) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		for k, v := range repository.GetGauges() {
			io.WriteString(res, fmt.Sprintf("%s = %v\n", k, v))
		}
		for k, v := range repository.GetCounters() {
			io.WriteString(res, fmt.Sprintf("%s = %d\n", k, v))
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	}
}

func GetMetricValueHandler(repository repository.Repository) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, internal.ParamMetricType)
		metricName := chi.URLParam(req, internal.ParamMetricName)

		switch metricType {
		case internal.MetricTypeGauge:
			if metricValue, ok := repository.GetGauge(metricName); ok {
				valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
				res.Write([]byte(valueAsString))
			} else {
				http.Error(res, "Metric not found", http.StatusNotFound)
			}
		case internal.MetricTypeCounter:
			if metricValue, ok := repository.GetCounter(metricName); ok {
				valueAsString := strconv.FormatInt(metricValue, 10)
				res.Write([]byte(valueAsString))
			} else {
				http.Error(res, "Metric not found", http.StatusNotFound)
			}
		default:
			http.Error(res, "Bad metric type!", http.StatusBadRequest)
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricHandler(repository repository.Repository) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, internal.ParamMetricType)
		metricName := chi.URLParam(req, internal.ParamMetricName)
		metricValue := chi.URLParam(req, internal.ParamMetricValue)

		if metricName == "" {
			http.Error(res, "Empty metric name!", http.StatusNotFound)
		}

		switch metricType {
		case internal.MetricTypeGauge:
			if value, err := strconv.ParseFloat(metricValue, 64); err != nil {
				http.Error(res, "Bad metric value!", http.StatusBadRequest)
			} else {
				repository.UpdateGauge(metricName, value)
			}
		case internal.MetricTypeCounter:
			if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
				http.Error(res, "Bad metric value!", http.StatusBadRequest)
			} else {
				repository.UpdateCounter(metricName, value)
			}
		default:
			http.Error(res, "Bad metric type!", http.StatusBadRequest)
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	}
}
