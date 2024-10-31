package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metric"
	proto2 "github.com/Vidkin/metrics/internal/proto"
	mock2 "github.com/Vidkin/metrics/internal/repository/mock"
	"github.com/Vidkin/metrics/internal/repository/storage"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/pkg/interceptors"
	"github.com/Vidkin/metrics/proto"
)

func BenchmarkCollectAndSendMetrics(b *testing.B) {
	serverRepository := router.NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()
	client.SetDoNotParseResponse(true)
	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	mw := New(memoryStorage, memStats, client, nil, &config.AgentConfig{Key: "", RateLimit: 5})
	var serverURL = ts.URL + "/updates/"

	b.ResetTimer()
	b.Run("poll", func(b *testing.B) {
		var count int64 = 1
		for i := 0; i < 100; i++ {
			chIn := make(chan []*metric.Metric, 10)
			go mw.CollectMetrics(context.TODO(), chIn, count)
			for w := 1; w <= mw.config.RateLimit; w++ {
				go mw.SendMetrics(context.TODO(), chIn, serverURL)
			}
		}
	})
}

func TestSendMetrics(t *testing.T) {
	tests := []struct {
		name           string
		sendToWrongURL bool
	}{
		{
			name:           "test send ok",
			sendToWrongURL: false,
		},
		{
			name:           "test send to wrong url",
			sendToWrongURL: true,
		},
	}

	serverRepository := router.NewMemoryStorage()
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	client := resty.New()
	client.SetDoNotParseResponse(true)
	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	mw := New(memoryStorage, memStats, client, nil, &config.AgentConfig{Key: ""})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)
			chIn := make(chan []*metric.Metric, 10)
			go mw.CollectMetrics(context.TODO(), chIn, 10)

			if test.sendToWrongURL {
				mw.SendMetrics(context.TODO(), chIn, ts.URL+"/wrong_url/")
				ctx := context.TODO()
				testMetrics, _ := mw.repository.GetMetrics(ctx)
				serverMetrics, _ := serverRepository.GetMetrics(ctx)
				assert.NotEqual(t, testMetrics, serverMetrics)
			} else {
				mw.SendMetrics(context.TODO(), chIn, ts.URL+"/updates/")
				ctx := context.TODO()
				testMetrics, _ := mw.repository.GetMetrics(ctx)
				serverMetrics, _ := serverRepository.GetMetrics(ctx)
				assert.ElementsMatch(t, testMetrics, serverMetrics)
			}
		})
	}
}

func TestSendMetricsGRPC(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "test send ok",
			wantErr: false,
		},
		{
			name:    "test send error",
			wantErr: true,
		},
	}

	serverRepository := &storage.FileStorage{
		FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
		Gauge:           make(map[string]float64),
		Counter:         make(map[string]int64),
	}
	ms := &proto2.MetricsServer{
		Repository:    serverRepository,
		LastStoreTime: time.Now(),
		RetryCount:    2,
		StoreInterval: 10,
	}
	defer os.Remove(filepath.Join(os.TempDir(), "metricsTestFile.test"))
	defer os.Remove(filepath.Join(os.TempDir(), "metricsTestFile2.test"))

	var s *grpc.Server
	s = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LoggingInterceptor,
			interceptors.TrustedSubnetInterceptor("127.0.0.0/24"),
			interceptors.HashInterceptor("testKey")))
	proto.RegisterMetricsServer(s, ms)

	listen, err := net.Listen("tcp", "127.0.0.1:8080")
	require.NoError(t, err)
	go func() {
		err = s.Serve(listen)
		require.NoError(t, err)
	}()
	defer s.Stop()

	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	conn, err := grpc.NewClient("127.0.0.1:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		require.NoError(t, err)
	}(conn)
	clientGRPC := proto.NewMetricsClient(conn)
	mw := New(memoryStorage, memStats, nil, clientGRPC, &config.AgentConfig{Key: "testKey"})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			chIn := make(chan []*metric.Metric, 10)
			go mw.CollectMetrics(context.TODO(), chIn, 10)

			if test.wantErr {
				mw.config.Key = "badKey"
			}
			mw.SendMetricsGRPC(context.Background(), chIn)
			ctx := context.TODO()
			testMetrics, _ := mw.repository.GetMetrics(ctx)
			serverMetrics, _ := serverRepository.GetMetrics(ctx)
			if !test.wantErr {
				assert.ElementsMatch(t, testMetrics, serverMetrics)
			} else {
				assert.Equal(t, 0, len(serverMetrics))
				assert.NotEqual(t, 0, len(testMetrics))
			}
		})
	}
}

