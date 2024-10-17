package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	me "github.com/Vidkin/metrics/internal/metric"
)

func TestPostgresStorage_Ping(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "test ping ok",
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = pgStorage.Ping(context.TODO())
			assert.NoError(t, err)
		})
	}
}

func TestPostgresStorage_Close(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "test close ok",
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}

	var pgStorage PostgresStorage
	pgStorage.Conn = adminDB

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = pgStorage.Close()
			assert.NoError(t, err)
		})
	}
}

func TestPostgresStorage_Metric(t *testing.T) {
	floatValue := 16.4
	floatValue2 := 16.5
	intValue := int64(12)
	intValue2 := int64(12)
	tests := []struct {
		metricAdd    *me.Metric
		metricUpdate *me.Metric
		name         string
		wantErr      bool
	}{
		{
			name: "test add gauge metric",
			metricAdd: &me.Metric{
				ID:    "gaugeTest",
				MType: MetricTypeGauge,
				Value: &floatValue,
			},
		},
		{
			name: "test add counter metric",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
		},
		{
			name: "test update gauge metric",
			metricAdd: &me.Metric{
				ID:    "gaugeTestUpdate",
				MType: MetricTypeGauge,
				Value: &floatValue,
			},
			metricUpdate: &me.Metric{
				ID:    "gaugeTestUpdate",
				MType: MetricTypeGauge,
				Value: &floatValue2,
			},
		},
		{
			name: "test update counter metric",
			metricAdd: &me.Metric{
				ID:    "counterTestUpdate",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
			metricUpdate: &me.Metric{
				ID:    "counterTestUpdate",
				MType: MetricTypeCounter,
				Delta: &intValue2,
			},
		},
		{
			name: "test error unknown metric type",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: "unknownMetricType",
				Delta: &intValue,
			},
			wantErr: true,
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
			}
			if tt.metricAdd != nil && tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.Error(t, err)
			}
			if tt.metricAdd != nil && tt.metricUpdate != nil && !tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricUpdate)
				assert.NoError(t, err)
			}
		})
	}
}

func TestPostgresStorage_DeleteMetric(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		metricAdd *me.Metric
		name      string
		wantErr   bool
	}{
		{
			name: "test delete gauge metric ok",
			metricAdd: &me.Metric{
				ID:    "gaugeTest",
				MType: MetricTypeGauge,
				Value: &floatValue,
			},
		},
		{
			name: "test delete counter metric ok",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
		},
		{
			name: "test error unknown metric type",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
			wantErr: true,
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = pgStorage.DeleteMetric(context.TODO(), tt.metricAdd.MType, tt.metricAdd.ID)
				assert.NoError(t, err)
			}
			if tt.metricAdd != nil && tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = pgStorage.DeleteMetric(context.TODO(), "unknownMType", tt.metricAdd.ID)
				assert.Error(t, err)
			}
		})
	}
}

func TestPostgresStorage_GetMetric(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		metricAdd *me.Metric
		name      string
		wantErr   bool
	}{
		{
			name: "test get gauge metric ok",
			metricAdd: &me.Metric{
				ID:    "gaugeTest",
				MType: MetricTypeGauge,
				Value: &floatValue,
			},
		},
		{
			name: "test get counter metric ok",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
		},
		{
			name: "test error unknown metric type",
			metricAdd: &me.Metric{
				ID:    "counterTest",
				MType: MetricTypeCounter,
				Delta: &intValue,
			},
			wantErr: true,
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err = pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				metric, errGet := pgStorage.GetMetric(context.TODO(), tt.metricAdd.MType, tt.metricAdd.ID)
				assert.NoError(t, errGet)
				assert.Equal(t, tt.metricAdd, metric)
			}
			if tt.metricAdd != nil && tt.wantErr {
				errUpdate := pgStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, errUpdate)
				_, errGet := pgStorage.GetMetric(context.TODO(), "unknownMType", tt.metricAdd.ID)
				assert.Error(t, errGet)
			}
		})
	}
}

