package router

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/Vidkin/metrics/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
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
	Repository    repository.Repository
	Router        chi.Router
	RetryCount    int
	LastStoreTime time.Time
	StoreInterval int
}

type Dumper interface {
	Dump(metric *metric.Metric) error
	FullDump() error
}

func Ping(r repository.Repository, ctx context.Context) error {
	if pinger, ok := r.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return errors.New("provided Repository does not implement Pinger")
}

func DumpMetric(r repository.Repository, m *metric.Metric) error {
	if dumper, ok := r.(Dumper); ok {
		return dumper.Dump(m)
	}
	return errors.New("provided Repository does not implement Dumper")
}

func Close(r repository.Repository) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return errors.New("provided Repository does not implement Pinger")
}

func NewMetricRouter(router *chi.Mux, repository repository.Repository, serverConfig *config.ServerConfig) *MetricRouter {
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
		router.Route("/updates", func(r chi.Router) {
			r.Post("/", mr.UpdateMetricsHandlerJSON)
		})
	})
	mr.Router = router
	mr.Repository = repository
	mr.StoreInterval = serverConfig.StoreInterval
	mr.RetryCount = serverConfig.RetryCount
	mr.LastStoreTime = time.Now()
	return &mr
}

func (mr *MetricRouter) RootHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html")

	var (
		metrics []*metric.Metric
		err     error
	)

	for i := 0; i <= mr.RetryCount; i++ {
		metrics, err = mr.Repository.GetMetrics(req.Context())
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("error get metrics", zap.Error(err))
			http.Error(res, "error get metrics", http.StatusInternalServerError)
			return
		}
		break
	}

	res.WriteHeader(http.StatusOK)
	for _, me := range metrics {
		if me.MType == MetricTypeGauge {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %v\n", me.ID, *me.Value))
		}
		if me.MType == MetricTypeCounter {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %d\n", me.ID, *me.Delta))
		}
	}
}

func (mr *MetricRouter) PingDBHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	if err := Ping(mr.Repository, req.Context()); err != nil {
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

	var (
		me  *metric.Metric
		err error
	)

	for i := 0; i <= mr.RetryCount; i++ {
		me, err = mr.Repository.GetMetric(req.Context(), metricType, metricName)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("metric not found", zap.Error(err))
			http.Error(res, "metric not found", http.StatusNotFound)
			return
		}
		break
	}

	res.WriteHeader(http.StatusOK)
	_, err = res.Write([]byte(me.ValueAsString()))
	if err != nil {
		logger.Log.Info("can't write metric value", zap.Error(err))
		http.Error(res, "Can't write metric value", http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func (mr *MetricRouter) DumpMetric(metric *metric.Metric) error {
	if mr.StoreInterval == 0 {
		for i := 0; i <= mr.RetryCount; i++ {
			err := DumpMetric(mr.Repository, metric)
			if err != nil {
				var pathErr *os.PathError
				if errors.As(err, &pathErr) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
				logger.Log.Info("error saving metric", zap.Error(err))
				return errors.New("error saving metric")
			}
			break
		}
	}
	return nil
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

	me := metric.Metric{
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
		me.Value = &floatValue
	}

	if metricType == MetricTypeCounter {
		intValue, err = strconv.ParseInt(metricValue, 10, 64)
		me.Delta = &intValue
	}

	if err != nil {
		logger.Log.Info("can't convert metric value", zap.Error(err))
		http.Error(res, "Bad metric value!", http.StatusBadRequest)
		return
	}

	for i := 0; i <= mr.RetryCount; i++ {
		err := mr.Repository.UpdateMetric(req.Context(), &me)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("bad metric value", zap.Error(err))
			http.Error(res, "bad metric value", http.StatusInternalServerError)
			return
		}
		break
	}

	if err := mr.DumpMetric(&me); err != nil {
		http.Error(res, "error saving metric", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (mr *MetricRouter) UpdateMetricHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var me metric.Metric
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&me); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	switch me.MType {
	case MetricTypeGauge:
		if me.Value == nil {
			http.Error(res, "empty metric value", http.StatusBadRequest)
			return
		}
	case MetricTypeCounter:
		if me.Delta == nil {
			http.Error(res, "empty metric delta", http.StatusBadRequest)
			return
		}
	default:
		http.Error(res, "bad metric type", http.StatusBadRequest)
		return
	}

	for i := 0; i <= mr.RetryCount; i++ {
		err := mr.Repository.UpdateMetric(req.Context(), &me)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("error update metric", zap.Error(err))
			http.Error(res, "error update metric", http.StatusInternalServerError)
			return
		}
		break
	}

	if err := mr.DumpMetric(&me); err != nil {
		http.Error(res, "error saving metric", http.StatusInternalServerError)
		return
	}
	var (
		actualMetric *metric.Metric
		err          error
	)
	for i := 0; i <= mr.RetryCount; i++ {
		actualMetric, err = mr.Repository.GetMetric(req.Context(), me.MType, me.ID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("error get actual metric value", zap.Error(err))
			http.Error(res, "error get actual metric value", http.StatusInternalServerError)
			return
		}
		break
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(actualMetric); err != nil {
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

	var me metric.Metric
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&me); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	if me.MType != MetricTypeGauge && me.MType != MetricTypeCounter {
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
		return
	}

	var (
		respMetric *metric.Metric
		err        error
	)
	for i := 0; i <= mr.RetryCount; i++ {
		respMetric, err = mr.Repository.GetMetric(req.Context(), me.MType, me.ID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("metric not found", zap.Error(err))
			http.Error(res, "metric not found", http.StatusNotFound)
			return
		}
		break
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

func (mr *MetricRouter) UpdateMetricsHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var metrics []metric.Metric
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metrics); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}

	for _, m := range metrics {
		if (m.Value == nil && m.Delta == nil) || (m.MType != MetricTypeCounter && m.MType != MetricTypeGauge) {
			http.Error(res, "bad metric", http.StatusBadRequest)
			return
		}
	}

	for i := 0; i <= mr.RetryCount; i++ {
		err := mr.Repository.UpdateMetrics(req.Context(), &metrics)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info("error update metrics", zap.Error(err))
			http.Error(res, "error update metrics", http.StatusInternalServerError)
			return
		}
		break
	}

	for _, me := range metrics {
		if err := mr.DumpMetric(&me); err != nil {
			http.Error(res, "error saving metric", http.StatusInternalServerError)
			return
		}
	}

	for i, m := range metrics {
		var (
			updated *metric.Metric
			err     error
		)
		for i := 0; i <= mr.RetryCount; i++ {
			updated, err = mr.Repository.GetMetric(req.Context(), m.MType, m.ID)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgerrcode.IsConnectionException(pgErr.Code) && i != mr.RetryCount {
						logger.Log.Info("repository connection error", zap.Error(err))
						time.Sleep(time.Duration(1+i*2) * time.Second)
						continue
					}
				}
				logger.Log.Info("error get updated metric", zap.Error(err))
				http.Error(res, "error get updated metric", http.StatusInternalServerError)
				return
			}
			break
		}
		metrics[i] = *updated
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(res)
	if err := enc.Encode(metrics); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		http.Error(res, "error encoding response", http.StatusInternalServerError)
		return
	}
}
