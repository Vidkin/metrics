// Package router provides an HTTP routing implementation for handling metrics-related operations.
package router

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/pkg/middleware"
)

// Constants for metric parameters and types.
//
// These constants are used in the context of metrics handling within the
// MetricRouter. They define the names of URL parameters and the types of
// metrics that can be processed.
//
// Parameters:
//   - ParamMetricType: The name of the URL parameter that specifies the type
//     of the metric (e.g., "counter" or "gauge").
//   - ParamMetricName: The name of the URL parameter that specifies the name
//     of the metric.
//   - ParamMetricValue: The name of the URL parameter that specifies the value
//     of the metric.
//
// Metric Types:
//   - MetricTypeCounter: A constant representing the "counter" metric type.
//   - MetricTypeGauge: A constant representing the "gauge" metric type.
const (
	ParamMetricType  = "metricType"
	ParamMetricName  = "metricName"
	ParamMetricValue = "metricValue"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

// MetricRouter is a struct that manages HTTP routing for metrics-related
// operations. It holds a reference to a Repository for data storage and
// retrieval, as well as a chi.Router for handling HTTP requests. The
// MetricRouter also maintains configuration settings such as the number
// of retry attempts for database operations, the last time metrics were
// stored, and the interval for storing metrics.
//
// Fields:
//   - Repository: An instance of the Repository interface that provides
//     methods for updating, retrieving, and deleting metrics from a data
//     store.
//   - Router: A chi.Router instance that defines the routing for HTTP
//     requests related to metrics.
//   - RetryCount: The number of times to retry database operations in case
//     of transient errors.
//   - LastStoreTime: A time.Time value that indicates the last time metrics
//     were successfully stored in the repository.
//   - StoreInterval: An integer that specifies the interval for storing
//     metrics, which can be used to control when metrics should be dumped
//     to the repository.
type MetricRouter struct {
	Repository    Repository
	Router        chi.Router
	LastStoreTime time.Time
	RetryCount    int
	StoreInterval int
}

// Repository defines the methods required for a metrics data store.
// It provides an abstraction for updating, deleting, and retrieving
// metrics from a persistent storage solution. Implementations of this
// interface should handle the underlying data storage logic, allowing
// the MetricRouter to interact with various data sources without
// being tightly coupled to a specific implementation.
type Repository interface {
	UpdateMetric(ctx context.Context, metric *metric.Metric) error
	UpdateMetrics(ctx context.Context, metrics *[]metric.Metric) error
	DeleteMetric(ctx context.Context, mType string, name string) error

	GetMetric(ctx context.Context, mType string, name string) (*metric.Metric, error)
	GetMetrics(ctx context.Context) ([]*metric.Metric, error)
	GetGauges(ctx context.Context) ([]*metric.Metric, error)
	GetCounters(ctx context.Context) ([]*metric.Metric, error)
}

// Dumper defines the methods required for dump metrics
// to a storage system or output format. Implementations of this interface
// should provide functionality to save individual metrics as well as to
// perform a complete dump of all metrics.
type Dumper interface {
	Dump(metric *metric.Metric) error
	FullDump() error
}

// Ping checks the availability of the provided Repository by attempting to
// ping it. If the Repository implements the driver.Pinger interface, it
// calls the Ping method on it, passing the provided context. If the
// Repository does not implement the Pinger interface, it returns an error
// indicating that the provided Repository does not support pinging.
//
// Parameters:
//   - r: A Repository instance that is expected to implement the Pinger
//     interface.
//   - ctx: A context.Context to control the lifetime of the ping operation.
//
// Returns:
//   - An error if the ping operation fails or if the Repository does not
//     implement the Pinger interface; otherwise, it returns nil.
func Ping(r Repository, ctx context.Context) error {
	if pinger, ok := r.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return errors.New("provided Repository does not implement Pinger")
}

// DumpMetric attempts to dump a given metric to the provided Repository.
// It checks if the Repository implements the Dumper interface. If it does,
// the function calls the Dump method of the Dumper interface to persist
// the metric. If the Repository does not implement the Dumper interface,
// the function returns an error indicating that the provided Repository
// cannot perform the dump operation.
//
// Parameters:
//   - r: A Repository instance that is expected to implement the Dumper
//     interface.
//   - m: A pointer to the metric.Metric that needs to be dumped.
//
// Returns:
//   - An error if the dumping operation fails or if the Repository does
//     not implement the Dumper interface; otherwise, it returns nil.
func DumpMetric(r Repository, m *metric.Metric) error {
	if dumper, ok := r.(Dumper); ok {
		return dumper.Dump(m)
	}
	return errors.New("provided Repository does not implement Dumper")
}

// Close attempts to close the provided Repository if it implements the
// io.Closer interface. If the Repository does implement the Closer
// interface, the function calls its Close method to release any resources
// or connections. If the Repository does not implement the Closer
// interface, the function returns an error indicating that the provided
// Repository cannot be closed.
//
// Parameters:
//   - r: A Repository instance that is expected to implement the io.Closer
//     interface.
//
// Returns:
//   - An error if the closing operation fails or if the Repository does
//     not implement the Closer interface; otherwise, it returns nil.
func Close(r Repository) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return errors.New("provided Repository does not implement Closer")
}

