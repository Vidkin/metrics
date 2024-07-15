package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"github.com/Vidkin/metrics/internal/logger"
	me "github.com/Vidkin/metrics/internal/metric"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var Migrations embed.FS

type PostgresStorage struct {
	GaugeMetrics   []*me.Metric
	CounterMetrics []*me.Metric
	AllMetrics     []*me.Metric
	Conn           *sql.DB
}

func (p *PostgresStorage) UpdateMetric(ctx context.Context, metric *me.Metric) error {
	switch metric.MType {
	case MetricTypeGauge:
		_, err := p.GetMetric(ctx, metric.MType, metric.ID)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := p.Conn.ExecContext(ctx, "INSERT INTO gauge (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Value)
			return err
		}
		if err != nil {
			logger.Log.Info("error get gauge metric", zap.Error(err))
			return err
		}
		_, err = p.Conn.ExecContext(ctx, "UPDATE gauge SET metric_value=$1 WHERE metric_name=$2", *metric.Value, metric.ID)
		return err
	case MetricTypeCounter:
		_, err := p.GetMetric(ctx, metric.MType, metric.ID)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := p.Conn.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Delta)
			return err
		}
		if err != nil {
			logger.Log.Info("error get counter metric", zap.Error(err))
			return err
		}
		_, err = p.Conn.ExecContext(ctx, "UPDATE counter SET metric_value=metric_value+$1 WHERE metric_name=$2", *metric.Delta, metric.ID)
		return err
	default:
		return errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) UpdateMetrics(ctx context.Context, metrics *[]me.Metric) error {
	tx, err := p.Conn.Begin()
	if err != nil {
		logger.Log.Info("error begin tx", zap.Error(err))
		return err
	}
	defer tx.Rollback()
	for _, metric := range *metrics {
		switch metric.MType {
		case MetricTypeGauge:
			row := tx.QueryRowContext(ctx, "SELECT metric_id from gauge WHERE metric_name=$1", metric.ID)
			var metricID int64
			err = row.Scan(&metricID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					_, err := tx.ExecContext(ctx, "INSERT INTO gauge (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Value)
					if err != nil {
						logger.Log.Info("error insert gauge metric", zap.Error(err))
						return err
					}
				} else {
					logger.Log.Info("error scan gauge metric", zap.Error(err))
					return err
				}
			}
			_, err = tx.ExecContext(ctx, "UPDATE gauge SET metric_value=$1 WHERE metric_id=$2", *metric.Value, metricID)
			if err != nil {
				logger.Log.Info("error update gauge metric", zap.Error(err))
				return err
			}
		case MetricTypeCounter:
			row := tx.QueryRowContext(ctx, "SELECT metric_id from counter WHERE metric_name=$1", metric.ID)
			var metricID int64
			err = row.Scan(&metricID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					_, err := tx.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Delta)
					if err != nil {
						logger.Log.Info("error insert counter metric", zap.Error(err))
						return err
					}
				} else {
					logger.Log.Info("error scan counter metric", zap.Error(err))
					return err
				}
			}
			_, err = tx.ExecContext(ctx, "UPDATE counter SET metric_value=metric_value+$1 WHERE metric_id=$2", *metric.Delta, metricID)
			if err != nil {
				logger.Log.Info("error update counter metric", zap.Error(err))
				return err
			}
		default:
			logger.Log.Info("unknown metric type")
			return errors.New("unknown metric type")
		}
	}
	return tx.Commit()
}

