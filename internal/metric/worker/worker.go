package worker

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/go-resty/resty/v2"
	"io"
	"math/rand/v2"
	"runtime"
	"strings"
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
	repository router.Repository
	memStats   *runtime.MemStats
	client     *resty.Client
	config     *config.AgentConfig
}

func New(repository router.Repository, memStats *runtime.MemStats, client *resty.Client, config *config.AgentConfig) *MetricWorker {
	return &MetricWorker{
		repository: repository,
		memStats:   memStats,
		client:     client,
		config:     config,
	}
}

func (mw *MetricWorker) CollectMetrics(count int64) {
	gaugeMetrics := map[string]float64{
		GaugeMetricAlloc:         float64(mw.memStats.Alloc),
		GaugeMetricBuckHashSys:   float64(mw.memStats.BuckHashSys),
		GaugeMetricFrees:         float64(mw.memStats.Frees),
		GaugeMetricMCacheSys:     float64(mw.memStats.MCacheSys),
		GaugeMetricMSpanInuse:    float64(mw.memStats.MSpanInuse),
		GaugeMetricNumForcedGC:   float64(mw.memStats.NumForcedGC),
		GaugeMetricGCCPUFraction: mw.memStats.GCCPUFraction,
		GaugeMetricGCSys:         float64(mw.memStats.GCSys),
		GaugeMetricHeapAlloc:     float64(mw.memStats.HeapAlloc),
		GaugeMetricHeapIdle:      float64(mw.memStats.HeapIdle),
		GaugeMetricHeapInuse:     float64(mw.memStats.HeapInuse),
		GaugeMetricHeapObjects:   float64(mw.memStats.HeapObjects),
		GaugeMetricHeapReleased:  float64(mw.memStats.HeapReleased),
		GaugeMetricHeapSys:       float64(mw.memStats.HeapSys),
		GaugeMetricLastGC:        float64(mw.memStats.LastGC),
		GaugeMetricLookups:       float64(mw.memStats.Lookups),
		GaugeMetricMCacheInuse:   float64(mw.memStats.MCacheInuse),
		GaugeMetricMSpanSys:      float64(mw.memStats.MSpanSys),
		GaugeMetricMallocs:       float64(mw.memStats.Mallocs),
		GaugeMetricNextGC:        float64(mw.memStats.NextGC),
		GaugeMetricNumGC:         float64(mw.memStats.NumGC),
		GaugeMetricOtherSys:      float64(mw.memStats.OtherSys),
		GaugeMetricPauseTotalNs:  float64(mw.memStats.PauseTotalNs),
		GaugeMetricStackInuse:    float64(mw.memStats.StackInuse),
		GaugeMetricStackSys:      float64(mw.memStats.StackSys),
		GaugeMetricSys:           float64(mw.memStats.Sys),
		GaugeMetricTotalAlloc:    float64(mw.memStats.TotalAlloc),
		GaugeMetricRandomValue:   rand.Float64(),
	}
	for k, v := range gaugeMetrics {
		mw.repository.UpdateMetric(&metric.Metric{
			ID:    k,
			MType: MetricTypeGauge,
			Value: &v,
		})
	}
	mw.repository.UpdateMetric(&metric.Metric{
		ID:    CounterMetricPollCount,
		MType: MetricTypeCounter,
		Delta: &count,
	})
}

func (mw *MetricWorker) SendMetric(url string, metric *metric.Metric) (int, string, error) {
	body, err := json.Marshal(metric)
	if err != nil {
		return 0, "", err
	}

	buf := bytes.NewBuffer(nil)
	zb := gzip.NewWriter(buf)
	_, err = zb.Write(body)
	if err != nil {
		return 0, "", err
	}

	err = zb.Close()
	if err != nil {
		return 0, "", err
	}

	resp, err := mw.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(buf).
		Post(url)

	if err != nil {
		return 0, "", err
	}
	defer resp.RawBody().Close()

	contentEncoding := resp.Header().Get("Content-Encoding")
	var or io.ReadCloser
	if strings.Contains(contentEncoding, "gzip") {
		cr, err := gzip.NewReader(resp.RawBody())
		if err != nil {
			return 0, "", err
		}
		or = cr
	} else {
		or = resp.RawBody()
	}
	respBody, err := io.ReadAll(or)
	if err != nil {
		return 0, "", err
	}

	return resp.StatusCode(), string(respBody), nil
}

func (mw *MetricWorker) SendMetrics(url string) {
	for _, metric := range mw.repository.GetMetrics() {
		_, _, err := mw.SendMetric(url, metric)
		if err != nil {
			continue
		}
	}
}

func (mw *MetricWorker) Poll() {
	startTime := time.Now()
	var url = "http://" + mw.config.ServerAddress.Address + "/update/"
	var count int64 = 0
	for {
		currentTime := time.Now()
		runtime.ReadMemStats(mw.memStats)
		mw.CollectMetrics(count)

		if currentTime.Sub(startTime).Seconds() >= float64(mw.config.ReportInterval) {
			startTime = currentTime
			mw.SendMetrics(url)
		}
		time.Sleep(time.Duration(mw.config.PollInterval) * time.Second)
		count++
	}
}