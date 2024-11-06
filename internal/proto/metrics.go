// Package proto provides a gRPC server implementation for handling metrics-related operations.
// It defines the MetricsServer struct and methods for updating and dumping metrics to a repository.
package proto

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/proto"
)

// MetricsServer is a gRPC server that handles metrics-related operations.
// It implements the proto.MetricsServer interface and provides methods for
// updating and dumping metrics to a specified repository.
type MetricsServer struct {
	proto.UnimplementedMetricsServer
	Repository    router.Repository // Repository for storing metrics
	LastStoreTime time.Time         // Last time metrics were successfully stored
	RetryCount    int               // Number of retry attempts for database operations
	StoreInterval int               // Interval for storing metrics
}

// Dumper defines the methods required for dumping metrics to a storage system.
// Implementations of this interface should provide functionality to save individual metrics
// as well as to perform a complete dump of all metrics.
type Dumper interface {
	Dump(metric *metric.Metric) error // Dumps a single metric
	FullDump() error                  // Performs a complete dump of all metrics
}

// DumpMetric attempts to dump a given metric to the provided Repository.
// It checks if the Repository implements the Dumper interface. If it does,
// the function calls the Dump method of the Dumper interface to persist
// the metric. If the Repository does not implement the Dumper interface,
// the function returns an error indicating that the provided Repository
// cannot perform the dump operation.
//
// Parameters:
//   - r: A Repository instance that is expected to implement the Dumper interface.
//   - m: A pointer to the metric.Metric that needs to be dumped.
//
// Returns:
//   - An error if the dumping operation fails or if the Repository does not
//     implement the Dumper interface; otherwise, it returns nil.
func DumpMetric(r router.Repository, m *metric.Metric) error {
	if dumper, ok := r.(Dumper); ok {
		return dumper.Dump(m)
	}
	return errors.New("provided Repository does not implement Dumper")
}

// DumpMetric attempts to dump a given metric to the MetricsServer's Repository.
// It retries the dump operation based on the configured RetryCount and handles
// connection errors by waiting and retrying.
//
// Parameters:
//   - metric: A pointer to the metric.Metric that needs to be dumped.
//
// Returns:
//   - An error if the dumping operation fails; otherwise, it returns nil.
func (m *MetricsServer) DumpMetric(metric *metric.Metric) error {
	if m.StoreInterval == 0 {
		for i := 0; i <= m.RetryCount; i++ {
			err := DumpMetric(m.Repository, metric)
			if err != nil {
				var pathErr *os.PathError
				if errors.As(err, &pathErr) && i != m.RetryCount {
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

// UpdateMetrics handles the gRPC request to update multiple metrics.
// It processes the incoming UpdateMetricsRequest, updates the metrics in the repository,
// and returns the updated metrics in the response.
//
// Parameters:
//   - ctx: A context.Context to control the lifetime of the operation.
//   - in: A pointer to the proto.UpdateMetricsRequest containing the metrics to update.
//
// Returns:
//   - A pointer to the proto.UpdateMetricsResponse containing the updated metrics,
//     or an error if the operation fails.
func (m *MetricsServer) UpdateMetrics(ctx context.Context, in *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	var response proto.UpdateMetricsResponse
	var metrics []metric.Metric

	for _, protoMetric := range in.Metrics {
		var me metric.Metric
		if protoMetric.Type == proto.Metric_GAUGE {
			me = metric.Metric{
				ID:    protoMetric.Id,
				MType: router.MetricTypeGauge,
				Value: &protoMetric.Value,
			}
		} else if protoMetric.Type == proto.Metric_COUNTER {
			me = metric.Metric{
				ID:    protoMetric.Id,
				MType: router.MetricTypeCounter,
				Delta: &protoMetric.Delta,
			}
		} else {
			logger.Log.Info(`unknown metric type`)
			return nil, status.Errorf(codes.InvalidArgument, `unknown metric type`)
		}
		metrics = append(metrics, me)
	}

	for i := 0; i <= m.RetryCount; i++ {
		err := m.Repository.UpdateMetrics(ctx, &metrics)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsConnectionException(pgErr.Code) && i != m.RetryCount {
					logger.Log.Info("repository connection error", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
			}
			logger.Log.Info(`can't update metrics in database`, zap.Error(err))
			return nil, status.Errorf(codes.Internal, `can't update metrics in database`)
		}
		break
	}

	for _, me := range metrics {
		if err := m.DumpMetric(&me); err != nil {
			logger.Log.Info(`error saving metrics`, zap.Error(err))
			return nil, status.Errorf(codes.Internal, `error saving metrics`)
		}
	}

	for _, met := range metrics {
		var (
			updated *metric.Metric
			err     error
		)
		for r := 0; r <= m.RetryCount; r++ {
			updated, err = m.Repository.GetMetric(ctx, met.MType, met.ID)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgerrcode.IsConnectionException(pgErr.Code) && r != m.RetryCount {
						logger.Log.Info("repository connection error", zap.Error(err))
						time.Sleep(time.Duration(1+r*2) * time.Second)
						continue
					}
				}
				logger.Log.Info("error get updated metric", zap.Error(err))
				return nil, status.Errorf(codes.Internal, `error get updated metric`)
			}
			prMetric := &proto.Metric{
				Id: updated.ID,
			}
			if updated.MType == router.MetricTypeGauge {
				prMetric.Type = proto.Metric_GAUGE
				prMetric.Value = *updated.Value
			} else {
				prMetric.Type = proto.Metric_COUNTER
				prMetric.Delta = *updated.Delta
			}
			response.Metrics = append(response.Metrics, prMetric)
			break
		}
	}

	return &response, nil
}
