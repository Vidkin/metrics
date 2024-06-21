package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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
		r.Get("/", logger.LoggingHandler(mr.RootHandler))
		router.Route("/value", func(r chi.Router) {
			r.Post("/", logger.LoggingHandler(mr.GetMetricValueHandlerJSON))
			r.Get("/{metricType}/{metricName}", logger.LoggingHandler(mr.GetMetricValueHandler))
		})
		router.Route("/update", func(r chi.Router) {
			r.Post("/", logger.LoggingHandler(mr.UpdateMetricHandlerJSON))
			r.Post("/{metricType}/{metricName}/{metricValue}", logger.LoggingHandler(mr.UpdateMetricHandler))
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
				logger.Log.Info("can't convert metric value", zap.Error(err))
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
				logger.Log.Info("can't convert metric value", zap.Error(err))
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
			logger.Log.Info("can't convert metric value", zap.Error(err))
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			mr.Repository.UpdateGauge(metricName, value)
		}
	case MetricTypeCounter:
		if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			logger.Log.Info("can't convert metric value", zap.Error(err))
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

func (mr *MetricRouter) UpdateMetricHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var metric models.Metrics
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	if metric.MType == MetricTypeGauge {
		if metric.Value == nil {
			http.Error(res, "empty metric value", http.StatusBadRequest)
			return
		}
		mr.Repository.UpdateGauge(metric.ID, *metric.Value)
	} else if metric.MType == MetricTypeCounter {
		if metric.Delta == nil {
			http.Error(res, "empty metric delta", http.StatusBadRequest)
			return
		}
		mr.Repository.UpdateCounter(metric.ID, *metric.Delta)
	} else {
		http.Error(res, "bad metric type", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(res)
	if err := enc.Encode(metric); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		http.Error(res, "error encoding response", http.StatusInternalServerError)
		return
	}
}

func (mr *MetricRouter) GetMetricValueHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var metric models.Metrics

	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	respMetric := models.Metrics{
		ID:    metric.ID,
		MType: metric.MType,
	}

	if metric.MType == MetricTypeCounter {
		if v, ok := mr.Repository.GetCounter(metric.ID); !ok {
			http.Error(res, "metric not found", http.StatusNotFound)
			return
		} else {
			respMetric.Delta = &v
		}
	} else if metric.MType == MetricTypeGauge {
		if v, ok := mr.Repository.GetGauge(metric.ID); !ok {
			http.Error(res, "metric not found", http.StatusNotFound)
			return
		} else {
			respMetric.Value = &v
		}
	} else {
		http.Error(res, "bad metric type", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res)
	if err := enc.Encode(respMetric); err != nil {
		logger.Log.Info("error encoding response metric", zap.Error(err))
		http.Error(res, "error encoding response metric", http.StatusInternalServerError)
		return
	}
}
