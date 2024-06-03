package handlers

import (
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

func GetMetricValueHandler(repository repository.Repository) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "metricType")
		metricName := chi.URLParam(req, "metricName")

		switch metricType {
		case internal.MetricTypeGauge:
			if metricValue, ok := repository.GetGauge(metricName); ok {
				valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
				res.Write([]byte(valueAsString))
			} else {
				http.Error(res, "Metric not found", http.StatusNotFound)
			}
		case internal.MetricTypeCounter:
			if metricValue, ok := repository.GetCounter(metricName); ok {
				valueAsString := strconv.FormatInt(metricValue, 10)
				res.Write([]byte(valueAsString))
			} else {
				http.Error(res, "Metric not found", http.StatusNotFound)
			}
		default:
			http.Error(res, "Bad metric type!", http.StatusBadRequest)
		}
		res.Header().Set("Content-Type", "plain/text")
		res.WriteHeader(http.StatusOK)
	}
}

func UpdateMetricHandler(repository repository.Repository) http.HandlerFunc {
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
