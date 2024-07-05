package repository

import (
	"context"
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

func (m *MemoryStorage) UpdateMetric(metric *me.Metric) {
	switch metric.MType {
	case MetricTypeGauge:
		m.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		m.Counter[metric.ID] += *metric.Delta
	}
}

func (m *MemoryStorage) DeleteMetric(mType string, name string) {
	switch mType {
	case MetricTypeGauge:
		delete(m.Gauge, name)
	case MetricTypeCounter:
		delete(m.Counter, name)
	}
}

func (m *MemoryStorage) GetMetric(mType string, name string) (*me.Metric, bool) {
	var metric me.Metric
	switch mType {
	case MetricTypeGauge:
		v, ok := m.Gauge[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeGauge
		metric.Value = &v
	case MetricTypeCounter:
		v, ok := m.Counter[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeCounter
		metric.Delta = &v
	}
	return &metric, true
}

func (m *MemoryStorage) GetMetrics() []*me.Metric {
	m.allMetrics = m.allMetrics[:0]
	m.allMetrics = append(m.allMetrics, m.GetGauges()...)
	m.allMetrics = append(m.allMetrics, m.GetCounters()...)
	return m.allMetrics
}

func (m *MemoryStorage) GetGauges() []*me.Metric {
	m.gaugeMetrics = m.gaugeMetrics[:0]
	for k, v := range m.Gauge {
		m.gaugeMetrics = append(m.gaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return m.gaugeMetrics
}

func (m *MemoryStorage) GetCounters() []*me.Metric {
	m.counterMetrics = m.counterMetrics[:0]
	for k, v := range m.Counter {
		m.counterMetrics = append(m.counterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return m.counterMetrics
}

func (m *MemoryStorage) SaveMetric(ctx context.Context, metric *me.Metric) error {
	return nil
}

func (m *MemoryStorage) Save(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Load(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Close() error {
	return nil
}
