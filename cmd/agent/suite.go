package main

import (
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/repository"
	"github.com/go-resty/resty/v2"
	"math/rand/v2"
	"runtime"
	"strconv"
	"time"
)

func collectMetrics(repository repository.Repository, memStats *runtime.MemStats) {
	repository.UpdateGauge(internal.GaugeMetricAlloc, float64(memStats.Alloc))
	repository.UpdateGauge(internal.GaugeMetricBuckHashSys, float64(memStats.BuckHashSys))
	repository.UpdateGauge(internal.GaugeMetricFrees, float64(memStats.Frees))
	repository.UpdateGauge(internal.GaugeMetricGCCPUFraction, float64(memStats.GCCPUFraction))
	repository.UpdateGauge(internal.GaugeMetricGCSys, float64(memStats.GCSys))
	repository.UpdateGauge(internal.GaugeMetricHeapAlloc, float64(memStats.HeapAlloc))
	repository.UpdateGauge(internal.GaugeMetricHeapIdle, float64(memStats.HeapIdle))
	repository.UpdateGauge(internal.GaugeMetricHeapInuse, float64(memStats.HeapInuse))
	repository.UpdateGauge(internal.GaugeMetricHeapObjects, float64(memStats.HeapObjects))
	repository.UpdateGauge(internal.GaugeMetricHeapReleased, float64(memStats.HeapReleased))
	repository.UpdateGauge(internal.GaugeMetricHeapSys, float64(memStats.HeapSys))
	repository.UpdateGauge(internal.GaugeMetricLastGC, float64(memStats.LastGC))
	repository.UpdateGauge(internal.GaugeMetricLookups, float64(memStats.Lookups))
	repository.UpdateGauge(internal.GaugeMetricMCacheInuse, float64(memStats.MCacheInuse))
	repository.UpdateGauge(internal.GaugeMetricMCacheSys, float64(memStats.MCacheSys))
	repository.UpdateGauge(internal.GaugeMetricMSpanInuse, float64(memStats.MSpanInuse))
	repository.UpdateGauge(internal.GaugeMetricMSpanSys, float64(memStats.MSpanSys))
	repository.UpdateGauge(internal.GaugeMetricMallocs, float64(memStats.Mallocs))
	repository.UpdateGauge(internal.GaugeMetricNextGC, float64(memStats.NextGC))
	repository.UpdateGauge(internal.GaugeMetricNumForcedGC, float64(memStats.NumForcedGC))
	repository.UpdateGauge(internal.GaugeMetricNumGC, float64(memStats.NumGC))
	repository.UpdateGauge(internal.GaugeMetricOtherSys, float64(memStats.OtherSys))
	repository.UpdateGauge(internal.GaugeMetricPauseTotalNs, float64(memStats.PauseTotalNs))
	repository.UpdateGauge(internal.GaugeMetricStackInuse, float64(memStats.StackInuse))
	repository.UpdateGauge(internal.GaugeMetricStackSys, float64(memStats.StackSys))
	repository.UpdateGauge(internal.GaugeMetricSys, float64(memStats.Sys))
	repository.UpdateGauge(internal.GaugeMetricTotalAlloc, float64(memStats.TotalAlloc))
	repository.UpdateGauge(internal.GaugeMetricRandomValue, rand.Float64())
	repository.UpdateCounter(internal.CounterMetricPollCount, 1)
}

func SendMetric(client *resty.Client, url string, metricType string, metricName string, metricValue string) (int, error) {
	url += metricType + "/" + metricName + "/" + metricValue

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain; charset=utf-8").
		Post(url)

	if err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}

func SendMetrics(client *resty.Client, url string, repository repository.Repository) {
	for metricName, metricValue := range repository.GetGauges() {
		valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
		SendMetric(client, url, internal.MetricTypeGauge, metricName, valueAsString)
	}
	for metricName, metricValue := range repository.GetCounters() {
		valueAsString := strconv.FormatInt(metricValue, 10)
		SendMetric(client, url, internal.MetricTypeCounter, metricName, valueAsString)
	}
}

func Poll(client *resty.Client, repository repository.Repository, memStats *runtime.MemStats) {
	startTime := time.Now()
	url := "http://" + ServerAddr.String() + "/update/"

	for {
		currentTime := time.Now()
		runtime.ReadMemStats(memStats)
		collectMetrics(repository, memStats)

		if currentTime.Sub(startTime).Seconds() >= float64(ReportInterval) {
			startTime = currentTime
			SendMetrics(client, url, repository)
		}
		time.Sleep(time.Duration(PollInterval) * time.Second)
	}
}
