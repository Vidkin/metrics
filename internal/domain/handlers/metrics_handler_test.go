package handlers

import (
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
	}
	tests := []struct {
		name        string
		repository  repository.Repository
		metricType  string
		metricName  string
		metricValue string
		want        want
	}{
		{
			name:        "test update gauge status ok",
			metricType:  internal.MetricTypeGauge,
			metricName:  "param1",
			metricValue: "17.340",
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
		{
			name:        "test update counter status ok",
			metricType:  internal.MetricTypeCounter,
			metricName:  "param1",
			metricValue: "12",
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusOK,
			},
		},
		{
			name:        "test update counter status bad request",
			metricType:  internal.MetricTypeCounter,
			metricName:  "param1",
			metricValue: "test",
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusBadRequest,
			},
		},
		{
			name:        "test update gauge status bad request",
			metricType:  internal.MetricTypeCounter,
			metricName:  "param1",
			metricValue: "test",
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusBadRequest,
			},
		},
		{
			name:        "test method not allowed",
			metricType:  internal.MetricTypeCounter,
			metricName:  "param1",
			metricValue: "test",
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusMethodNotAllowed,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpMethod := http.MethodPost
			if test.want.statusCode == http.StatusMethodNotAllowed {
				httpMethod = http.MethodGet
			}
			request := httptest.NewRequest(httpMethod, "http://localhost:8080/update", nil)
			request.SetPathValue(internal.ParamMetricType, test.metricType)
			request.SetPathValue(internal.ParamMetricName, test.metricName)
			request.SetPathValue(internal.ParamMetricValue, test.metricValue)

			// создаём новый Recorder
			w := httptest.NewRecorder()

			metricsHandler := MetricsHandler(test.repository)
			metricsHandler(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if res.StatusCode == http.StatusOK {
				if test.metricType == internal.MetricTypeGauge {
					metricValue, _ := strconv.ParseFloat(test.metricValue, 64)
					assert.Equal(t, metricValue, test.repository.GetGauges()[test.metricName])
				}
				if test.metricType == internal.MetricTypeCounter {
					metricValue, _ := strconv.ParseInt(test.metricValue, 10, 64)
					assert.Equal(t, metricValue, test.repository.GetCounters()[test.metricName])
				}
			}
		})
	}
}
