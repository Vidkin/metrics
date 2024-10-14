package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/router"
)

func BenchmarkCollectAndSendMetrics(b *testing.B) {
	serverRepository := router.NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()
	client.SetDoNotParseResponse(true)
	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	mw := New(memoryStorage, memStats, client, &config.AgentConfig{Key: "", RateLimit: 5})
	var serverURL = ts.URL + "/updates/"

	b.ResetTimer()
	b.Run("poll", func(b *testing.B) {
		var count int64 = 1
		for i := 0; i < 100; i++ {
			chIn := make(chan *metric.Metric, 10)
			go mw.CollectMetrics(chIn, count)
			for w := 1; w <= mw.config.RateLimit; w++ {
				go mw.SendMetrics(chIn, serverURL)
			}
		}
	})
}

func TestSendMetrics(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongURL bool
	}{
		{
			name:           "test send ok",
			sendToWrongURL: false,
		},
		{
			name:           "test send to wrong url",
			sendToWrongURL: true,
		},
	}

	serverRepository := router.NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()
	client.SetDoNotParseResponse(true)
	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	mw := New(memoryStorage, memStats, client, &config.AgentConfig{Key: ""})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			chIn := make(chan *metric.Metric, 10)
			go mw.CollectMetrics(chIn, 10)

			if test.sendToWrongURL {
				mw.SendMetrics(chIn, ts.URL+"/wrong_url/")
				ctx := context.TODO()
				testMetrics, _ := mw.repository.GetMetrics(ctx)
				serverMetrics, _ := serverRepository.GetMetrics(ctx)
				assert.NotEqual(t, testMetrics, serverMetrics)
			} else {
				mw.SendMetrics(chIn, ts.URL+"/updates/")
				ctx := context.TODO()
				testMetrics, _ := mw.repository.GetMetrics(ctx)
				serverMetrics, _ := serverRepository.GetMetrics(ctx)
				assert.ElementsMatch(t, testMetrics, serverMetrics)
			}
		})
	}
}

func TestSendMetric(t *testing.T) {
	var testIntValue int64 = 42
	var testFloatValue = 42.5
	type want struct {
		resp       string
		statusCode int
	}
	tests := []struct {
		name           string
		metric         metric.Metric
		want           want
		sendToWrongURL bool
		statusCode     int
	}{
		{
			name:           "test send counter ok",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: MetricTypeCounter,
				ID:    "test",
				Delta: &testIntValue,
			},
			want: want{
				resp:       `{"type":"counter","id":"test","delta":42}`,
				statusCode: http.StatusOK,
			},
		},
		{
			name:           "test send gauge ok",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: MetricTypeGauge,
				ID:    "test",
				Value: &testFloatValue,
			},
			want: want{
				resp:       `{"type":"gauge","id":"test","value":42.5}`,
				statusCode: http.StatusOK,
			},
		},
		{
			name:           "test send bad metric type",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "test",
				Delta: &testIntValue,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:           "test send empty metric name",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "",
				Delta: &testIntValue,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:           "test send bad metric value",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "test",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	serverRepository := router.NewMemoryStorage()
	client := resty.New()
	client.SetDoNotParseResponse(true)
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	mw := New(nil, nil, client, &config.AgentConfig{Key: ""})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				_, _, err := mw.SendMetric(ts.URL+"/wrong_url/", &test.metric)
				assert.NotNil(t, err)
			} else {
				respCode, respBody, err := mw.SendMetric(ts.URL+"/update/", &test.metric)
				assert.Equal(t, test.want.statusCode, respCode)
				if test.want.statusCode == http.StatusOK {
					assert.JSONEq(t, test.want.resp, respBody)
					assert.Nil(t, err)
				}
			}
		})
	}
}