// NewMetricRouter initializes a new MetricRouter with the provided chi.Mux,
// Repository, and server configuration. It sets up the necessary middleware
// for logging, hashing (if a key is provided), and gzip compression. The
// function also defines the routing for various HTTP endpoints related to
// metrics, including handlers for retrieving, updating, and checking the
// status of metrics.
//
// Parameters:
//   - router: A pointer to a chi.Mux instance that will handle the HTTP
//     routing for the metrics API.
//   - repository: An instance of the Repository interface that will be
//     used for storing and retrieving metrics data.
//   - serverConfig: A pointer to a config.ServerConfig struct that contains
//     configuration settings such as the store interval and retry count.
//
// Returns:
//   - A pointer to a newly created MetricRouter instance, which is ready
//     to handle HTTP requests related to metrics.
func NewMetricRouter(router *chi.Mux, repository Repository, serverConfig *config.ServerConfig) *MetricRouter {
	var mr MetricRouter
	router.Use(middleware.Logging)
	if serverConfig.TrustedSubnet != "" {
		router.Use(middleware.TrustedSubnet(serverConfig.TrustedSubnet))
	}
	if serverConfig.Key != "" {
		router.Use(middleware.Hash(serverConfig.Key))
	}
	router.Use(middleware.Gzip)

	router.Route("/", func(r chi.Router) {
		r.Get("/", mr.RootHandler)
		r.Route("/ping", func(r chi.Router) {
			r.Get("/", mr.PingDBHandler)
		})
		r.Route("/value", func(r chi.Router) {
			r.Post("/", mr.GetMetricValueHandlerJSON)
			r.Get("/{metricType}/{metricName}", mr.GetMetricValueHandler)
		})
		r.Route("/update", func(r chi.Router) {
			r.Post("/", mr.UpdateMetricHandlerJSON)
			r.Post("/{metricType}/{metricName}/{metricValue}", mr.UpdateMetricHandler)
		})
		r.Route("/updates", func(r chi.Router) {
			r.Post("/", mr.UpdateMetricsHandlerJSON)
		})
	})
	mr.Router = router
	mr.Repository = repository
	mr.StoreInterval = (int)(serverConfig.StoreInterval)
	mr.RetryCount = serverConfig.RetryCount
	mr.LastStoreTime = time.Now()
	return &mr
}

// RootHandler handles HTTP GET requests to the root endpoint ("/") of the
// metrics API. It retrieves all metrics from the repository and writes
// them to the HTTP response in a plain text format. The response content
// type is set to "text/html".
//
// Parameters:
//   - res: An http.ResponseWriter used to construct the HTTP response.
//   - req: An http.Request containing the details of the incoming request.
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

	for _, me := range metrics {
		if me.MType == MetricTypeGauge {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %v\n", me.ID, *me.Value))
		}
		if me.MType == MetricTypeCounter {
			_, _ = io.WriteString(res, fmt.Sprintf("%s = %d\n", me.ID, *me.Delta))
		}
	}

	res.WriteHeader(http.StatusOK)
}

