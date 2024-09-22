package router

import (
	"bytes"
	"fmt"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
)

var (
	serverRepository = NewMemoryStorage()
	chiRouter        = chi.NewRouter()
	serverConfig     = config.ServerConfig{StoreInterval: 300}
	metricRouter     = NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts               = httptest.NewServer(metricRouter.Router)
)

func ExampleMetricRouter_RootHandler() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println(string(respBody))
	// Output:
	// testGauge = 1.25
	// testCounter = 1
}

func ExampleMetricRouter_GetMetricValueHandler() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/value/gauge/testGauge", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println(string(respBody))
	// Output:
	// 1.25
}

func ExampleMetricRouter_GetMetricValueHandlerJSON() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	requestBody := `{
		"id": "testGauge",
		"type": "gauge"
	}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/value", bytes.NewBuffer([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println(string(respBody))
	// Output:
	// {"id":"testGauge","type":"gauge","value":1.25}
}

func ExampleMetricRouter_UpdateMetricHandler() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update/gauge/testGauge/12.43", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

func ExampleMetricRouter_UpdateMetricHandlerJSON() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	requestBody := `{
		"id": "testGauge",
		"type": "gauge",
		"value": 13.22
	}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/update", bytes.NewBuffer([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println(string(respBody))
	// Output:
	// {"id":"testGauge","type":"gauge","value":13.22}
}

func ExampleMetricRouter_UpdateMetricsHandlerJSON() {
	clear(serverRepository.Gauge)
	clear(serverRepository.Counter)

	serverRepository.Gauge["testGauge"] = 1.25
	serverRepository.Counter["testCounter"] = 1

	requestBody := `[{
		"id": "testGauge",
		"type": "gauge",
		"value": 13.22
		},{
		"id": "testCounter",
		"type": "counter",
		"delta": 65
	}]`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/updates", bytes.NewBuffer([]byte(requestBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("Content-Encoding", "")

	resp, _ := ts.Client().Do(req)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println(string(respBody))
	// Output:
	// [{"id":"testGauge","type":"gauge","value":13.22},{"id":"testCounter","type":"counter","delta":66}]
}
