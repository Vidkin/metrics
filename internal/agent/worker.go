// Package agent provides functionality for collecting and sending metrics from the agent to a server.
package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	pb "google.golang.org/protobuf/proto"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/pkg/hash"
	"github.com/Vidkin/metrics/pkg/ip"
	"github.com/Vidkin/metrics/proto"
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
	clientGRPC proto.MetricsClient
	config     *config.AgentConfig
}

func New(repository router.Repository, memStats *runtime.MemStats, client *resty.Client, clientGRPC proto.MetricsClient, config *config.AgentConfig) *MetricWorker {
	return &MetricWorker{
		repository: repository,
		memStats:   memStats,
		client:     client,
		clientGRPC: clientGRPC,
		config:     config,
	}
}

func (mw *MetricWorker) CollectMetrics(ctx context.Context, chIn chan []*metric.Metric, count int64) {
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

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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
	}
	cMetric := &metric.Metric{
		ID:    CounterMetricPollCount,
		MType: MetricTypeCounter,
		Delta: &count,
	}
	err = mw.repository.UpdateMetric(ctx, cMetric)
	if err != nil {
		logger.Log.Error("error update counter metric", zap.Error(err))
		return
	}

	metrics, err := mw.repository.GetMetrics(ctx)
	if err != nil {
		logger.Log.Error("error get metrics from repository", zap.Error(err))
		return
	}
	chIn <- metrics
}

func (mw *MetricWorker) SendMetric(ctx context.Context, url string, metric *metric.Metric) (int, string, error) {
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

	interfaces, err := ip.GetMyInterfaces()
	if err != nil {
		logger.Log.Info("error get net interfaces", zap.Error(err))
		return 0, "", err
	}
	if len(interfaces) == 0 {
		logger.Log.Info("error get net interfaces")
		return 0, "", errors.New("error get net interfaces")
	}

	req := mw.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("X-Real-IP", interfaces[0]).
		SetBody(buf)

	resp, err := req.SetContext(ctx).Post(url)
	if err != nil {
		logger.Log.Info("error post request", zap.Error(err))
		return 0, "", err
	}
	defer func(body io.ReadCloser) {
		err = body.Close()
		if err != nil {
			logger.Log.Info("error close resp raw body", zap.Error(err))
		}
	}(resp.RawBody())

	contentEncoding := resp.Header().Get("Content-Encoding")
	var or io.ReadCloser
	if strings.Contains(contentEncoding, "gzip") {
		var cr *gzip.Reader
		cr, err = gzip.NewReader(resp.RawBody())
		if err != nil {
			logger.Log.Info("error init gzip reader", zap.Error(err))
			return 0, "", err
		}
		or = cr
	} else {
		or = resp.RawBody()
	}

	select {
	case <-ctx.Done():
		logger.Log.Info("SendMetric shutdown due to context cancellation")
		return 0, "", ctx.Err() // Возвращаем ошибку отмены контекста
	default:
		respBody, err := io.ReadAll(or)
		if err != nil {
			logger.Log.Info("error read response body", zap.Error(err))
			return 0, "", err
		}
		return resp.StatusCode(), string(respBody), nil
	}
}

