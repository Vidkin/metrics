package repository

import (
	"context"
	"errors"
	me "github.com/Vidkin/metrics/internal/metric"
)

const (
	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

type MemoryStorage struct {
	Gauge          map[string]float64
	Counter        map[string]int64
	gaugeMetrics   []*me.Metric
	counterMetrics []*me.Metric
	allMetrics     []*me.Metric
}

func NewMemoryStorage() *MemoryStorage {
	var m MemoryStorage
	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)
	m.gaugeMetrics = make([]*me.Metric, 0)
	m.counterMetrics = make([]*me.Metric, 0)
	m.allMetrics = make([]*me.Metric, 0)
	return &m
}

func (m *MemoryStorage) UpdateMetric(_ context.Context, metric *me.Metric) error {
	switch metric.MType {
	case MetricTypeGauge:
		m.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		m.Counter[metric.ID] += *metric.Delta
	}
	return nil
}

func (m *MemoryStorage) UpdateMetrics(_ context.Context, metrics *[]me.Metric) error {
	for _, metric := range *metrics {
		switch metric.MType {
		case MetricTypeGauge:
			m.Gauge[metric.ID] = *metric.Value
		case MetricTypeCounter:
			m.Counter[metric.ID] += *metric.Delta
		}
	}
	return nil
}

func (m *MemoryStorage) DeleteMetric(_ context.Context, mType string, name string) error {
	switch mType {
	case MetricTypeGauge:
		delete(m.Gauge, name)
	case MetricTypeCounter:
		delete(m.Counter, name)
	}
	return nil
}

func (m *MemoryStorage) GetMetric(_ context.Context, mType string, name string) (*me.Metric, error) {
	var metric me.Metric
	switch mType {
	case MetricTypeGauge:
		v, ok := m.Gauge[name]
		if !ok {
			return nil, errors.New("metric not found")
		}
		metric.ID = name
		metric.MType = MetricTypeGauge
		metric.Value = &v
	case MetricTypeCounter:
		v, ok := m.Counter[name]
		if !ok {
			return nil, errors.New("metric not found")
		}
		metric.ID = name
		metric.MType = MetricTypeCounter
		metric.Delta = &v
	}
	return &metric, nil
}

func (m *MemoryStorage) GetMetrics(ctx context.Context) ([]*me.Metric, error) {
	m.allMetrics = m.allMetrics[:0]
	if _, err := m.GetGauges(ctx); err != nil {
		return nil, err
	}
	if _, err := m.GetCounters(ctx); err != nil {
		return nil, err
	}
	m.allMetrics = append(m.allMetrics, m.gaugeMetrics...)
	m.allMetrics = append(m.allMetrics, m.counterMetrics...)
	return m.allMetrics, nil
}

func (m *MemoryStorage) GetGauges(_ context.Context) ([]*me.Metric, error) {
	m.gaugeMetrics = m.gaugeMetrics[:0]
	for k, v := range m.Gauge {
		m.gaugeMetrics = append(m.gaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return m.gaugeMetrics, nil
}

func (m *MemoryStorage) GetCounters(_ context.Context) ([]*me.Metric, error) {
	m.counterMetrics = m.counterMetrics[:0]
	for k, v := range m.Counter {
		m.counterMetrics = append(m.counterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return m.counterMetrics, nil
}

func (m *MemoryStorage) Ping(_ context.Context) error {
	return nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
