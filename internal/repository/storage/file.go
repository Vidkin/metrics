package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
	me "github.com/Vidkin/metrics/internal/metric"
)

type FileStorage struct {
	Gauge           map[string]float64
	Counter         map[string]int64
	FileStoragePath string
	GaugeMetrics    []*me.Metric
	CounterMetrics  []*me.Metric
	AllMetrics      []*me.Metric
	mu              sync.RWMutex
}

func (f *FileStorage) UpdateMetric(_ context.Context, metric *me.Metric) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch metric.MType {
	case MetricTypeGauge:
		f.Gauge[metric.ID] = *metric.Value
	case MetricTypeCounter:
		f.Counter[metric.ID] += *metric.Delta
	default:
		return errors.New("unknown metric type")
	}
	return nil
}

func (f *FileStorage) UpdateMetrics(_ context.Context, metrics *[]me.Metric) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, metric := range *metrics {
		switch metric.MType {
		case MetricTypeGauge:
			f.Gauge[metric.ID] = *metric.Value
		case MetricTypeCounter:
			f.Counter[metric.ID] += *metric.Delta
		default:
			return errors.New("unknown metric type")
		}
	}
	return nil
}

func (f *FileStorage) DeleteMetric(_ context.Context, mType string, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch mType {
	case MetricTypeGauge:
		delete(f.Gauge, name)
	case MetricTypeCounter:
		delete(f.Counter, name)
	default:
		return errors.New("unknown metric type")
	}
	return nil
}

func (f *FileStorage) GetMetric(_ context.Context, mType string, name string) (*me.Metric, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

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
	default:
		return nil, errors.New("unknown metric type")
	}
	return &metric, nil
}

func (f *FileStorage) GetMetrics(ctx context.Context) ([]*me.Metric, error) {
	f.AllMetrics = f.AllMetrics[:0]
	if _, err := f.GetGauges(ctx); err != nil {
		return nil, err
	}
	if _, err := f.GetCounters(ctx); err != nil {
		return nil, err
	}
	f.AllMetrics = append(f.AllMetrics, f.GaugeMetrics...)
	f.AllMetrics = append(f.AllMetrics, f.CounterMetrics...)
	return f.AllMetrics, nil
}

func (f *FileStorage) GetGauges(_ context.Context) ([]*me.Metric, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.GaugeMetrics = f.GaugeMetrics[:0]
	for k, v := range f.Gauge {
		f.GaugeMetrics = append(f.GaugeMetrics, &me.Metric{
			ID:    k,
			Value: &v,
			MType: MetricTypeGauge,
		})
	}
	return f.GaugeMetrics, nil
}

func (f *FileStorage) GetCounters(_ context.Context) ([]*me.Metric, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.CounterMetrics = f.CounterMetrics[:0]
	for k, v := range f.Counter {
		f.CounterMetrics = append(f.CounterMetrics, &me.Metric{
			ID:    k,
			Delta: &v,
			MType: MetricTypeCounter,
		})
	}
	return f.CounterMetrics, nil
}

func (f *FileStorage) Dump(metric *me.Metric) error {
	file, err := os.OpenFile(f.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil && err != io.EOF {
		logger.Log.Info("error open file", zap.Error(err))
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		logger.Log.Info("error read file", zap.Error(err))
		return err
	}

	var metrics []me.Metric
	if len(data) != 0 {
		if err = json.Unmarshal(data, &metrics); err != nil {
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
		logger.Log.Info("error marshal metrics", zap.Error(err))
		return err
	}

	err = os.WriteFile(f.FileStoragePath, b, 0666)
	if err != nil {
		logger.Log.Info("error write file", zap.Error(err))
		return err
	}
	return nil
}

func (f *FileStorage) FullDump() error {
	file, err := os.OpenFile(f.FileStoragePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logger.Log.Info("error open file", zap.Error(err))
		return err
	}
	defer file.Close()

	gauge, err := f.GetGauges(context.TODO())
	if err != nil {
		logger.Log.Info("error get gauges", zap.Error(err))
		return err
	}
	counter, err := f.GetCounters(context.TODO())
	if err != nil {
		logger.Log.Info("error get counters", zap.Error(err))
		return err
	}
	metrics := append(gauge, counter...)

	b, err := json.Marshal(metrics)
	if err != nil {
		logger.Log.Info("error marshal metrics", zap.Error(err))
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		logger.Log.Info("error write file", zap.Error(err))
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
		logger.Log.Info("error open file", zap.Error(err))
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		logger.Log.Info("error read file", zap.Error(err))
		return err
	}
	var metrics []me.Metric
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
