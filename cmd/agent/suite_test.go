package main

import (
	"fmt"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMetrics(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongUrl bool
		repository     repository.Repository
	}{
		{
			name:           "test send ok",
			sendToWrongUrl: false,
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{"param1": 45.21, "param2": 12},
				Counter: map[string]int64{"param2": 1},
			},
		},
		{
			name:           "test send to wrong url",
			sendToWrongUrl: true,
			repository: &storage.MemStorage{
				Gauge:   map[string]float64{"param1": 45.21, "param2": 12},
				Counter: map[string]int64{"param2": 1},
			},
		},
	}

	mux := http.NewServeMux()
	serverRepository := storage.New()
	pattern := fmt.Sprintf("/update/{%s}/{%s}/{%s}", internal.ParamMetricType, internal.ParamMetricName, internal.ParamMetricValue)
	mux.HandleFunc(pattern, handlers.MetricsHandler(serverRepository))
	mockServer := httptest.NewServer(mux)
	defer mockServer.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongUrl {
				SendMetrics(mockServer.URL+"/wrong_url/", test.repository)
				assert.NotEqual(t, test.repository, serverRepository)
			} else {
				SendMetrics(mockServer.URL+"/update/", test.repository)
				assert.Equal(t, test.repository, serverRepository)
			}
		})
	}
}

func TestSendMetric(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongUrl bool
		metricType     string
		metricName     string
		metricValue    string
		statusCode     int
	}{
		{
			name:           "test send ok",
			sendToWrongUrl: false,
			metricType:     internal.MetricTypeGauge,
			metricName:     "test",
			metricValue:    "25",
			statusCode:     http.StatusOK,
		},
		{
			name:           "test send bad metric type",
			sendToWrongUrl: false,
			metricType:     "bad_metric_type",
			metricName:     "test",
			metricValue:    "25",
			statusCode:     http.StatusBadRequest,
		},
		{
			name:           "test send empty metric name",
			sendToWrongUrl: false,
			metricType:     "gauge",
			metricName:     "",
			metricValue:    "25",
			statusCode:     http.StatusNotFound,
		},
		{
			name:           "test send bad metric value",
			sendToWrongUrl: false,
			metricType:     "gauge",
			metricName:     "test",
			metricValue:    "bad_value",
			statusCode:     http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			serverRepository := storage.New()
			pattern := fmt.Sprintf("/update/{%s}/{%s}/{%s}", internal.ParamMetricType, internal.ParamMetricName, internal.ParamMetricValue)
			mux.HandleFunc(pattern, handlers.MetricsHandler(serverRepository))
			mockServer := httptest.NewServer(mux)
			defer mockServer.Close()

			if test.sendToWrongUrl {
				_, err := SendMetric(mockServer.URL+"/wrong_url/", test.metricType, test.metricName, test.metricValue)
				assert.NotNil(t, err)
			} else {
				respCode, err := SendMetric(mockServer.URL+"/update/", test.metricType, test.metricName, test.metricValue)
				assert.Equal(t, test.statusCode, respCode)
				assert.Nil(t, err)
			}
		})
	}
}