// PingDBHandler handles HTTP GET requests to the "/ping" endpoint of the
// metrics API. It checks the availability of the database by attempting to
// ping the repository associated with the MetricRouter.
//
// Parameters:
//   - res: An http.ResponseWriter used to construct the HTTP response.
//   - req: An http.Request containing the details of the incoming request.
func (mr *MetricRouter) PingDBHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	if err := Ping(mr.Repository, req.Context()); err != nil {
		logger.Log.Info("couldn't connect to database")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// GetMetricValueHandler handles HTTP GET requests to retrieve the value of
// a specific metric identified by its type and name. The metric type must
// be either "gauge" or "counter".
//
// Parameters:
//   - res: An http.ResponseWriter used to construct the HTTP response.
//   - req: An http.Request containing the details of the incoming request.
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

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err = res.Write([]byte(me.ValueAsString()))
	if err != nil {
		logger.Log.Info("can't write metric value", zap.Error(err))
		http.Error(res, "Can't write metric value", http.StatusInternalServerError)
	}
	res.WriteHeader(http.StatusOK)
}

// DumpMetric attempts to persist a given metric to the repository if the
// StoreInterval is set to zero. The method retries the dumping operation
// up to the configured RetryCount in case of transient errors, such as
// connection issues.
//
// Parameters:
//   - metric: A pointer to the metric.Metric that needs to be dumped.
//
// Returns:
//   - An error if the dumping operation fails; otherwise, it returns nil.
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

// UpdateMetricHandler handles HTTP POST requests to update the value of
// a specific metric identified by its type and name. The metric type must
// be either "gauge" or "counter".
//
// Parameters:
//   - res: An http.ResponseWriter used to construct the HTTP response.
//   - req: An http.Request containing the details of the incoming request.
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

// UpdateMetricHandlerJSON handles HTTP POST requests for updating a metric in JSON format.
//
// Parameters:
// - res: An http.ResponseWriter used to construct the HTTP response.
// - req: An http.Request containing the HTTP request data, including the JSON body.
func (mr *MetricRouter) UpdateMetricHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var me metric.Metric
	if err := json.NewDecoder(req.Body).Decode(&me); err != nil {
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
	data, err := json.Marshal(actualMetric)
	if err != nil {
		logger.Log.Info("error marshal json response", zap.Error(err))
		http.Error(res, "error marshal json response", http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		logger.Log.Info("error write response data", zap.Error(err))
		http.Error(res, "error write response data", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// GetMetricValueHandlerJSON handles HTTP POST requests for retrieving a metric value in JSON format.
//
// Parameters:
// - res: An http.ResponseWriter used to construct the HTTP response.
// - req: An http.Request containing the HTTP request data, including the JSON body.
func (mr *MetricRouter) GetMetricValueHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var me metric.Metric
	if err := json.NewDecoder(req.Body).Decode(&me); err != nil {
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

	data, err := json.Marshal(respMetric)
	if err != nil {
		logger.Log.Info("error marshal json response", zap.Error(err))
		http.Error(res, "error marshal json response", http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		logger.Log.Info("error write response data", zap.Error(err))
		http.Error(res, "error write response data", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// UpdateMetricsHandlerJSON handles HTTP POST requests for updating multiple metrics in JSON format.
//
// Parameters:
// - res: An http.ResponseWriter used to construct the HTTP response.
// - req: An http.Request containing the HTTP request data, including the JSON body.
func (mr *MetricRouter) UpdateMetricsHandlerJSON(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, "only application/json content-type allowed", http.StatusBadRequest)
		return
	}

	var metrics []metric.Metric
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		http.Error(res, "can't decode request body", http.StatusBadRequest)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			logger.Log.Info("can't close request body", zap.Error(err))
		}
	}(req.Body)

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
		for r := 0; r <= mr.RetryCount; r++ {
			updated, err = mr.Repository.GetMetric(req.Context(), m.MType, m.ID)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgerrcode.IsConnectionException(pgErr.Code) && r != mr.RetryCount {
						logger.Log.Info("repository connection error", zap.Error(err))
						time.Sleep(time.Duration(1+r*2) * time.Second)
						continue
					}
				}
				logger.Log.Info("error get updated metric", zap.Error(err))
				http.Error(res, "error get updated metric", http.StatusInternalServerError)
				return
			}
			metrics[i] = *updated
			break
		}
	}
	res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res)
	if err := enc.Encode(metrics); err != nil {
		logger.Log.Info("error encoding response", zap.Error(err))
		http.Error(res, "error encoding response", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}
