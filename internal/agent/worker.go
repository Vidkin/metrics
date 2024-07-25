package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/pkg/hash"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"io"
	"math/rand/v2"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	GaugeMetricAlloc          = "Alloc"
	GaugeMetricBuckHashSys    = "BuckHashSys"
	GaugeMetricFrees          = "Frees"
	GaugeMetricGCCPUFraction  = "GCCPUFraction"
	GaugeMetricGCSys          = "GCSys"
	GaugeMetricHeapAlloc      = "HeapAlloc"
	GaugeMetricHeapIdle       = "HeapIdle"
	GaugeMetricHeapInuse      = "HeapInuse"
	GaugeMetricHeapObjects    = "HeapObjects"
	GaugeMetricHeapReleased   = "HeapReleased"
	GaugeMetricHeapSys        = "HeapSys"
	GaugeMetricLastGC         = "LastGC"
	GaugeMetricLookups        = "Lookups"
	GaugeMetricMCacheInuse    = "MCacheInuse"
	GaugeMetricMCacheSys      = "MCacheSys"
	GaugeMetricMSpanInuse     = "MSpanInuse"
	GaugeMetricMSpanSys       = "MSpanSys"
	GaugeMetricMallocs        = "Mallocs"
	GaugeMetricNextGC         = "NextGC"
	GaugeMetricNumForcedGC    = "NumForcedGC"
	GaugeMetricNumGC          = "NumGC"
	GaugeMetricOtherSys       = "OtherSys"
	GaugeMetricPauseTotalNs   = "PauseTotalNs"
	GaugeMetricStackInuse     = "StackInuse"
	GaugeMetricStackSys       = "StackSys"
	GaugeMetricSys            = "Sys"
	GaugeMetricTotalAlloc     = "TotalAlloc"
	GaugeMetricRandomValue    = "RandomValue"
	GaugeMetricTotalMemory    = "TotalMemory"
	GaugeMetricFreeMemory     = "FreeMemory"
	GaugeMetricCPUutilization = "CPUutilization"

	CounterMetricPollCount = "PollCount"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"

	RequestRetryCount = 3
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

func (mw *MetricWorker) CollectMetrics(chIn chan *metric.Metric, count int64) {
	defer close(chIn)
	runtime.ReadMemStats(mw.memStats)

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		logger.Log.Error("error collect memory metrics", zap.Error(err))
		return
	}
	totalMemory := vmStat.Total / 1024 / 1024
	freeMemory := vmStat.Free / 1024 / 1024

	percentages, err := cpu.Percent(0, true)
	if err != nil {
		logger.Log.Error("error collect cpu utilization metrics", zap.Error(err))
		return
	}

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
		GaugeMetricTotalMemory:   float64(totalMemory),
		GaugeMetricFreeMemory:    float64(freeMemory),
		GaugeMetricRandomValue:   rand.Float64(),
	}
	for i, percentage := range percentages {
		gaugeMetrics[GaugeMetricCPUutilization+strconv.Itoa(i+1)] = percentage
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for k, v := range gaugeMetrics {
		gMetric := &metric.Metric{
			ID:    k,
			MType: MetricTypeGauge,
			Value: &v,
		}
		err = mw.repository.UpdateMetric(ctx, gMetric)
		if err != nil {
			logger.Log.Error("error update gauge metric", zap.Error(err))
			return
		}
		chIn <- gMetric
	}
	cMetric := &metric.Metric{
		ID:    CounterMetricPollCount,
		MType: MetricTypeCounter,
		Delta: &count,
	}
	err = mw.repository.UpdateMetric(ctx, cMetric)
	chIn <- cMetric
	if err != nil {
		logger.Log.Error("error update counter metric", zap.Error(err))
		return
	}

	return
}

func (mw *MetricWorker) SendMetric(url string, metric *metric.Metric) (int, string, error) {
	body, err := json.Marshal(metric)
	if err != nil {
		logger.Log.Info("error marshal body", zap.Error(err))
		return 0, "", err
	}

	buf := bytes.NewBuffer(nil)
	zb := gzip.NewWriter(buf)
	_, err = zb.Write(body)
	if err != nil {
		logger.Log.Info("error gzip body", zap.Error(err))
		return 0, "", err
	}

	err = zb.Close()
	if err != nil {
		logger.Log.Info("error close gzip buffer", zap.Error(err))
		return 0, "", err
	}

	resp, err := mw.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(buf).
		Post(url)

	if err != nil {
		logger.Log.Info("error post request", zap.Error(err))
		return 0, "", err
	}
	defer resp.RawBody().Close()

	contentEncoding := resp.Header().Get("Content-Encoding")
	var or io.ReadCloser
	if strings.Contains(contentEncoding, "gzip") {
		cr, err := gzip.NewReader(resp.RawBody())
		if err != nil {
			logger.Log.Info("error init gzip reader", zap.Error(err))
			return 0, "", err
		}
		or = cr
	} else {
		or = resp.RawBody()
	}
	respBody, err := io.ReadAll(or)
	if err != nil {
		logger.Log.Info("error read response body", zap.Error(err))
		return 0, "", err
	}

	return resp.StatusCode(), string(respBody), nil
}

func (mw *MetricWorker) SendMetrics(chIn chan *metric.Metric, serverURL string) {
	for m := range chIn {
		body, _ := json.Marshal([]*metric.Metric{m})
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, _ = zb.Write(body)
		zb.Close()

		for i := 0; i <= RequestRetryCount; i++ {
			req := mw.client.R()
			if mw.config.Key != "" {
				h := hash.GetHashSHA256(mw.config.Key, buf.Bytes())
				hEnc := base64.StdEncoding.EncodeToString(h)
				req.SetHeader("HashSHA256", hEnc)
			}
			_, err := req.
				SetHeader("Content-Type", "application/json").
				SetHeader("Content-Encoding", "gzip").
				SetHeader("Accept-Encoding", "gzip").
				SetBody(buf).
				Post(serverURL)
			if err != nil {
				var urlErr *url.Error
				if errors.As(err, &urlErr) && i != RequestRetryCount {
					logger.Log.Info("error post request", zap.Error(err))
					time.Sleep(time.Duration(1+i*2) * time.Second)
					continue
				}
				logger.Log.Info("error post request", zap.Error(err))
				return
			}
			break
		}
	}
}

func (mw *MetricWorker) Poll() {
	startTime := time.Now()
	var serverURL = "http://" + mw.config.ServerAddress.Address + "/updates/"
	var count int64 = 0
	for {
		currentTime := time.Now()
		chIn := make(chan *metric.Metric)
		go mw.CollectMetrics(chIn, count)

		if currentTime.Sub(startTime).Seconds() >= float64(mw.config.ReportInterval) {
			startTime = currentTime
			for w := 1; w <= mw.config.RateLimit; w++ {
				go mw.SendMetrics(chIn, serverURL)
			}
		}
		time.Sleep(time.Duration(mw.config.PollInterval) * time.Second)
		count++
	}
}