func (mw *MetricWorker) SendMetricsGRPC(ctx context.Context, chIn chan []*metric.Metric) {
	for {
		select {
		case m, ok := <-chIn:
			if !ok {
				return
			}

			protoMetrics := make([]*proto.Metric, len(m))
			for i, met := range m {
				protoMetrics[i] = &proto.Metric{
					Id: met.ID,
				}
				if met.MType == MetricTypeCounter {
					protoMetrics[i].Delta = *met.Delta
					protoMetrics[i].Type = proto.Metric_COUNTER
				} else {
					protoMetrics[i].Value = *met.Value
					protoMetrics[i].Type = proto.Metric_GAUGE
				}
			}

			for i := 0; i <= RequestRetryCount; i++ {
				ctxTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
				defer cancel()

				req := &proto.UpdateMetricsRequest{
					Metrics: protoMetrics,
				}

				if mw.config.Key != "" {
					data, err := pb.Marshal(req)
					if err != nil {
						logger.Log.Error("failed to marshal request: %v", zap.Error(err))
						continue
					}
					h := hash.GetHashSHA256(mw.config.Key, data)
					hEnc := base64.StdEncoding.EncodeToString(h)
					md := metadata.New(map[string]string{"HashSHA256": hEnc})
					ctxTimeout = metadata.NewOutgoingContext(ctxTimeout, md)
				}

				_, err := mw.clientGRPC.UpdateMetrics(ctxTimeout, req)
				if err != nil {
					if e, ok := status.FromError(err); ok {
						logger.Log.Error("code = " + e.Code().String() + ", message = " + e.Message())
					} else {
						logger.Log.Error("error update metrics", zap.Error(err))
					}

					if i != RequestRetryCount {
						continue
					}
				}
				break
			}

		case <-ctx.Done():
			return
		}
	}
}

func (mw *MetricWorker) SendMetrics(ctx context.Context, chIn chan []*metric.Metric, serverURL string) {
	for {
		select {
		case m, ok := <-chIn:
			if !ok {
				return
			}
			body, _ := json.Marshal(m)
			buf := bytes.NewBuffer([]byte{})
			zb := gzip.NewWriter(buf)
			_, _ = zb.Write(body)
			err := zb.Close()
			if err != nil {
				logger.Log.Info("error close gzip writer", zap.Error(err))
			}

			for i := 0; i <= RequestRetryCount; i++ {
				req := mw.client.R()
				if mw.config.Key != "" {
					h := hash.GetHashSHA256(mw.config.Key, buf.Bytes())
					hEnc := base64.StdEncoding.EncodeToString(h)
					req.SetHeader("HashSHA256", hEnc)
				}
				interfaces, err := ip.GetMyInterfaces()
				if err != nil || len(interfaces) == 0 {
					logger.Log.Info("error get net interfaces", zap.Error(err))
					return
				}
				_, err = req.
					SetHeader("Content-Type", "application/json").
					SetHeader("Content-Encoding", "gzip").
					SetHeader("Accept-Encoding", "gzip").
					SetHeader("X-Real-IP", interfaces[0]).
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

		case <-ctx.Done():
			return
		}
	}
}

func (mw *MetricWorker) Poll(ctx context.Context) {
	startTime := time.Now()
	protocol := "http"
	if mw.config.CryptoKey != "" {
		protocol = "https"
	}
	var serverURL = protocol + "://" + mw.config.ServerAddress.Address + "/updates/"
	var count int64 = 0

	var wg sync.WaitGroup

	for {
		chIn := make(chan []*metric.Metric, mw.config.RateLimit)
		select {
		case <-ctx.Done():
			logger.Log.Info("application shutdown")
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if !mw.config.UseGRPC {
				mw.SendMetrics(ctxTimeout, chIn, serverURL)
			} else {
				mw.SendMetricsGRPC(ctxTimeout, chIn)
			}

			ctxWait, cancelWait := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelWait()
			go func() {
				wg.Wait()
			}()

			<-ctxWait.Done()
			return
		default:
			currentTime := time.Now()

			wg.Add(1)
			go func() {
				defer wg.Done()
				mw.CollectMetrics(ctx, chIn, count)
			}()

			if currentTime.Sub(startTime).Seconds() >= float64(mw.config.ReportInterval) {
				startTime = currentTime
				for w := 1; w <= mw.config.RateLimit; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						if !mw.config.UseGRPC {
							mw.SendMetrics(ctx, chIn, serverURL)
						} else {
							mw.SendMetricsGRPC(ctx, chIn)
						}
					}()
				}
			}
			time.Sleep(time.Duration(mw.config.PollInterval) * time.Second)
			count++
		}
	}
}
