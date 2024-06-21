package handlers

import (
	"bytes"
	"github.com/Vidkin/metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func testJSONRequest(t *testing.T, ts *httptest.Server, method,
	path string, json string, contentType string) (*http.Response, string) {
	jsonBytes := []byte(json)
	req, err := http.NewRequest(method, ts.URL+path, bytes.NewBuffer(jsonBytes))
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)
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

	serverRepository := repository.New()
	metricRouter := NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, http.MethodPost, test.url)
			defer resp.Body.Close()
			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

func TestGetMetricValueHandler(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		value       string
	}

	var tests = []struct {
		name       string
		url        string
		want       want
		repository Repository
	}{
		{
			name: "test get gauge metric ok",
			url:  "/value/gauge/param1",
			repository: &repository.MemStorage{
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
			repository: &repository.MemStorage{
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
			repository: &repository.MemStorage{
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
			repository: &repository.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	serverRepository := repository.New()
	metricRouter := NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			for k, v := range test.repository.GetGauges() {
				serverRepository.UpdateGauge(k, v)
			}
			for k, v := range test.repository.GetCounters() {
				serverRepository.UpdateCounter(k, v)
			}
			resp, value := testRequest(t, ts, http.MethodGet, test.url)
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
		statusCode  int
		value       string
	}

	var tests = []struct {
		name       string
		want       want
		repository Repository
	}{
		{
			name: "test get all known metrics ok",
			repository: &repository.MemStorage{
				Gauge:   map[string]float64{"param1": 17.34},
				Counter: map[string]int64{"param2": 2},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "param1 = 17.34\nparam2 = 2\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "test empty metrics repository",
			repository: &repository.MemStorage{
				Gauge:   map[string]float64{},
				Counter: map[string]int64{},
			},
			want: want{
				statusCode:  http.StatusOK,
				value:       "",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	serverRepository := repository.New()
	metricRouter := NewMetricRouter(serverRepository)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			for k, v := range test.repository.GetGauges() {
				serverRepository.UpdateGauge(k, v)
			}
			for k, v := range test.repository.GetCounters() {
				serverRepository.UpdateCounter(k, v)
			}
			resp, value := testRequest(t, ts, http.MethodGet, "")
			defer resp.Body.Close()

			assert.Equal(t, test.want.statusCode, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			if test.want.statusCode != http.StatusNotFound {
				assert.Equal(t, test.want.value, value)
			}
		})
	}
}

func TestUpdateMetricHandlerJSON(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		respBody    string
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
				respBody:    `[{"id":"test","type":"gauge","value":13.5}]`,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"gauge","value":13.5}]`,
		},
		{
			name: "test update counter status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `[{"id":"test","type":"counter","delta":13}]`,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"counter","delta":13}]`,
		},
		{
			name: "test update two metrics status ok",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `[{"id":"test","type":"gauge","value":13.5},{"id":"test2","type":"counter","delta":13}]`,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"gauge","value":13.5},{"id":"test2","type":"counter","delta":13}]`,
		},
		{
			name: "test bad content-type",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
			contentType: "text/plain",
			json:        `[{"id":"test","type":"counter","delta":13}]`,
		},
		{
			name: "test update with empty value",
			want: want{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				respBody:    `[]`,
			},
			contentType: "application/json",
			json:        `[{"id":"test","type":"counter"}]`,
		},
		{
			name: "test bad request body",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
			contentType: "application/json",
			json:        ``,
		},
	}

	serverRepository := repository.New()
	metricRouter := NewMetricRouter(serverRepository)
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
