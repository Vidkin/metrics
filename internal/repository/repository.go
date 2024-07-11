package repository

import (
	"context"
	me "github.com/Vidkin/metrics/internal/metric"
)

type Repository interface {
	UpdateMetric(ctx context.Context, metric *me.Metric) error
	UpdateMetrics(ctx context.Context, metrics *[]me.Metric) error
	DeleteMetric(ctx context.Context, mType string, name string) error

	GetMetric(ctx context.Context, mType string, name string) (*me.Metric, error)
	GetMetrics(ctx context.Context) ([]*me.Metric, error)
	GetGauges(ctx context.Context) ([]*me.Metric, error)
	GetCounters(ctx context.Context) ([]*me.Metric, error)
}
