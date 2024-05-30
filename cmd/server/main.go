package main

import (
	"fmt"
	"github.com/Vidkin/metrics/internal"
	"net/http"
	"strconv"
)

var memStorage = internal.NewMemStorage()

func requestHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests allowed!", http.StatusMethodNotAllowed)
		return
	}

	metricType := req.PathValue(internal.ParamMetricType)
	metricName := req.PathValue(internal.ParamMetricName)
	metricValue := req.PathValue(internal.ParamMetricValue)

	switch metricType {
	case internal.MetricTypeGauge:
		if value, err := strconv.ParseFloat(metricValue, 64); err != nil {
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			memStorage.UpdateGauge(metricName, value)
		}
	case internal.MetricTypeCounter:
		if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			http.Error(res, "Bad metric value!", http.StatusBadRequest)
		} else {
			memStorage.UpdateCounter(metricName, value)
		}
	default:
		http.Error(res, "Bad metric type!", http.StatusBadRequest)
	}

	res.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	pattern := fmt.Sprintf("/update/{%s}/{%s}/{%s}", internal.ParamMetricType, internal.ParamMetricName, internal.ParamMetricValue)
	mux.HandleFunc(pattern, requestHandler)
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}
}
