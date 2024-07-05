package repository

import (
	"context"
	"encoding/json"
	me "github.com/Vidkin/metrics/internal/metric"
	"io"
	"os"
)

type FileStorage struct {
	Gauge           map[string]float64
	Counter         map[string]int64
	gaugeMetrics    []*me.Metric
	counterMetrics  []*me.Metric
	allMetrics      []*me.Metric
	FileStoragePath string
}

func NewFileStorage(fileStoragePath string) *FileStorage {
	var f FileStorage
	f.Gauge = make(map[string]float64)
	f.Counter = make(map[string]int64)
	f.gaugeMetrics = make([]*me.Metric, 0)
	f.counterMetrics = make([]*me.Metric, 0)
	f.allMetrics = make([]*me.Metric, 0)
	f.FileStoragePath = fileStoragePath
	return &f
}

func (f *FileStorage) UpdateMetric(metric *me.Metric) {
	switch metric.MType {
	case MetricTypeGauge:
		f.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		f.Counter[metric.ID] += *metric.Delta
	}
}

func (f *FileStorage) DeleteMetric(mType string, name string) {
	switch mType {
	case MetricTypeGauge:
		delete(f.Gauge, name)
	case MetricTypeCounter:
		delete(f.Counter, name)
	}
}

func (f *FileStorage) GetMetric(mType string, name string) (*me.Metric, bool) {
	var metric me.Metric
	switch mType {
	case MetricTypeGauge:
		v, ok := f.Gauge[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeGauge
		metric.Value = &v
	case MetricTypeCounter:
		v, ok := f.Counter[name]
		if !ok {
			return nil, false
		}
		metric.ID = name
		metric.MType = MetricTypeCounter
		metric.Delta = &v
	}
	return &metric, true
}

func (f *FileStorage) GetMetrics() []*me.Metric {
	f.allMetrics = f.allMetrics[:0]
	f.allMetrics = append(f.allMetrics, f.GetGauges()...)
	f.allMetrics = append(f.allMetrics, f.GetCounters()...)
	return f.allMetrics
}

func (f *FileStorage) GetGauges() []*me.Metric {
	f.gaugeMetrics = f.gaugeMetrics[:0]
	for k, v := range f.Gauge {
		f.gaugeMetrics = append(f.gaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return f.gaugeMetrics
}

func (f *FileStorage) GetCounters() []*me.Metric {
	f.counterMetrics = f.counterMetrics[:0]
	for k, v := range f.Counter {
		f.counterMetrics = append(f.counterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return f.counterMetrics
}

func (f *FileStorage) SaveMetric(ctx context.Context, metric *me.Metric) error {
	file, err := os.OpenFile(f.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
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

	err = os.WriteFile(f.FileStoragePath, b, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileStorage) Save(ctx context.Context) error {
	file, err := os.OpenFile(f.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	gauge := f.GetGauges()
	counter := f.GetCounters()
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

func (f *FileStorage) Load(ctx context.Context) error {
	file, err := os.OpenFile(f.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
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
		f.UpdateMetric(&metric)
	}
	return nil
}

func (f *FileStorage) Ping(ctx context.Context) error {
	return nil
}

func (f *FileStorage) Close() error {
	return nil
}
