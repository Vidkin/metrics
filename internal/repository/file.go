package repository

import (
	"context"
	"encoding/json"
	"errors"
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

func (f *FileStorage) UpdateMetric(_ context.Context, metric *me.Metric) error {
	switch metric.MType {
	case MetricTypeGauge:
		f.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		f.Counter[metric.ID] = *metric.Delta
	}
	return nil
}

func (f *FileStorage) UpdateMetrics(_ context.Context, metrics *[]me.Metric) error {
	for _, metric := range *metrics {
		switch metric.MType {
		case MetricTypeGauge:
			f.Gauge[metric.ID] = *metric.Value
		case MetricTypeCounter:
			f.Counter[metric.ID] = *metric.Delta
		}
	}
	return nil
}

func (f *FileStorage) DeleteMetric(_ context.Context, mType string, name string) error {
	switch mType {
	case MetricTypeGauge:
		delete(f.Gauge, name)
	case MetricTypeCounter:
		delete(f.Counter, name)
	}
	return nil
}

func (f *FileStorage) GetMetric(_ context.Context, mType string, name string) (*me.Metric, error) {
	var metric me.Metric
	switch mType {
	case MetricTypeGauge:
		v, ok := f.Gauge[name]
		if !ok {
			return nil, errors.New("metric not found")
		}
		metric.ID = name
		metric.MType = MetricTypeGauge
		metric.Value = &v
	case MetricTypeCounter:
		v, ok := f.Counter[name]
		if !ok {
			return nil, errors.New("metric not found")
		}
		metric.ID = name
		metric.MType = MetricTypeCounter
		metric.Delta = &v
	}
	return &metric, nil
}

func (f *FileStorage) GetMetrics(ctx context.Context) ([]*me.Metric, error) {
	f.allMetrics = f.allMetrics[:0]
	if _, err := f.GetGauges(ctx); err != nil {
		return nil, err
	}
	if _, err := f.GetCounters(ctx); err != nil {
		return nil, err
	}
	f.allMetrics = append(f.allMetrics, f.gaugeMetrics...)
	f.allMetrics = append(f.allMetrics, f.counterMetrics...)
	return f.allMetrics, nil
}

func (f *FileStorage) GetGauges(_ context.Context) ([]*me.Metric, error) {
	f.gaugeMetrics = f.gaugeMetrics[:0]
	for k, v := range f.Gauge {
		f.gaugeMetrics = append(f.gaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return f.gaugeMetrics, nil
}

func (f *FileStorage) GetCounters(_ context.Context) ([]*me.Metric, error) {
	f.counterMetrics = f.counterMetrics[:0]
	for k, v := range f.Counter {
		f.counterMetrics = append(f.counterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return f.counterMetrics, nil
}

func (f *FileStorage) SaveMetric(metric *me.Metric) error {
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
	for index, met := range metrics {
		if met.ID == metric.ID && met.MType == metric.MType {
			if met.MType == MetricTypeCounter {
				metrics[index].Delta = metric.Delta
			}
			if met.MType == MetricTypeGauge {
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

	gauge, err := f.GetGauges(ctx)
	if err != nil {
		return err
	}
	counter, err := f.GetCounters(ctx)
	if err != nil {
		return err
	}
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
		if err := f.UpdateMetric(ctx, &metric); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileStorage) Ping(_ context.Context) error {
	return nil
}

func (f *FileStorage) Close() error {
	return nil
}
