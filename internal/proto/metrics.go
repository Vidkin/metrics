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

type MetricsServer struct {
	proto.UnimplementedMetricsServer
	Repository    router.Repository
	LastStoreTime time.Time
	RetryCount    int
	StoreInterval int
}

type Dumper interface {
	Dump(metric *metric.Metric) error
	FullDump() error
}

func DumpMetric(r router.Repository, m *metric.Metric) error {
	if dumper, ok := r.(Dumper); ok {
		return dumper.Dump(m)
	}
	return errors.New("provided Repository does not implement Dumper")
}

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
	// TODO: реализовать интерсепторы для логгирования и для trusted_subnet?
}
