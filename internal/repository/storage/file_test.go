package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	me "github.com/Vidkin/metrics/internal/metric"
)

func TestFileStorage_UpdateMetric(t *testing.T) {
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

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
			}
			if tt.metricAdd != nil && tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.Error(t, err)
			}
			if tt.metricAdd != nil && tt.metricUpdate != nil && !tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = fileStorage.UpdateMetric(context.TODO(), tt.metricUpdate)
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileStorage_DeleteMetric(t *testing.T) {
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

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)

	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = fileStorage.DeleteMetric(context.TODO(), tt.metricAdd.MType, tt.metricAdd.ID)
				assert.NoError(t, err)
			}
			if tt.metricAdd != nil && tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				err = fileStorage.DeleteMetric(context.TODO(), "unknownMType", tt.metricAdd.ID)
				assert.Error(t, err)
			}
		})
	}
}

func TestFileStorage_GetMetric(t *testing.T) {
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

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				err := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, err)
				metric, errGet := fileStorage.GetMetric(context.TODO(), tt.metricAdd.MType, tt.metricAdd.ID)
				assert.NoError(t, errGet)
				assert.Equal(t, tt.metricAdd, metric)
			}
			if tt.metricAdd != nil && tt.wantErr {
				errUpdate := fileStorage.UpdateMetric(context.TODO(), tt.metricAdd)
				assert.NoError(t, errUpdate)
				_, errGet := fileStorage.GetMetric(context.TODO(), "unknownMType", tt.metricAdd.ID)
				assert.Error(t, errGet)
			}
		})
	}
}

func TestFileStorage_GetMetrics(t *testing.T) {
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

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricAdd != nil && !tt.wantErr {
				for _, metric := range tt.metricAdd {
					err := fileStorage.UpdateMetric(context.TODO(), metric)
					assert.NoError(t, err)
				}
				metrics, err := fileStorage.GetMetrics(context.TODO())
				assert.NoError(t, err)
				assert.Equal(t, len(tt.metricAdd), len(metrics))
				assert.Equal(t, tt.metricAdd, metrics)
			}
		})
	}
}

func TestFileStorage_UpdateMetrics(t *testing.T) {
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

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAdd != nil && !tt.wantErr {
				err := fileStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.NoError(t, err)
			}
			if tt.metricsAdd != nil && tt.wantErr {
				err := fileStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.Error(t, err)
			}
			if tt.metricsAdd != nil && tt.metricsUpdate != nil && !tt.wantErr {
				err := fileStorage.UpdateMetrics(context.TODO(), tt.metricsAdd)
				assert.NoError(t, err)

				err = fileStorage.UpdateMetrics(context.TODO(), tt.metricsUpdate)
				assert.NoError(t, err)

				for _, metric := range *tt.metricsUpdate {
					m, err := fileStorage.GetMetric(context.TODO(), metric.MType, metric.ID)
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

func TestFileStorage_Dump(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		metricsAdd    *[]me.Metric
		metricsUpdate *[]me.Metric
		name          string
		wantErr       bool
		found         bool
	}{
		{
			name: "test dump ok",
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
			name:    "test bad file name",
			wantErr: true,
		},
		{
			name: "test metrics found ok",
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
			wantErr: false,
			found:   true,
		},
	}

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAdd != nil && !tt.wantErr {
				for _, m := range *tt.metricsAdd {
					err := fileStorage.Dump(&m)
					assert.NoError(t, err)
					if tt.found {
						err = fileStorage.Dump(&m)
						assert.NoError(t, err)
					}
				}
			}
			if tt.wantErr == true {
				bak := fileStorage.FileStoragePath
				fileStorage.FileStoragePath = "/badPath//"
				err := fileStorage.Dump(nil)
				assert.Error(t, err)
				fileStorage.FileStoragePath = bak
			}
		})
	}
}

func TestFileStorage_FullDump(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		metricsAdd    *[]me.Metric
		metricsUpdate *[]me.Metric
		name          string
		wantErr       bool
		found         bool
	}{
		{
			name: "test full dump ok",
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
			name:    "test bad file name",
			wantErr: true,
		},
	}

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAdd != nil && !tt.wantErr {
				err := fileStorage.FullDump()
				assert.NoError(t, err)
			}
			if tt.wantErr == true {
				bak := fileStorage.FileStoragePath
				fileStorage.FileStoragePath = "/badPath//"
				err := fileStorage.FullDump()
				assert.Error(t, err)
				fileStorage.FileStoragePath = bak
			}
		})
	}
}

func TestFileStorage_Load(t *testing.T) {
	floatValue := 16.4
	intValue := int64(12)
	tests := []struct {
		metricsAdd    *[]me.Metric
		metricsUpdate *[]me.Metric
		name          string
		wantErr       bool
		found         bool
	}{
		{
			name: "test load ok",
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
			name:    "test bad file name",
			wantErr: true,
		},
	}

	var fileStorage FileStorage
	fileStorage.Gauge = make(map[string]float64)
	fileStorage.Counter = make(map[string]int64)
	fileStorage.GaugeMetrics = make([]*me.Metric, 0)
	fileStorage.CounterMetrics = make([]*me.Metric, 0)
	fileStorage.AllMetrics = make([]*me.Metric, 0)
	fileStorage.FileStoragePath = filepath.Join(os.TempDir(), "metricsTestFile.test")

	defer os.Remove(fileStorage.FileStoragePath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsAdd != nil && !tt.wantErr {
				err := fileStorage.FullDump()
				assert.NoError(t, err)
				err = fileStorage.Load(context.TODO())
				assert.NoError(t, err)
			}
			if tt.wantErr == true {
				bak := fileStorage.FileStoragePath
				fileStorage.FileStoragePath = "/badPath//"
				err := fileStorage.Load(context.TODO())
				assert.Error(t, err)
				fileStorage.FileStoragePath = bak
			}
		})
	}
}