func (p *PostgresStorage) DeleteMetric(ctx context.Context, mType string, name string) error {
	switch mType {
	case MetricTypeGauge:
		stmt, err := p.Conn.PrepareContext(ctx, "DELETE from gauge WHERE metric_name=$1")
		if err != nil {
			logger.Log.Info("error prepare stmt", zap.Error(err))
			return err
		}
		defer stmt.Close()
		_, err = p.Conn.ExecContext(ctx, "DELETE from gauge WHERE metric_name=$1", name)
		if err != nil {
			logger.Log.Info("error delete gauge metric", zap.Error(err))
		}
		return err
	case MetricTypeCounter:
		stmt, err := p.Conn.PrepareContext(ctx, "DELETE from counter WHERE metric_name=$1")
		if err != nil {
			logger.Log.Info("error prepare stmt", zap.Error(err))
			return err
		}
		defer stmt.Close()
		_, err = stmt.ExecContext(ctx, name)
		if err != nil {
			logger.Log.Info("error delete counter metric", zap.Error(err))
		}
		return err
	default:
		logger.Log.Info("unknown metric type")
		return errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) GetMetric(ctx context.Context, mType string, name string) (*me.Metric, error) {
	switch mType {
	case MetricTypeGauge:
		stmt, err := p.Conn.PrepareContext(ctx, "SELECT metric_name, metric_value from gauge WHERE metric_name=$1")
		if err != nil {
			logger.Log.Info("error prepare stmt", zap.Error(err))
			return nil, err
		}
		defer stmt.Close()
		row := stmt.QueryRowContext(ctx, name)
		var m me.Metric
		err = row.Scan(&m.ID, &m.Value)
		if err != nil {
			logger.Log.Info("error scan gauge metric", zap.Error(err))
			return nil, err
		}
		m.MType = MetricTypeGauge
		return &m, nil
	case MetricTypeCounter:
		stmt, err := p.Conn.PrepareContext(ctx, "SELECT metric_name, metric_value from counter WHERE metric_name=$1")
		if err != nil {
			logger.Log.Info("error prepare stmt", zap.Error(err))
			return nil, err
		}
		defer stmt.Close()
		row := stmt.QueryRowContext(ctx, name)
		var m me.Metric
		err = row.Scan(&m.ID, &m.Delta)
		if err != nil {
			logger.Log.Info("error scan counter metric", zap.Error(err))
			return nil, err
		}
		m.MType = MetricTypeCounter
		return &m, nil
	default:
		logger.Log.Info("unknown metric type")
		return nil, errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) GetMetrics(ctx context.Context) ([]*me.Metric, error) {
	p.AllMetrics = p.AllMetrics[:0]
	if _, err := p.GetGauges(ctx); err != nil {
		logger.Log.Info("error get gauges", zap.Error(err))
		return nil, err
	}
	if _, err := p.GetCounters(ctx); err != nil {
		logger.Log.Info("error get counters", zap.Error(err))
		return nil, err
	}
	p.AllMetrics = append(p.AllMetrics, p.GaugeMetrics...)
	p.AllMetrics = append(p.AllMetrics, p.CounterMetrics...)
	return p.AllMetrics, nil
}

func (p *PostgresStorage) GetGauges(ctx context.Context) ([]*me.Metric, error) {
	p.GaugeMetrics = p.GaugeMetrics[:0]
	stmt, err := p.Conn.PrepareContext(ctx, "SELECT metric_name, metric_value from gauge")
	if err != nil {
		logger.Log.Info("error prepare stmt", zap.Error(err))
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		logger.Log.Info("error get gauges", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Value); err != nil {
			logger.Log.Info("error scan gauge metric", zap.Error(err))
			return nil, err
		}
		m.MType = MetricTypeGauge
		p.GaugeMetrics = append(p.GaugeMetrics, &m)
	}
	if rows.Err() != nil {
		logger.Log.Info("error rows", zap.Error(err))
		return nil, err
	}
	return p.GaugeMetrics, nil
}

func (p *PostgresStorage) GetCounters(ctx context.Context) ([]*me.Metric, error) {
	p.CounterMetrics = p.CounterMetrics[:0]
	stmt, err := p.Conn.PrepareContext(ctx, "SELECT metric_name, metric_value from counter")
	if err != nil {
		logger.Log.Info("error prepare stmt", zap.Error(err))
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		logger.Log.Info("error get counters", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Delta); err != nil {
			logger.Log.Info("error scan counter metric", zap.Error(err))
			return nil, err
		}
		m.MType = MetricTypeCounter
		p.CounterMetrics = append(p.CounterMetrics, &m)
	}
	if rows.Err() != nil {
		logger.Log.Info("error rows", zap.Error(err))
		return nil, err
	}
	return p.CounterMetrics, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.Conn.PingContext(ctx)
}

func (p *PostgresStorage) Close() error {
	return p.Conn.Close()
}
