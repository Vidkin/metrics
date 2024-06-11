package handlers

import (
	"fmt"
	"github.com/Vidkin/metrics/internal"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

type MetricRouter struct {
	Repository Repository
	Router     chi.Router
}

type Repository interface {
	UpdateGauge(key string, value float64)
	UpdateCounter(key string, value int64)

	GetGauges() map[string]float64
	GetCounters() map[string]int64

	GetGauge(metricName string) (float64, bool)
	GetCounter(metricName string) (int64, bool)
}

func NewMetricRouter(repository Repository) *MetricRouter {
	var mr MetricRouter
	router := chi.NewRouter()
	router.Route("/", func(r chi.Router) {
		r.Get("/", mr.RootHandler())
		router.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", mr.GetMetricValueHandler())
		})
		router.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", mr.UpdateMetricHandler())
		})
	})
	mr.Router = router
	mr.Repository = repository
	return &mr
}

func (mr *MetricRouter) RootHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		for k, v := range mr.Repository.GetGauges() {
			io.WriteString(res, fmt.Sprintf("%s = %v\n", k, v))
		}
		for k, v := range mr.Repository.GetCounters() {
			io.WriteString(res, fmt.Sprintf("%s = %d\n", k, v))
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	}
}

func (mr *MetricRouter) GetMetricValueHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, internal.ParamMetricType)
		metricName := chi.URLParam(req, internal.ParamMetricName)

		switch metricType {
		case internal.MetricTypeGauge:
			if metricValue, ok := mr.Repository.GetGauge(metricName); ok {
				valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
				res.Write([]byte(valueAsString))
			} else {
				http.Error(res, "Metric not found", http.StatusNotFound)
			}
		case internal.MetricTypeCounter:
			if metricValue, ok := mr.Repository.GetCounter(metricName); ok {
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

func (mr *MetricRouter) UpdateMetricHandler() http.HandlerFunc {
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
				mr.Repository.UpdateGauge(metricName, value)
			}
		case internal.MetricTypeCounter:
			if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
				http.Error(res, "Bad metric value!", http.StatusBadRequest)
			} else {
				mr.Repository.UpdateCounter(metricName, value)
			}
		default:
			http.Error(res, "Bad metric type!", http.StatusBadRequest)
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	}
}
