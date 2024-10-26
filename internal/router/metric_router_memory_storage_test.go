package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/repository/storage"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string, acceptEncoding bool) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	if acceptEncoding == true {
		req.Header.Set("Accept-Encoding", "gzip")
	} else {
		req.Header.Set("Accept-Encoding", "")
	}
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if acceptEncoding {
		dec, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		respBody, err := io.ReadAll(dec)
		require.NoError(t, err)
		return resp, string(respBody)
	} else {
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp, string(respBody)
	}
}

func testJSONRequest(t *testing.T, ts *httptest.Server, method,
	path string, json string, contentType string) (*http.Response, string) {
	jsonBytes := []byte(json)
	req, err := http.NewRequest(method, ts.URL+path, bytes.NewBuffer(jsonBytes))
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept-Encoding", "")

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestUpdateMetricHandler(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
	}

	var tests = []struct {
		name string
		url  string
		want want
	}{
		{
			name: "test update gauge status ok",
			url:  "/update/gauge/param1/17.340",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test update counter status ok",
			url:  "/update/counter/param1/12",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test update counter status bad value",
			url:  "/update/counter/param1/testBadRequest",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test update gauge status bad value",
			url:  "/update/gauge/param1/testBadRequest",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test bad metric type",
			url:  "/update/badMetricType/param1/testMethodNotAllowed",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test without metric name",
			url:  "/update/gauge/17",
			want: want{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, http.MethodPost, test.url, false)
			defer resp.Body.Close()
			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

func TestGetMetricValueHandler(t *testing.T) {
	type want struct {
		contentType string
		value       string
		statusCode  int
	}

	var tests = []struct {
		repository Repository
		name       string
		url        string
		want       want
	}{
		{
			name: "test get gauge metric ok",
			url:  "/value/gauge/param1",
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"param1": 17.34},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "17.34",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test get counter metric ok",
			url:  "/value/counter/param1",
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{"param1": 12},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "12",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test get unknown metric",
			url:  "/value/counter/param1",
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test get metric without name",
			url:  "/value/counter/",
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			ctx := context.TODO()
			metrics, _ := test.repository.GetMetrics(ctx)
			for _, metric := range metrics {
				serverRepository.UpdateMetric(ctx, metric)
			}
			resp, value := testRequest(t, ts, http.MethodGet, test.url, false)
			defer resp.Body.Close()

			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			if test.want.statusCode != http.StatusNotFound {
				assert.Equal(t, test.want.value, value)
			}
		})
	}
}

func TestRootHandler(t *testing.T) {
	type want struct {
		contentType string
		value       string
		statusCode  int
	}

	var tests = []struct {
		repository     Repository
		name           string
		want           want
		acceptEncoding bool
	}{
		{
			name:           "test get all known metrics with encoding",
			acceptEncoding: false,
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"param1": 17.34},
				Counter: map[string]int64{"param2": 2},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "param1 = 17.34\nparam2 = 2\n",
				contentType: "text/html",
			},
		},
		{
			name:           "test get all known metrics without encoding",
			acceptEncoding: false,
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"param1": 17.34},
				Counter: map[string]int64{"param2": 2},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "param1 = 17.34\nparam2 = 2\n",
				contentType: "text/html",
			},
		},
		{
			name: "test empty metrics repository",
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "",
				contentType: "text/html",
			},
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			ctx := context.TODO()
			metrics, _ := test.repository.GetMetrics(ctx)
			for _, metric := range metrics {
				serverRepository.UpdateMetric(ctx, metric)
			}

			resp, value := testRequest(t, ts, http.MethodGet, "", test.acceptEncoding)
			defer resp.Body.Close()

			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, test.want.value, value)
		})
	}
}

func TestUpdateMetricHandlerJSON(t *testing.T) {
	type want struct {
		contentType string
		respBody    string
		statusCode  int
	}

	var tests = []struct {
		name        string
		json        string
		contentType string
		want        want
	}{
		{
			name: "test update gauge status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `{"id":"test","type":"gauge","value":13.5}`,
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"gauge","value":13.5}`,
		},
		{
			name: "test update counter status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `{"id":"test","type":"counter","delta":13}`,
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"counter","delta":13}`,
		},
		{
			name: "test update two metrics status ok",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"gauge","value":13.5},{"id":"test2","type":"counter","delta":13}]`,
		},
		{
			name: "test bad content-type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "text/plain",
			json:        `{"id":"test","type":"counter","delta":13}`,
		},
		{
			name: "test update with empty value",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"counter"}`,
		},
		{
			name: "test bad metric type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"badType","delta":13}`,
		},
		{
			name: "test bad request body",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        ``,
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			resp, respBody := testJSONRequest(t, ts, http.MethodPost, "/update", test.json, test.contentType)
			defer resp.Body.Close()
			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
				assert.JSONEq(t, test.want.respBody, respBody)
			}
		})
	}
}

