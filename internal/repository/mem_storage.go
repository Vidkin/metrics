package repository

import (
	"encoding/json"
	"io"
	"os"
)

type MemStorage struct {
	Gauge           map[string]float64
	Counter         map[string]int64
	FileStoragePath string
}

func NewMemoryStorage(fileStoragePath string) *MemStorage {
	var m MemStorage
	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)
	m.FileStoragePath = fileStoragePath
	return &m
}

func (m *MemStorage) UpdateGauge(key string, value float64) {
	m.Gauge[key] = value
}

func (m *MemStorage) UpdateCounter(key string, value int64) {
	m.Counter[key] += value
}

func (m *MemStorage) GetGauges() map[string]float64 {
	return m.Gauge
}

func (m *MemStorage) GetCounters() map[string]int64 {
	return m.Counter
}

func (m *MemStorage) GetGauge(metricName string) (value float64, ok bool) {
	v, ok := m.Gauge[metricName]
	return v, ok
}

func (m *MemStorage) GetCounter(metricName string) (value int64, ok bool) {
	v, ok := m.Counter[metricName]
	return v, ok
}

func (m *MemStorage) Save() error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)

	metrics := make(map[string]map[string]interface{})
	metrics["gauge"] = make(map[string]interface{})
	metrics["counter"] = make(map[string]interface{})

	for k, v := range m.Gauge {
		metrics["gauge"][k] = v
	}
	for k, v := range m.Counter {
		metrics["counter"][k] = v
	}
	return enc.Encode(metrics)
}

func (m *MemStorage) SaveCounter(metricName string, metricValue int64) error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil && err != io.EOF {
		return err
	}
	defer file.Close()

	metrics := make(map[string]map[string]interface{})
	metrics["counter"] = make(map[string]interface{})

	if err != nil && err != io.EOF {
		dec := json.NewDecoder(file)
		if err := dec.Decode(&metrics); err != nil {
			return err
		}
	}

	metrics["counter"][metricName] = metricValue
	enc := json.NewEncoder(file)
	return enc.Encode(metrics)
}

func (m *MemStorage) SaveGauge(metricName string, metricValue float64) error {
	file, err := os.OpenFile(m.FileStoragePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil && err != io.EOF {
		return err
	}
	defer file.Close()

	metrics := make(map[string]map[string]interface{})
	metrics["gauge"] = make(map[string]interface{})

	if err != nil && err != io.EOF {
		dec := json.NewDecoder(file)
		if err := dec.Decode(&metrics); err != nil {
			return err
		}
	}

	metrics["gauge"][metricName] = metricValue
	enc := json.NewEncoder(file)
	return enc.Encode(metrics)
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

	dec := json.NewDecoder(file)
	metrics := make(map[string]map[string]interface{})
	metrics["counter"] = make(map[string]interface{})
	metrics["gauge"] = make(map[string]interface{})

	if err := dec.Decode(&metrics); err != nil {
		return err
	}

	for k, v := range metrics["counter"] {
		m.UpdateCounter(k, int64(v.(float64)))
	}
	for k, v := range metrics["gauge"] {
		m.UpdateGauge(k, v.(float64))
	}
	return nil
}
