package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Vidkin/metrics/internal/domain/middleware"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	ParamMetricType  = "metricType"
	ParamMetricName  = "metricName"
	ParamMetricValue = "metricValue"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

type MetricRouter struct {
	Repository    Repository
	Router        chi.Router
	LastStoreTime time.Time
	StoreInterval int
}

type Repository interface {
	UpdateGauge(key string, value float64)
	UpdateCounter(key string, value int64)

	GetGauges() map[string]float64
	GetCounters() map[string]int64

	GetGauge(metricName string) (float64, bool)
	GetCounter(metricName string) (int64, bool)

	Save() error
	Load() error
	SaveGauge(metricName string, metricValue float64) error
	SaveCounter(metricName string, metricValue int64) error
}

func NewMetricRouter(repository Repository, storeInterval int) *MetricRouter {
	var mr MetricRouter
	router := chi.NewRouter()

	router.Use(middleware.Logging)
	router.Use(middleware.Gzip)

	router.Route("/", func(r chi.Router) {
		r.Get("/", mr.RootHandler)
		router.Route("/value", func(r chi.Router) {
			r.Post("/", mr.GetMetricValueHandlerJSON)
			r.Get("/{metricType}/{metricName}", mr.GetMetricValueHandler)
		})
		router.Route("/update", func(r chi.Router) {
			r.Post("/", mr.UpdateMetricHandlerJSON)
			r.Post("/{metricType}/{metricName}/{metricValue}", mr.UpdateMetricHandler)
		})
	})
	mr.Router = router
	mr.Repository = repository
	mr.StoreInterval = storeInterval
	mr.LastStoreTime = time.Now()
	return &mr
}

func (mr *MetricRouter) RootHandler(res http.ResponseWriter, _ *http.Request) {
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)

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
			return
		} else {
			mr.Repository.UpdateGauge(metricName, value)
			if mr.StoreInterval == 0 {
				if err := mr.Repository.SaveGauge(metricName, value); err != nil {
					logger.Log.Info("error saving gauge metric", zap.Error(err))
					http.Error(res, "error saving gauge metric", http.StatusInternalServerError)
					return
				}
			}
		}
	case MetricTypeCounter:
		if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			logger.Log.Info("can't convert metric value", zap.Error(err))
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
			return
		} else {
			mr.Repository.UpdateCounter(metricName, value)
			if mr.StoreInterval == 0 {
				if err := mr.Repository.SaveCounter(metricName, value); err != nil {
					logger.Log.Info("error saving counter metric", zap.Error(err))
					http.Error(res, "error saving counter metric", http.StatusInternalServerError)
					return
				}
			}
		}
	default:
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
		if mr.StoreInterval == 0 {
			if err := mr.Repository.SaveGauge(metric.ID, *metric.Value); err != nil {
				logger.Log.Info("error saving gauge metric", zap.Error(err))
				http.Error(res, "error saving gauge metric", http.StatusInternalServerError)
				return
			}
		}
	} else if metric.MType == MetricTypeCounter {
		if metric.Delta == nil {
			http.Error(res, "empty metric delta", http.StatusBadRequest)
			return
		}
		mr.Repository.UpdateCounter(metric.ID, *metric.Delta)
		if mr.StoreInterval == 0 {
			if err := mr.Repository.SaveCounter(metric.ID, *metric.Delta); err != nil {
				logger.Log.Info("error saving counter metric", zap.Error(err))
				http.Error(res, "error saving counter metric", http.StatusInternalServerError)
				return
			}
		}
	} else {
		http.Error(res, "bad metric type", http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

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
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(respMetric); err != nil {
		logger.Log.Info("error encoding response metric", zap.Error(err))
		http.Error(res, "error encoding response metric", http.StatusInternalServerError)
		return
	}
}
