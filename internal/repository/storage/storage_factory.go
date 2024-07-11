package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	me "github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
	"os"
	"time"
)

func NewMemoryStorage() *MemoryStorage {
	var m MemoryStorage
	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)
	m.GaugeMetrics = make([]*me.Metric, 0)
	m.CounterMetrics = make([]*me.Metric, 0)
	m.AllMetrics = make([]*me.Metric, 0)
	return &m
}

func NewFileStorage(fileStoragePath string) *FileStorage {
	var f FileStorage
	f.Gauge = make(map[string]float64)
	f.Counter = make(map[string]int64)
	f.GaugeMetrics = make([]*me.Metric, 0)
	f.CounterMetrics = make([]*me.Metric, 0)
	f.AllMetrics = make([]*me.Metric, 0)
	f.FileStoragePath = fileStoragePath
	return &f
}

func NewPostgresStorage(dbDSN string) (*PostgresStorage, error) {
	var p PostgresStorage
	p.GaugeMetrics = make([]*me.Metric, 0)
	p.CounterMetrics = make([]*me.Metric, 0)
	p.AllMetrics = make([]*me.Metric, 0)

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

	d, err := iofs.New(Migrations, "migrations")
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
	p.conn = db
	return &p, nil
}

func NewRepository(cfg *config.ServerConfig) (repository.Repository, error) {
	if cfg.DatabaseDSN != "" {
		return NewPostgresStorage(cfg.DatabaseDSN)
	}

	if cfg.FileStoragePath != "" {
		fileStorage := NewFileStorage(cfg.FileStoragePath)
		if cfg.Restore {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			for i := 0; i <= cfg.RetryCount; i++ {
				err := fileStorage.Load(ctx)
				if err != nil {
					var pathErr *os.PathError
					if errors.As(err, &pathErr) && i != cfg.RetryCount {
						logger.Log.Info("repository connection error", zap.Error(err))
						time.Sleep(time.Duration(1+i*2) * time.Second)
						continue
					}
					logger.Log.Info("error load saved metrics", zap.Error(err))
				}
				break
			}
		}
		return fileStorage, nil
	}

	memStorage := NewMemoryStorage()
	return memStorage, nil
}
