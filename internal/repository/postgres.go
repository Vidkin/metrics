package repository

import (
	"context"
	"database/sql"
	"embed"
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
	Gauge          map[string]float64
	Counter        map[string]int64
	gaugeMetrics   []*me.Metric
	counterMetrics []*me.Metric
	allMetrics     []*me.Metric
	db             *sql.DB
}

func NewPostgresStorage(dbDSN string) (*PostgresStorage, error) {
	var p PostgresStorage
	p.Gauge = make(map[string]float64)
	p.Counter = make(map[string]int64)
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

func (p *PostgresStorage) UpdateMetric(metric *me.Metric) {
	switch metric.MType {
	case MetricTypeGauge:
		p.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		p.Counter[metric.ID] += *metric.Delta
	}
}

func (p *PostgresStorage) DeleteMetric(mType string, name string) {
	switch mType {
	case MetricTypeGauge:
		delete(p.Gauge, name)
	case MetricTypeCounter:
		delete(p.Counter, name)
	}
}

func (p *PostgresStorage) GetMetric(mType string, name string) (*me.Metric, bool) {
	var metric me.Metric
	switch mType {
	case MetricTypeGauge:
		v, ok := p.Gauge[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeGauge
		metric.Value = &v
	case MetricTypeCounter:
		v, ok := p.Counter[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeCounter
		metric.Delta = &v
	}
	return &metric, true
}

func (p *PostgresStorage) GetMetrics() []*me.Metric {
	p.allMetrics = p.allMetrics[:0]
	p.allMetrics = append(p.allMetrics, p.GetGauges()...)
	p.allMetrics = append(p.allMetrics, p.GetCounters()...)
	return p.allMetrics
}

func (p *PostgresStorage) GetGauges() []*me.Metric {
	p.gaugeMetrics = p.gaugeMetrics[:0]
	for k, v := range p.Gauge {
		p.gaugeMetrics = append(p.gaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return p.gaugeMetrics
}

func (p *PostgresStorage) GetCounters() []*me.Metric {
	p.counterMetrics = p.counterMetrics[:0]
	for k, v := range p.Counter {
		p.counterMetrics = append(p.counterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return p.counterMetrics
}

func (p *PostgresStorage) SaveMetric(ctx context.Context, metric *me.Metric) error {
	var row *sql.Row
	var id string

	if metric.MType == MetricTypeGauge {
		row = p.db.QueryRowContext(ctx, "SELECT metric_id FROM gauge WHERE metric_name = $1", metric.ID)
	} else {
		row = p.db.QueryRowContext(ctx, "SELECT metric_id FROM counter WHERE metric_name = $1", metric.ID)
	}
	if err := row.Scan(&id); err != nil && err != sql.ErrNoRows {
		return err
	}

	var err error
	switch metric.MType {
	case MetricTypeGauge:
		if id != "" {
			_, err = p.db.ExecContext(ctx, "UPDATE gauge SET metric_value = $1", metric.Value)
		} else {
			_, err = p.db.ExecContext(ctx, "INSERT INTO gauge (metric_name, metric_value) VALUES ($1, $2)", metric.ID, metric.Value)
		}
	case MetricTypeCounter:
		if id != "" {
			_, err = p.db.ExecContext(ctx, "UPDATE counter SET metric_value = $1", metric.Delta)
		} else {
			_, err = p.db.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, metric.Delta)
		}
	}
	return err
}

func (p *PostgresStorage) Save(ctx context.Context) error {
	gauge := p.GetGauges()
	counter := p.GetCounters()
	metrics := append(gauge, counter...)

	var (
		err error
		row *sql.Row
		id  string
	)
	for _, metric := range metrics {
		if metric.MType == MetricTypeGauge {
			row = p.db.QueryRowContext(ctx, "SELECT metric_id FROM gauge WHERE metric_name = $1", metric.ID)
		} else {
			row = p.db.QueryRowContext(ctx, "SELECT metric_id FROM counter WHERE metric_name = $1", metric.ID)
		}
		if err = row.Scan(&id); err != nil && err != sql.ErrNoRows {
			return err
		}
		if id != "" {
			if metric.MType == MetricTypeGauge {
				_, err = p.db.ExecContext(ctx, "UPDATE gauge SET metric_value = $1", metric.Value)
			} else {
				_, err = p.db.ExecContext(ctx, "UPDATE gauge SET metric_value = $1", metric.Delta)
			}
		} else {
			if metric.MType == MetricTypeGauge {
				_, err = p.db.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, metric.Value)
			} else {
				_, err = p.db.ExecContext(ctx, "INSERT INTO counter (metric_name, metric_value) VALUES ($1, $2)", metric.ID, metric.Delta)
			}
		}

	}
	return err
}

func (p *PostgresStorage) Load(ctx context.Context) error {
	rows, err := p.db.QueryContext(ctx, "SELECT metric_name, metric_value from gauge")
	if err != nil {
		return err
	}

	p.allMetrics = p.allMetrics[:0]
	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Value); err != nil {
			return err
		}
		m.MType = MetricTypeGauge
		p.allMetrics = append(p.allMetrics, &m)
	}

	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()

	rows, err = p.db.QueryContext(ctx, "SELECT metric_name, metric_value from counter")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var m me.Metric
		if err := rows.Scan(&m.ID, &m.Delta); err != nil {
			return err
		}
		m.MType = MetricTypeCounter
		p.allMetrics = append(p.allMetrics, &m)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	for _, metric := range p.allMetrics {
		p.UpdateMetric(metric)
	}
	return nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
