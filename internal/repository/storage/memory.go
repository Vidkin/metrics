package storage

import (
	"context"
	"errors"
	"sync"

	me "github.com/Vidkin/metrics/internal/metric"
)

const (
	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

type MemoryStorage struct {
	mu             sync.RWMutex
	Gauge          map[string]float64
	Counter        map[string]int64
	GaugeMetrics   []*me.Metric
	CounterMetrics []*me.Metric
	AllMetrics     []*me.Metric
}

func (m *MemoryStorage) UpdateMetric(_ context.Context, metric *me.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch metric.MType {
	case MetricTypeGauge:
		m.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		m.Counter[metric.ID] += *metric.Delta
	}
	return nil
}

func (m *MemoryStorage) UpdateMetrics(_ context.Context, metrics *[]me.Metric) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()

	switch mType {
	case MetricTypeGauge:
		delete(m.Gauge, name)
	case MetricTypeCounter:
		delete(m.Counter, name)
	}
	return nil
}

func (m *MemoryStorage) GetMetric(_ context.Context, mType string, name string) (*me.Metric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
	m.AllMetrics = m.AllMetrics[:0]
	if _, err := m.GetGauges(ctx); err != nil {
		return nil, err
	}
	if _, err := m.GetCounters(ctx); err != nil {
		return nil, err
	}
	m.AllMetrics = append(m.AllMetrics, m.GaugeMetrics...)
	m.AllMetrics = append(m.AllMetrics, m.CounterMetrics...)
	return m.AllMetrics, nil
}

func (m *MemoryStorage) GetGauges(_ context.Context) ([]*me.Metric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GaugeMetrics = m.GaugeMetrics[:0]
	for k, v := range m.Gauge {
		m.GaugeMetrics = append(m.GaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return m.GaugeMetrics, nil
}

func (m *MemoryStorage) GetCounters(_ context.Context) ([]*me.Metric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.CounterMetrics = m.CounterMetrics[:0]
	for k, v := range m.Counter {
		m.CounterMetrics = append(m.CounterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return m.CounterMetrics, nil
}
