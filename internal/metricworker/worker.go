package metricworker

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

type MetricWorker struct {
	repository handlers.Repository
	memStats   *runtime.MemStats
	client     *resty.Client
	config     *config.AgentConfig
}

func New(repository handlers.Repository, memStats *runtime.MemStats, client *resty.Client, config *config.AgentConfig) *MetricWorker {
	return &MetricWorker{
		repository: repository,
		memStats:   memStats,
		client:     client,
		config:     config,
	}
}

func (mw *MetricWorker) CollectMetrics() {
	mw.repository.UpdateGauge(GaugeMetricAlloc, float64(mw.memStats.Alloc))
	mw.repository.UpdateGauge(GaugeMetricBuckHashSys, float64(mw.memStats.BuckHashSys))
	mw.repository.UpdateGauge(GaugeMetricFrees, float64(mw.memStats.Frees))
	mw.repository.UpdateGauge(GaugeMetricGCCPUFraction, mw.memStats.GCCPUFraction)
	mw.repository.UpdateGauge(GaugeMetricGCSys, float64(mw.memStats.GCSys))
	mw.repository.UpdateGauge(GaugeMetricHeapAlloc, float64(mw.memStats.HeapAlloc))
	mw.repository.UpdateGauge(GaugeMetricHeapIdle, float64(mw.memStats.HeapIdle))
	mw.repository.UpdateGauge(GaugeMetricHeapInuse, float64(mw.memStats.HeapInuse))
	mw.repository.UpdateGauge(GaugeMetricHeapObjects, float64(mw.memStats.HeapObjects))
	mw.repository.UpdateGauge(GaugeMetricHeapReleased, float64(mw.memStats.HeapReleased))
	mw.repository.UpdateGauge(GaugeMetricHeapSys, float64(mw.memStats.HeapSys))
	mw.repository.UpdateGauge(GaugeMetricLastGC, float64(mw.memStats.LastGC))
	mw.repository.UpdateGauge(GaugeMetricLookups, float64(mw.memStats.Lookups))
	mw.repository.UpdateGauge(GaugeMetricMCacheInuse, float64(mw.memStats.MCacheInuse))
	mw.repository.UpdateGauge(GaugeMetricMCacheSys, float64(mw.memStats.MCacheSys))
	mw.repository.UpdateGauge(GaugeMetricMSpanInuse, float64(mw.memStats.MSpanInuse))
	mw.repository.UpdateGauge(GaugeMetricMSpanSys, float64(mw.memStats.MSpanSys))
	mw.repository.UpdateGauge(GaugeMetricMallocs, float64(mw.memStats.Mallocs))
	mw.repository.UpdateGauge(GaugeMetricNextGC, float64(mw.memStats.NextGC))
	mw.repository.UpdateGauge(GaugeMetricNumForcedGC, float64(mw.memStats.NumForcedGC))
	mw.repository.UpdateGauge(GaugeMetricNumGC, float64(mw.memStats.NumGC))
	mw.repository.UpdateGauge(GaugeMetricOtherSys, float64(mw.memStats.OtherSys))
	mw.repository.UpdateGauge(GaugeMetricPauseTotalNs, float64(mw.memStats.PauseTotalNs))
	mw.repository.UpdateGauge(GaugeMetricStackInuse, float64(mw.memStats.StackInuse))
	mw.repository.UpdateGauge(GaugeMetricStackSys, float64(mw.memStats.StackSys))
	mw.repository.UpdateGauge(GaugeMetricSys, float64(mw.memStats.Sys))
	mw.repository.UpdateGauge(GaugeMetricTotalAlloc, float64(mw.memStats.TotalAlloc))
	mw.repository.UpdateGauge(GaugeMetricRandomValue, rand.Float64())
	mw.repository.UpdateCounter(CounterMetricPollCount, 1)
}

func (mw *MetricWorker) SendMetric(url string, metricType string, metricName string, metricValue string) (int, error) {
	url += metricType + "/" + metricName + "/" + metricValue

	resp, err := mw.client.R().
		SetHeader("Content-Type", "text/plain; charset=utf-8").
		Post(url)

	if err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}

func (mw *MetricWorker) SendMetrics(url string) {
	for metricName, metricValue := range mw.repository.GetGauges() {
		valueAsString := strconv.FormatFloat(metricValue, 'g', -1, 64)
		_, err := mw.SendMetric(url, MetricTypeGauge, metricName, valueAsString)
		if err != nil {
			continue
		}
	}
	for metricName, metricValue := range mw.repository.GetCounters() {
		valueAsString := strconv.FormatInt(metricValue, 10)
		_, err := mw.SendMetric(url, MetricTypeCounter, metricName, valueAsString)
		if err != nil {
			continue
		}
	}
}

func (mw *MetricWorker) Poll() {
	startTime := time.Now()
	var url = "http://" + mw.config.ServerAddress.Address + "/update/"

	for {
		currentTime := time.Now()
		runtime.ReadMemStats(mw.memStats)
		mw.CollectMetrics()

		if currentTime.Sub(startTime).Seconds() >= float64(mw.config.ReportInterval) {
			startTime = currentTime
			mw.SendMetrics(url)
		}
		time.Sleep(time.Duration(mw.config.PollInterval) * time.Second)
	}
}