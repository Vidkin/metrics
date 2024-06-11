package metric_worker

import (
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/go-resty/resty/v2"
	"math/rand/v2"
	"runtime"
	"strconv"
	"time"
)

const (
	GaugeMetricAlloc         = "Alloc"
	GaugeMetricBuckHashSys   = "BuckHashSys"
	GaugeMetricFrees         = "Frees"
	GaugeMetricGCCPUFraction = "GCCPUFraction"
	GaugeMetricGCSys         = "GCSys"
	GaugeMetricHeapAlloc     = "HeapAlloc"
	GaugeMetricHeapIdle      = "HeapIdle"
	GaugeMetricHeapInuse     = "HeapInuse"
	GaugeMetricHeapObjects   = "HeapObjects"
	GaugeMetricHeapReleased  = "HeapReleased"
	GaugeMetricHeapSys       = "HeapSys"
	GaugeMetricLastGC        = "LastGC"
	GaugeMetricLookups       = "Lookups"
	GaugeMetricMCacheInuse   = "MCacheInuse"
	GaugeMetricMCacheSys     = "MCacheSys"
	GaugeMetricMSpanInuse    = "MSpanInuse"
	GaugeMetricMSpanSys      = "MSpanSys"
	GaugeMetricMallocs       = "Mallocs"
	GaugeMetricNextGC        = "NextGC"
	GaugeMetricNumForcedGC   = "NumForcedGC"
	GaugeMetricNumGC         = "NumGC"
	GaugeMetricOtherSys      = "OtherSys"
	GaugeMetricPauseTotalNs  = "PauseTotalNs"
	GaugeMetricStackInuse    = "StackInuse"
	GaugeMetricStackSys      = "StackSys"
	GaugeMetricSys           = "Sys"
	GaugeMetricTotalAlloc    = "TotalAlloc"
	GaugeMetricRandomValue   = "RandomValue"

	CounterMetricPollCount = "PollCount"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
)

func CollectMetrics(repository handlers.Repository, memStats *runtime.MemStats) {
	repository.UpdateGauge(GaugeMetricAlloc, float64(memStats.Alloc))
	repository.UpdateGauge(GaugeMetricBuckHashSys, float64(memStats.BuckHashSys))
	repository.UpdateGauge(GaugeMetricFrees, float64(memStats.Frees))
	repository.UpdateGauge(GaugeMetricGCCPUFraction, float64(memStats.GCCPUFraction))
	repository.UpdateGauge(GaugeMetricGCSys, float64(memStats.GCSys))
	repository.UpdateGauge(GaugeMetricHeapAlloc, float64(memStats.HeapAlloc))
	repository.UpdateGauge(GaugeMetricHeapIdle, float64(memStats.HeapIdle))
	repository.UpdateGauge(GaugeMetricHeapInuse, float64(memStats.HeapInuse))
	repository.UpdateGauge(GaugeMetricHeapObjects, float64(memStats.HeapObjects))
	repository.UpdateGauge(GaugeMetricHeapReleased, float64(memStats.HeapReleased))
	repository.UpdateGauge(GaugeMetricHeapSys, float64(memStats.HeapSys))
	repository.UpdateGauge(GaugeMetricLastGC, float64(memStats.LastGC))
	repository.UpdateGauge(GaugeMetricLookups, float64(memStats.Lookups))
	repository.UpdateGauge(GaugeMetricMCacheInuse, float64(memStats.MCacheInuse))
	repository.UpdateGauge(GaugeMetricMCacheSys, float64(memStats.MCacheSys))
	repository.UpdateGauge(GaugeMetricMSpanInuse, float64(memStats.MSpanInuse))
	repository.UpdateGauge(GaugeMetricMSpanSys, float64(memStats.MSpanSys))
	repository.UpdateGauge(GaugeMetricMallocs, float64(memStats.Mallocs))
	repository.UpdateGauge(GaugeMetricNextGC, float64(memStats.NextGC))
	repository.UpdateGauge(GaugeMetricNumForcedGC, float64(memStats.NumForcedGC))
	repository.UpdateGauge(GaugeMetricNumGC, float64(memStats.NumGC))
	repository.UpdateGauge(GaugeMetricOtherSys, float64(memStats.OtherSys))
	repository.UpdateGauge(GaugeMetricPauseTotalNs, float64(memStats.PauseTotalNs))
	repository.UpdateGauge(GaugeMetricStackInuse, float64(memStats.StackInuse))
	repository.UpdateGauge(GaugeMetricStackSys, float64(memStats.StackSys))
	repository.UpdateGauge(GaugeMetricSys, float64(memStats.Sys))
	repository.UpdateGauge(GaugeMetricTotalAlloc, float64(memStats.TotalAlloc))
	repository.UpdateGauge(GaugeMetricRandomValue, rand.Float64())
	repository.UpdateCounter(CounterMetricPollCount, 1)
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

func SendMetrics(client *resty.Client, url string, repository handlers.Repository) {
	for metricName, metricValue := range repository.GetGauges() {
		valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
		SendMetric(client, url, MetricTypeGauge, metricName, valueAsString)
	}
	for metricName, metricValue := range repository.GetCounters() {
		valueAsString := strconv.FormatInt(metricValue, 10)
		SendMetric(client, url, MetricTypeCounter, metricName, valueAsString)
	}
}

func Poll(client *resty.Client, repository handlers.Repository, memStats *runtime.MemStats, config *config.AgentConfig) {
	startTime := time.Now()
	var url = "http://" + config.ServerAddress.Address + "/update/"

	for {
		currentTime := time.Now()
		runtime.ReadMemStats(memStats)
		CollectMetrics(repository, memStats)

		if currentTime.Sub(startTime).Seconds() >= float64(config.ReportInterval) {
			startTime = currentTime
			SendMetrics(client, url, repository)
		}
		time.Sleep(time.Duration(config.PollInterval) * time.Second)
	}
}
