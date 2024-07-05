package router

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/pkg/middleware"
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
	UpdateMetric(metric *metric.Metric)
	DeleteMetric(mType string, name string)
	SaveMetric(ctx context.Context, metric *metric.Metric) error

	GetMetric(mType string, name string) (*metric.Metric, bool)
	GetMetrics() []*metric.Metric
	GetGauges() []*metric.Metric
	GetCounters() []*metric.Metric

	Save(ctx context.Context) error
	Load(ctx context.Context) error

	Ping(ctx context.Context) error
}

func NewMetricRouter(router *chi.Mux, repository Repository, serverConfig *config.ServerConfig) *MetricRouter {
	var mr MetricRouter
	router.Use(middleware.Logging)
	router.Use(middleware.Gzip)

	router.Route("/", func(r chi.Router) {
		r.Get("/", mr.RootHandler)
		router.Route("/ping", func(r chi.Router) {
			r.Get("/", mr.PingDBHandler)
		})
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
	mr.StoreInterval = serverConfig.StoreInterval
	mr.LastStoreTime = time.Now()
	return &mr
}

func (mr *MetricRouter) RootHandler(res http.ResponseWriter, _ *http.Request) {
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)

	for _, metric := range mr.Repository.GetMetrics() {
		if metric.MType == MetricTypeGauge {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %v\n", metric.ID, *metric.Value))
		}
		if metric.MType == MetricTypeCounter {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %d\n", metric.ID, *metric.Delta))
		}
	}
}

func (mr *MetricRouter) PingDBHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
	defer cancel()
	if err := mr.Repository.Ping(ctx); err != nil {
		logger.Log.Info("couldn't connect to database")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

func (mr *MetricRouter) GetMetricValueHandler(res http.ResponseWriter, req *http.Request) {
	metricType := chi.URLParam(req, ParamMetricType)
	metricName := chi.URLParam(req, ParamMetricName)

	if metricType != MetricTypeGauge && metricType != MetricTypeCounter {
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
		return
	}

	metric, ok := mr.Repository.GetMetric(metricType, metricName)
	if !ok {
		http.Error(res, "Metric not found", http.StatusNotFound)
		return
	}

	res.WriteHeader(http.StatusOK)
	_, err := res.Write([]byte(metric.ValueAsString()))
	if err != nil {
		logger.Log.Info("can't write metric value", zap.Error(err))
		http.Error(res, "Can't write metric value", http.StatusInternalServerError)
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

	if metricType != MetricTypeGauge && metricType != MetricTypeCounter {
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
		return
	}

	metric := metric.Metric{
		ID:    metricName,
		MType: metricType,
	}

	var (
		floatValue float64
		intValue   int64
		err        error
	)

	if metricType == MetricTypeGauge {
		floatValue, err = strconv.ParseFloat(metricValue, 64)
		metric.Value = &floatValue
	}

	if metricType == MetricTypeCounter {
		intValue, err = strconv.ParseInt(metricValue, 10, 64)
		metric.Delta = &intValue
	}

	if err != nil {
		logger.Log.Info("can't convert metric value", zap.Error(err))
		http.Error(res, "Bad metric value!", http.StatusBadRequest)
		return
	}

	mr.Repository.UpdateMetric(&metric)
	if mr.StoreInterval == 0 {
		if err := mr.Repository.SaveMetric(req.Context(), &metric); err != nil {
			logger.Log.Info("error saving metric", zap.Error(err))
			http.Error(res, "error saving  metric", http.StatusInternalServerError)
			return
		}
	}

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (mr *MetricRouter) UpdateMetricHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var metric metric.Metric
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case MetricTypeGauge:
		if metric.Value == nil {
			http.Error(res, "empty metric value", http.StatusBadRequest)
			return
		}
	case MetricTypeCounter:
		if metric.Delta == nil {
			http.Error(res, "empty metric delta", http.StatusBadRequest)
			return
		}
	default:
		http.Error(res, "bad metric type", http.StatusBadRequest)
		return
	}

	mr.Repository.UpdateMetric(&metric)
	if mr.StoreInterval == 0 {
		if err := mr.Repository.SaveMetric(req.Context(), &metric); err != nil {
			logger.Log.Info("error saving gauge metric", zap.Error(err))
			http.Error(res, "error saving gauge metric", http.StatusInternalServerError)
			return
		}
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

	var metric metric.Metric
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	if metric.MType != MetricTypeGauge && metric.MType != MetricTypeCounter {
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
		return
	}

	respMetric, ok := mr.Repository.GetMetric(metric.MType, metric.ID)
	if !ok {
		http.Error(res, "metric not found", http.StatusNotFound)
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
