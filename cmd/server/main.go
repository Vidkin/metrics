package main

import (
	"net/http"
	"strconv"
)

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	var m MemStorage
	m.gauge = make(map[string]float64)
	m.counter = make(map[string]int64)
	return &m
}

func (m *MemStorage) GetGauge() map[string]float64 {
	return m.gauge
}

func (m *MemStorage) GetCounter() map[string]int64 {
	return m.counter
}

var memStorage = NewMemStorage()

const (
	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

func requestHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests allowed!", http.StatusMethodNotAllowed)
		return
	}

	metricType := req.PathValue("metricType")
	metricName := req.PathValue("metricName")
	metricValue := req.PathValue("metricValue")

	switch metricType {
	case MetricTypeGauge:
		if s, err := strconv.ParseFloat(metricValue, 64); err != nil {
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			memStorage.GetGauge()[metricName] = s
		}
	case MetricTypeCounter:
		if s, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			memStorage.GetCounter()[metricName] += s
		}
	default:
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
	}

	res.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/{metricType}/{metricName}/{metricValue}", requestHandler)
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}
}
