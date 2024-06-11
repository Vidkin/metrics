package metricworker

import (
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMetrics(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongURL bool
		repository     handlers.Repository
	}{
		{
			name:           "test send ok",
			sendToWrongURL: false,
			repository: &repository.MemStorage{
				Gauge:   map[string]float64{"param1": 45.21, "param2": 12},
				Counter: map[string]int64{"param2": 1},
			},
		},
		{
			name:           "test send to wrong url",
			sendToWrongURL: true,
			repository: &repository.MemStorage{
				Gauge:   map[string]float64{"param1": 45.21, "param2": 12},
				Counter: map[string]int64{"param2": 1},
			},
		},
	}

	serverRepository := repository.New()
	metricRouter := handlers.NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				SendMetrics(client, ts.URL+"/wrong_url/", test.repository)
				assert.NotEqual(t, test.repository, serverRepository)
			} else {
				SendMetrics(client, ts.URL+"/update/", test.repository)
				assert.Equal(t, test.repository, serverRepository)
			}
		})
	}
}

func TestSendMetric(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongURL bool
		metricType     string
		metricName     string
		metricValue    string
		statusCode     int
	}{
		{
			name:           "test send ok",
			sendToWrongURL: false,
			metricType:     MetricTypeGauge,
			metricName:     "test",
			metricValue:    "25",
			statusCode:     http.StatusOK,
		},
		{
			name:           "test send bad metric type",
			sendToWrongURL: false,
			metricType:     "bad_metric_type",
			metricName:     "test",
			metricValue:    "25",
			statusCode:     http.StatusBadRequest,
		},
		{
			name:           "test send empty metric name",
			sendToWrongURL: false,
			metricType:     "gauge",
			metricName:     "",
			metricValue:    "25",
			statusCode:     http.StatusNotFound,
		},
		{
			name:           "test send bad metric value",
			sendToWrongURL: false,
			metricType:     "gauge",
			metricName:     "test",
			metricValue:    "bad_value",
			statusCode:     http.StatusBadRequest,
		},
	}

	serverRepository := repository.New()
	metricRouter := handlers.NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				_, err := SendMetric(client, ts.URL+"/wrong_url/", test.metricType, test.metricName, test.metricValue)
				assert.NotNil(t, err)
			} else {
				respCode, err := SendMetric(client, ts.URL+"/update/", test.metricType, test.metricName, test.metricValue)
				assert.Equal(t, test.statusCode, respCode)
				assert.Nil(t, err)
			}
		})
	}
}