func TestUpdateMetricsHandlerJSON(t *testing.T) {
	type want struct {
		contentType string
		respBody    string
		statusCode  int
	}

	var tests = []struct {
		name        string
		json        string
		contentType string
		want        want
	}{
		{
			name: "test update metrics status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `[{"id":"test","type":"gauge","value":13.5},{"id":"test1","type":"counter","delta":13}]`,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"gauge","value":13.5},{"id":"test1","type":"counter","delta":13}]`,
		},
		{
			name: "test bad content-type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "text/plain",
			json:        `[{"id":"test","type":"counter","delta":13}]`,
		},
		{
			name: "test update with empty value",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"counter"}]`,
		},
		{
			name: "test bad metric type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"badType","delta":13}]`,
		},
		{
			name: "test bad request body",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			contentType: "application/json",
			json:        ``,
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			resp, respBody := testJSONRequest(t, ts, http.MethodPost, "/updates", test.json, test.contentType)
			defer resp.Body.Close()
			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
				assert.JSONEq(t, test.want.respBody, respBody)
			}
		})
	}
}

func TestGetMetricValueHandlerJSON(t *testing.T) {
	type want struct {
		contentType string
		respBody    string
		statusCode  int
	}

	var tests = []struct {
		repository  Repository
		name        string
		json        string
		contentType string
		want        want
	}{
		{
			name: "test get counter metric status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `{"id":"test","type":"counter","delta":12}`,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{"test": 12},
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"counter"}`,
		},
		{
			name: "test get gauge metric status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `{"id":"test","type":"gauge","value":12.5}`,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"test": 12.5},
				Counter: map[string]int64{},
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"gauge"}`,
		},
		{
			name: "test bad content-type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"test": 12.5},
				Counter: map[string]int64{},
			},
			contentType: "text/plain",
			json:        `{"id":"test","type":"gauge"}`,
		},
		{
			name: "test bad metric type",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"test": 12.5},
				Counter: map[string]int64{},
			},
			contentType: "application/json",
			json:        `{"id":"test","type":"badType"}`,
		},
		{
			name: "test metric not found",
			want: want{
				statusCode: http.StatusNotFound,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"test": 12.5},
				Counter: map[string]int64{},
			},
			contentType: "application/json",
			json:        `{"id":"unknownMetric","type":"gauge"}`,
		},
		{
			name: "test bad request body",
			want: want{
				statusCode: http.StatusBadRequest,
			},
			repository: &storage.FileStorage{
				Gauge:   map[string]float64{"test": 12.5},
				Counter: map[string]int64{},
			},
			contentType: "application/json",
			json:        ``,
		},
	}

	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			ctx := context.TODO()
			metrics, _ := test.repository.GetMetrics(ctx)
			for _, metric := range metrics {
				serverRepository.UpdateMetric(ctx, metric)
			}
			resp, respBody := testJSONRequest(t, ts, http.MethodPost, "/value", test.json, test.contentType)
			defer resp.Body.Close()
			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
				assert.JSONEq(t, test.want.respBody, respBody)
			}
		})
	}
}

func TestTrustedSubnet(t *testing.T) {
	requestBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	// ожидаемое содержимое тела ответа при успешном запросе
	successBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	t.Run("good remote ip", func(t *testing.T) {
		serverRepository := NewMemoryStorage()
		chiRouter := chi.NewRouter()
		serverConfig := config.ServerConfig{StoreInterval: 300, TrustedSubnet: "192.168.0.1/24"}
		metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
		ts := httptest.NewServer(metricRouter.Router)
		defer ts.Close()

		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "")
		r.Header.Set("X-Real-IP", "192.168.0.222")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.JSONEq(t, successBody, string(b))
	})

	t.Run("bad cidr", func(t *testing.T) {
		serverRepository := NewMemoryStorage()
		chiRouter := chi.NewRouter()
		serverConfig := config.ServerConfig{StoreInterval: 300, TrustedSubnet: "errorCidr"}
		metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
		ts := httptest.NewServer(metricRouter.Router)
		defer ts.Close()

		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "")
		r.Header.Set("X-Real-IP", "192.168.0.222")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		defer resp.Body.Close()
	})

	t.Run("bad remote ip", func(t *testing.T) {
		serverRepository := NewMemoryStorage()
		chiRouter := chi.NewRouter()
		serverConfig := config.ServerConfig{StoreInterval: 300, TrustedSubnet: "192.168.0.1/24"}
		metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
		ts := httptest.NewServer(metricRouter.Router)
		defer ts.Close()

		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept-Encoding", "gzip")
		r.Header.Set("X-Real-IP", "192.162.2.1")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		defer resp.Body.Close()
	})

	t.Run("null real ip", func(t *testing.T) {
		serverRepository := NewMemoryStorage()
		chiRouter := chi.NewRouter()
		serverConfig := config.ServerConfig{StoreInterval: 300, TrustedSubnet: "192.168.0.1/24"}
		metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
		ts := httptest.NewServer(metricRouter.Router)
		defer ts.Close()

		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode)

		defer resp.Body.Close()
	})
}

func TestGzipCompression(t *testing.T) {
	serverRepository := NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	requestBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	// ожидаемое содержимое тела ответа при успешном запросе
	successBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.JSONEq(t, successBody, string(b))
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest("POST", ts.URL+"/update", buf)
		r.RequestURI = ""
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		b, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.JSONEq(t, successBody, string(b))
	})
}
