package metricworker

import (
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/models"
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
	client := resty.New()
	client.SetDoNotParseResponse(true)
	metricRouter := handlers.NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	mw := New(nil, nil, client, nil)

	for _, test := range tests {
		mw.repository = test.repository
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				mw.SendMetrics(ts.URL + "/wrong_url/")
				assert.NotEqual(t, test.repository, serverRepository)
			} else {
				mw.SendMetrics(ts.URL + "/update/")
				assert.Equal(t, test.repository, serverRepository)
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
		sendToWrongURL bool
		metric         models.Metrics
		statusCode     int
		want           want
	}{
		{
			name:           "test send counter ok",
			sendToWrongURL: false,
			metric: models.Metrics{
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
			metric: models.Metrics{
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
			metric: models.Metrics{
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
			metric: models.Metrics{
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
			metric: models.Metrics{
				MType: "badMetricType",
				ID:    "test",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	serverRepository := repository.New()
	client := resty.New()
	client.SetDoNotParseResponse(true)
	metricRouter := handlers.NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	mw := New(nil, nil, client, nil)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				_, _, err := mw.SendMetric(ts.URL+"/wrong_url/", test.metric)
				assert.NotNil(t, err)
			} else {
				respCode, respBody, err := mw.SendMetric(ts.URL+"/update/", test.metric)
				assert.Equal(t, test.want.statusCode, respCode)
				if test.want.statusCode == http.StatusOK {
					assert.JSONEq(t, test.want.resp, respBody)
					assert.Nil(t, err)
				}
			}
		})
	}
}