func TestPostgresStorage_GetMetrics(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		name      string
		metricAdd []*me.Metric
		wantErr   bool
	}{
		{
			name: "test get gauge metrics ok",
			metricAdd: []*me.Metric{
				{
					ID:    "gaugeTest",
					MType: MetricTypeGauge,
					Value: &floatValue,
				},
				{
					ID:    "counterTest",
					MType: MetricTypeCounter,
					Delta: &intValue,
				},
			},
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				for _, metric := range tt.metricAdd {
					err = pgStorage.UpdateMetric(context.TODO(), metric)
					assert.NoError(t, err)
				}
				metrics, err := pgStorage.GetMetrics(context.TODO())
				assert.NoError(t, err)
				assert.Equal(t, len(tt.metricAdd), len(metrics))
				assert.Equal(t, tt.metricAdd, metrics)
			}
		})
	}
}

func TestPostgresStorage_UpdateMetrics(t *testing.T) {
	floatValue := 16.4
	floatValue2 := 15.4
	intValue := int64(12)
	tests := []struct {
		metricsAdd    *[]me.Metric
		metricsUpdate *[]me.Metric
		name          string
		wantErr       bool
	}{
		{
			name: "test add metrics ok",
			metricsAdd: &[]me.Metric{
				{
					ID:    "gaugeTest",
					MType: MetricTypeGauge,
					Value: &floatValue,
				},
				{
					ID:    "counterTest",
					MType: MetricTypeCounter,
					Delta: &intValue,
				},
			},
		},
		{
			name: "test update metrics ok",
			metricsAdd: &[]me.Metric{
				{
					ID:    "gaugeTest",
					MType: MetricTypeGauge,
					Value: &floatValue,
				},
				{
					ID:    "gaugeTest2",
					MType: MetricTypeGauge,
					Value: &floatValue2,
				},
			},
			metricsUpdate: &[]me.Metric{
				{
					ID:    "gaugeTest",
					MType: MetricTypeGauge,
					Value: &floatValue2,
				},
				{
					ID:    "gaugeTest2",
					MType: MetricTypeGauge,
					Value: &floatValue,
				},
			},
		},
		{
			name:    "test error unknown metric type",
			wantErr: true,
			metricsAdd: &[]me.Metric{
				{
					ID:    "gaugeTest",
					MType: "unknownMetricType",
					Value: &floatValue,
				},
			},
		},
	}

	dbDSN := "user=postgres password=postgres dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	adminDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer adminDB.Close()

	var pgStorage PostgresStorage
	pgStorage.GaugeMetrics = make([]*me.Metric, 0)
	pgStorage.CounterMetrics = make([]*me.Metric, 0)
	pgStorage.AllMetrics = make([]*me.Metric, 0)
	pgStorage.Conn = adminDB

	_, err = adminDB.Exec(
		`CREATE TABLE gauge (
			metric_id SERIAL PRIMARY KEY,
			metric_name VARCHAR NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL
		);
	
		 CREATE TABLE counter (
		    metric_id SERIAL PRIMARY KEY,
		    metric_name VARCHAR NOT NULL,
		    metric_value BIGINT NOT NULL
		);`)
	if err != nil {
		fmt.Printf("Ошибка создания таблиц БД: %v\n", err)
	}

	defer func() {
		_, dropErr := adminDB.Exec("DROP TABLE gauge; DROP TABLE counter;")
		if dropErr != nil {
			fmt.Printf("Ошибка удаления таблиц БД: %v\n", dropErr)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAdd != nil && !tt.wantErr {
				err = pgStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.NoError(t, err)
			}
			if tt.metricsAdd != nil && tt.wantErr {
				err = pgStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.Error(t, err)
			}
			if tt.metricsAdd != nil && tt.metricsUpdate != nil && !tt.wantErr {
				err = pgStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.NoError(t, err)

				err = pgStorage.UpdateMetrics(context.TODO(), tt.metricsUpdate)
				assert.NoError(t, err)

				for _, metric := range *tt.metricsUpdate {
					m, err := pgStorage.GetMetric(context.TODO(), metric.MType, metric.ID)
					assert.NoError(t, err)
					assert.Equal(t, metric.ID, m.ID)
					assert.Equal(t, metric.MType, m.MType)
					if metric.MType == MetricTypeGauge {
						assert.Equal(t, *metric.Value, *m.Value)
					}
				}
			}
		})
	}
}