func TestSendMetric(t *testing.T) {
	var testIntValue int64 = 42
	var testFloatValue = 42.5
	type want struct {
		resp       string
		statusCode int
	}
	tests := []struct {
		name           string
		metric         metric.Metric
		want           want
		sendToWrongURL bool
		statusCode     int
	}{
		{
			name:           "test send counter ok",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: MetricTypeCounter,
				ID:    "test",
				Delta: &testIntValue,
			},
			want: want{
				resp:       `{"type":"counter","id":"test","delta":42}`,
				statusCode: http.StatusOK,
			},
		},
		{
			name:           "test send gauge ok",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: MetricTypeGauge,
				ID:    "test",
				Value: &testFloatValue,
			},
			want: want{
				resp:       `{"type":"gauge","id":"test","value":42.5}`,
				statusCode: http.StatusOK,
			},
		},
		{
			name:           "test send bad metric type",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "test",
				Delta: &testIntValue,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:           "test send empty metric name",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "",
				Delta: &testIntValue,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:           "test send bad metric value",
			sendToWrongURL: false,
			metric: metric.Metric{
				MType: "badMetricType",
				ID:    "test",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	serverRepository := router.NewMemoryStorage()
	client := resty.New()
	client.SetDoNotParseResponse(true)
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 300}
	metricRouter := router.NewMetricRouter(chiRouter, serverRepository, &serverConfig)
	ts := httptest.NewServer(metricRouter.Router)
	defer ts.Close()

	mw := New(nil, nil, client, nil, &config.AgentConfig{Key: ""})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clear(serverRepository.Gauge)
			clear(serverRepository.Counter)

			if test.sendToWrongURL {
				_, _, err := mw.SendMetric(context.TODO(), ts.URL+"/wrong_url/", &test.metric)
				assert.NotNil(t, err)
			} else {
				respCode, respBody, err := mw.SendMetric(context.TODO(), ts.URL+"/update/", &test.metric)
				assert.Equal(t, test.want.statusCode, respCode)
				if test.want.statusCode == http.StatusOK {
					assert.JSONEq(t, test.want.resp, respBody)
					assert.Nil(t, err)
				}
			}
		})
	}
}

func TestPoll(t *testing.T) {
	mockController := gomock.NewController(t)
	serverRepository := mock2.NewMockRepository(mockController)

	ms := &proto2.MetricsServer{
		Repository:    serverRepository,
		LastStoreTime: time.Now(),
		RetryCount:    2,
		StoreInterval: 10,
	}
	defer os.Remove(filepath.Join(os.TempDir(), "metricsTestFile.test"))
	defer os.Remove(filepath.Join(os.TempDir(), "metricsTestFile2.test"))

	var s *grpc.Server
	s = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LoggingInterceptor,
			interceptors.TrustedSubnetInterceptor("127.0.0.0/24"),
			interceptors.HashInterceptor("testKey")))
	proto.RegisterMetricsServer(s, ms)

	listen, err := net.Listen("tcp", "127.0.0.1:8081")
	require.NoError(t, err)
	go func() {
		err = s.Serve(listen)
		require.NoError(t, err)
	}()
	defer s.Stop()

	memStats := &runtime.MemStats{}
	memoryStorage := router.NewFileStorage("")
	conn, err := grpc.NewClient("127.0.0.1:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		require.NoError(t, err)
	}(conn)
	clientGRPC := proto.NewMetricsClient(conn)
	mw := New(memoryStorage, memStats, nil, clientGRPC, &config.AgentConfig{UseGRPC: true, ReportInterval: 1, PollInterval: 2, RateLimit: 1, Key: "testKey", ServerAddress: &config.ServerAddress{Address: "127.0.0.1:8081"}})

	serverRepository.EXPECT().
		UpdateMetrics(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	serverRepository.EXPECT().
		GetMetrics(gomock.Any()).
		Return([]*metric.Metric{}, nil).AnyTimes()

	fVal := 12.2
	serverRepository.EXPECT().
		GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&metric.Metric{ID: "test", MType: MetricTypeGauge, Value: &fVal}, nil).AnyTimes()

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	go mw.Poll(ctx)

	time.Sleep(5 * time.Second)
}
