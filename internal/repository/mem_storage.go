package repository

import (
	"encoding/json"
	me "github.com/Vidkin/metrics/internal/metric"
	"io"
	"os"
)

const (
	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

type MemStorage struct {
	Gauge           map[string]float64
	Counter         map[string]int64
	gaugeMetrics    []*me.Metric
	counterMetrics  []*me.Metric
	allMetrics      []*me.Metric
	FileStoragePath string
}

func NewMemoryStorage(fileStoragePath string) *MemStorage {
	var m MemStorage
	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)
	m.gaugeMetrics = make([]*me.Metric, 0)
	m.counterMetrics = make([]*me.Metric, 0)
	m.allMetrics = make([]*me.Metric, 0)
	m.FileStoragePath = fileStoragePath
	return &m
}

func (m *MemStorage) UpdateMetric(metric *me.Metric) {
	switch metric.MType {
	case MetricTypeGauge:
		m.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		m.Counter[metric.ID] += *metric.Delta
	}
}

func (m *MemStorage) DeleteMetric(mType string, name string) {
	switch mType {
	case MetricTypeGauge:
		delete(m.Gauge, name)
	case MetricTypeCounter:
		delete(m.Counter, name)
	}
}

func (m *MemStorage) GetMetric(mType string, name string) (*me.Metric, bool) {
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

func (m *MemStorage) GetMetrics() []*me.Metric {
	m.allMetrics = m.allMetrics[:0]
	m.allMetrics = append(m.allMetrics, m.GetGauges()...)
	m.allMetrics = append(m.allMetrics, m.GetCounters()...)
	return m.allMetrics
}

func (m *MemStorage) GetGauges() []*me.Metric {
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

func (m *MemStorage) GetCounters() []*me.Metric {
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

func (m *MemStorage) SaveMetric(metric *me.Metric) error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil && err != io.EOF {
		return err
	}
	defer file.Close()

	var metrics []me.Metric
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if len(data) != 0 {
		if err := json.Unmarshal(data, &metrics); err != nil {
			return err
		}
	}

	found := false
	for index, me := range metrics {
		if me.ID == metric.ID && me.MType == metric.MType {
			if me.MType == MetricTypeCounter {
				metrics[index].Delta = metric.Delta
			}
			if me.MType == MetricTypeGauge {
				metrics[index].Value = metric.Value
			}
			found = true
			break
		}
	}

	if !found {
		metrics = append(metrics, *metric)
	}

	b, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	err = os.WriteFile(m.FileStoragePath, b, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (m *MemStorage) Save() error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	gauge := m.GetGauges()
	counter := m.GetCounters()
	metrics := append(gauge, counter...)

	b, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (m *MemStorage) Load() error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	defer file.Close()

	var metrics []me.Metric
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		m.UpdateMetric(&metric)
	}
	return nil
}
