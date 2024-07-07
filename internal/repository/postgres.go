package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"github.com/Vidkin/metrics/internal/logger"
	me "github.com/Vidkin/metrics/internal/metric"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrations embed.FS

type PostgresStorage struct {
	gaugeMetrics   []*me.Metric
	counterMetrics []*me.Metric
	allMetrics     []*me.Metric
	db             *sql.DB
}

func NewPostgresStorage(dbDSN string) (*PostgresStorage, error) {
	var p PostgresStorage
	p.gaugeMetrics = make([]*me.Metric, 0)
	p.counterMetrics = make([]*me.Metric, 0)
	p.allMetrics = make([]*me.Metric, 0)

	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		logger.Log.Fatal("error open sql connection", zap.Error(err))
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Log.Fatal("can't create postgres driver for migrations", zap.Error(err))
		return nil, err
	}

	d, err := iofs.New(migrations, "migrations")
	if err != nil {
		logger.Log.Fatal("can't get migrations from FS", zap.Error(err))
		return nil, err
	}

	m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
	if err != nil {
		logger.Log.Fatal("can't create new migrate instance", zap.Error(err))
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Log.Fatal("can't exec migrations", zap.Error(err))
		return nil, err
	}
	p.db = db
	return &p, nil
}

func (p *PostgresStorage) UpdateMetric(ctx context.Context, metric *me.Metric) error {
	switch metric.MType {
	case MetricTypeGauge:
		_, err := p.GetMetric(ctx, metric.MType, metric.ID)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := p.db.ExecContext(ctx, "INSERT INTO gauge (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Value)
			return err
		}
		if err != nil {
			return err
		}
		_, err = p.db.ExecContext(ctx, "UPDATE gauge SET metric_value=$1 WHERE metric_name=$2", *metric.Value, metric.ID)
		return err
	case MetricTypeCounter:
		_, err := p.GetMetric(ctx, metric.MType, metric.ID)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := p.db.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Delta)
			return err
		}
		if err != nil {
			return err
		}
		_, err = p.db.ExecContext(ctx, "UPDATE counter SET metric_value=metric_value+$1 WHERE metric_name=$2", *metric.Delta, metric.ID)
		return err
	default:
		return errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) UpdateMetrics(ctx context.Context, metrics *[]me.Metric) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, metric := range *metrics {
		switch metric.MType {
		case MetricTypeGauge:
			row := tx.QueryRowContext(ctx, "SELECT metric_id from gauge WHERE metric_name=$1", metric.ID)
			var metricId int64
			err = row.Scan(&metricId)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					_, err := tx.ExecContext(ctx, "INSERT INTO gauge (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Value)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
			_, err = tx.ExecContext(ctx, "UPDATE gauge SET metric_value=$1 WHERE metric_id=$2", *metric.Value, metricId)
			if err != nil {
				return err
			}
		case MetricTypeCounter:
			row := tx.QueryRowContext(ctx, "SELECT metric_id from counter WHERE metric_name=$1", metric.ID)
			var metricId int64
			err = row.Scan(&metricId)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					_, err := tx.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, *metric.Delta)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
			_, err = tx.ExecContext(ctx, "UPDATE counter SET metric_value=metric_value+$1 WHERE metric_id=$2", *metric.Delta, metricId)
			if err != nil {
				return err
			}
		default:
			return errors.New("unknown metric type")
		}
	}
	return tx.Commit()
}

func (p *PostgresStorage) DeleteMetric(ctx context.Context, mType string, name string) error {
	switch mType {
	case MetricTypeGauge:
		stmt, err := p.db.PrepareContext(ctx, "DELETE from gauge WHERE metric_name=$1")
		if err != nil {
			return err
		}
		_, err = stmt.ExecContext(ctx, name)
		return err
	case MetricTypeCounter:
		stmt, err := p.db.PrepareContext(ctx, "DELETE from counter WHERE metric_name=$1")
		if err != nil {
			return err
		}
		_, err = stmt.ExecContext(ctx, name)
		return err
	default:
		return errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) GetMetric(ctx context.Context, mType string, name string) (*me.Metric, error) {
	switch mType {
	case MetricTypeGauge:
		stmt, err := p.db.PrepareContext(ctx, "SELECT metric_name, metric_value from gauge WHERE metric_name=$1")
		if err != nil {
			return nil, err
		}
		row := stmt.QueryRowContext(ctx, name)
		var m me.Metric
		err = row.Scan(&m.ID, &m.Value)
		if err != nil {
			return nil, err
		}
		m.MType = MetricTypeGauge
		return &m, nil
	case MetricTypeCounter:
		stmt, err := p.db.PrepareContext(ctx, "SELECT metric_name, metric_value from counter WHERE metric_name=$1")
		if err != nil {
			return nil, err
		}
		row := stmt.QueryRowContext(ctx, name)
		var m me.Metric
		err = row.Scan(&m.ID, &m.Delta)
		if err != nil {
			return nil, err
		}
		m.MType = MetricTypeCounter
		return &m, nil
	default:
		return nil, errors.New("unknown metric type")
	}
}

func (p *PostgresStorage) GetMetrics(ctx context.Context) ([]*me.Metric, error) {
	p.allMetrics = p.allMetrics[:0]
	if _, err := p.GetGauges(ctx); err != nil {
		return nil, err
	}
	if _, err := p.GetCounters(ctx); err != nil {
		return nil, err
	}
	p.allMetrics = append(p.allMetrics, p.gaugeMetrics...)
	p.allMetrics = append(p.allMetrics, p.counterMetrics...)
	return p.allMetrics, nil
}

func (p *PostgresStorage) GetGauges(ctx context.Context) ([]*me.Metric, error) {
	p.gaugeMetrics = p.gaugeMetrics[:0]
	stmt, err := p.db.PrepareContext(ctx, "SELECT metric_name, metric_value from gauge")
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Value); err != nil {
			return nil, err
		}
		m.MType = MetricTypeGauge
		p.gaugeMetrics = append(p.gaugeMetrics, &m)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return p.gaugeMetrics, nil
}

func (p *PostgresStorage) GetCounters(ctx context.Context) ([]*me.Metric, error) {
	p.counterMetrics = p.counterMetrics[:0]
	stmt, err := p.db.PrepareContext(ctx, "SELECT metric_name, metric_value from counter")
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Delta); err != nil {
			return nil, err
		}
		m.MType = MetricTypeCounter
		p.counterMetrics = append(p.counterMetrics, &m)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return p.counterMetrics, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
