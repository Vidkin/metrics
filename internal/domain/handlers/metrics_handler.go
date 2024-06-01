package handlers

import (
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"net/http"
	"strconv"
)

func MetricsHandler(repository repository.Repository) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
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
				repository.UpdateGauge(metricName, value)
			}
		case internal.MetricTypeCounter:
			if value, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
				http.Error(res, "Bad metric value!", http.StatusBadRequest)
			} else {
				repository.UpdateCounter(metricName, value)
			}
		default:
			http.Error(res, "Bad metric type!", http.StatusBadRequest)
		}
		res.WriteHeader(http.StatusOK)
	}
}
