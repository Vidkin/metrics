package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
)

const (
	ParamMetricType  = "metricType"
	ParamMetricName  = "metricName"
	ParamMetricValue = "metricValue"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
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
		r.Get("/", mr.RootHandler)
		router.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", mr.GetMetricValueHandler)
		})
		router.Route("/update", func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", mr.UpdateMetricHandler)
		})
	})
	mr.Router = router
	mr.Repository = repository
	return &mr
}

func (mr *MetricRouter) RootHandler(res http.ResponseWriter, _ *http.Request) {
	for k, v := range mr.Repository.GetGauges() {
		_, err := io.WriteString(res, fmt.Sprintf("%s = %v\n", k, v))
		if err != nil {
			continue
		}
	}
	for k, v := range mr.Repository.GetCounters() {
		_, err := io.WriteString(res, fmt.Sprintf("%s = %d\n", k, v))
		if err != nil {
			continue
		}
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (mr *MetricRouter) GetMetricValueHandler(res http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, ParamMetricType)
	metricName := chi.URLParam(req, ParamMetricName)

	switch metricType {
	case MetricTypeGauge:
		if metricValue, ok := mr.Repository.GetGauge(metricName); ok {
			valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
			_, err := res.Write([]byte(valueAsString))
			if err != nil {
				http.Error(res, "Can't convert metric value", http.StatusInternalServerError)
			}
		} else {
			http.Error(res, "Metric not found", http.StatusNotFound)
		}
	case MetricTypeCounter:
		if metricValue, ok := mr.Repository.GetCounter(metricName); ok {
			valueAsString := strconv.FormatInt(metricValue, 10)
			_, err := res.Write([]byte(valueAsString))
			if err != nil {
				http.Error(res, "Can't convert metric value", http.StatusInternalServerError)
			}
		} else {
			http.Error(res, "Metric not found", http.StatusNotFound)
		}
	default:
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (mr *MetricRouter) UpdateMetricHandler(res http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, ParamMetricType)
	metricName := chi.URLParam(req, ParamMetricName)
	metricValue := chi.URLParam(req, ParamMetricValue)

	if metricName == "" {
		http.Error(res, "Empty metric name!", http.StatusNotFound)
	}

	switch metricType {
	case MetricTypeGauge:
		if value, err := strconv.ParseFloat(metricValue, 64); err != nil {
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			mr.Repository.UpdateGauge(metricName, value)
		}
	case MetricTypeCounter:
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
