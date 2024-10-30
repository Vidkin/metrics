package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Vidkin/metrics/internal/config"
	me "github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/repository/storage"
	"github.com/Vidkin/metrics/internal/router"
)

func TestNewServerApp(t *testing.T) {
	tests := []struct {
		cfg     *config.ServerConfig
		name    string
		wantErr bool
	}{
		{
			name: "test bad log level",
			cfg: &config.ServerConfig{
				LogLevel: "badLevel",
			},
			wantErr: true,
		},
		{
			name: "test bad repo config",
			cfg: &config.ServerConfig{
				LogLevel:    "info",
				DatabaseDSN: "badDSN",
			},
			wantErr: true,
		},
		{
			name: "test good with gRPC",
			cfg: &config.ServerConfig{
				LogLevel:      "info",
				UseGRPC:       true,
				Key:           "testKey",
				TrustedSubnet: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "test good with HTTP",
			cfg: &config.ServerConfig{
				LogLevel:      "info",
				UseGRPC:       false,
				Key:           "testKey",
				TrustedSubnet: "127.0.0.1",
				ServerAddress: &config.ServerAddress{Address: "127.0.0.1:8080"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewServerApp(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestServerApp_DumpToFile(t *testing.T) {
	fVal := 12.2
	iVal := int64(1)

	tests := []struct {
		fileStorage     router.Repository
		serverApp       *ServerApp
		name            string
		fileStoragePath string
		wantErr         bool
	}{
		{
			name:            "test good",
			wantErr:         false,
			fileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
			fileStorage: &storage.FileStorage{
				FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
				GaugeMetrics: []*me.Metric{
					{
						MType: router.MetricTypeGauge,
						ID:    "gauge",
						Value: &fVal,
					},
				},
				CounterMetrics: []*me.Metric{
					{
						MType: router.MetricTypeCounter,
						ID:    "counter",
						Delta: &iVal,
					},
				},
			},
		},
		{
			name:            "test repository bad path",
			wantErr:         true,
			fileStoragePath: "/badPath//",
			fileStorage: &storage.FileStorage{
				FileStoragePath: "/badPath//",
				GaugeMetrics: []*me.Metric{
					{
						MType: router.MetricTypeGauge,
						ID:    "gauge",
						Value: &fVal,
					},
				},
				CounterMetrics: []*me.Metric{
					{
						MType: router.MetricTypeCounter,
						ID:    "counter",
						Delta: &iVal,
					},
				},
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				a := &ServerApp{
					config:     &config.ServerConfig{FileStoragePath: tt.fileStoragePath, RetryCount: 2},
					repository: tt.fileStorage,
				}
				err := a.DumpToFile()
				assert.Error(t, err)
			} else {
				a := &ServerApp{
					config:     &config.ServerConfig{FileStoragePath: tt.fileStoragePath, RetryCount: 2},
					repository: tt.fileStorage,
				}
				err := a.DumpToFile()
				assert.NoError(t, err)

				os.Remove(tt.fileStoragePath)
			}
		})
	}
}

func TestServerApp_Run(t *testing.T) {
	tests := []struct {
		config *config.ServerConfig
		name   string
	}{
		{
			name: "test with gRPC",
			config: &config.ServerConfig{
				ServerAddress: &config.ServerAddress{Address: "127.0.0.1:8080"},
				LogLevel:      "info",
				UseGRPC:       true,
				Key:           "testKey",
				TrustedSubnet: "127.0.0.1",
				StoreInterval: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverApp, err := NewServerApp(tt.config)
			require.NoError(t, err)
			go serverApp.Run()
			time.Sleep(2 * time.Second)
			serverApp.Stop()
		})
	}
}

func TestServerApp_Serve(t *testing.T) {
	tests := []struct {
		config *config.ServerConfig
		name   string
	}{
		{
			name: "test with gRPC",
			config: &config.ServerConfig{
				ServerAddress: &config.ServerAddress{Address: "127.0.0.1:8080"},
				LogLevel:      "info",
				UseGRPC:       true,
				Key:           "testKey",
				TrustedSubnet: "127.0.0.1",
			},
		},
		{
			name: "test with HTTP",
			config: &config.ServerConfig{
				ServerAddress: &config.ServerAddress{Address: "127.0.0.1:8080"},
				LogLevel:      "info",
				UseGRPC:       false,
				Key:           "testKey",
				TrustedSubnet: "127.0.0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverApp, err := NewServerApp(tt.config)
			require.NoError(t, err)
			go serverApp.Serve()
			time.Sleep(1 * time.Second)
			serverApp.Stop()
		})
	}
}

func TestServerApp_Stop(t *testing.T) {
	fVal := 12.2
	iVal := int64(1)

	tests := []struct {
		fileStorage     router.Repository
		serverApp       *ServerApp
		fileStoragePath string
		name            string
		wantErr         bool
	}{
		{
			name:            "test good",
			wantErr:         false,
			fileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
			fileStorage: &storage.FileStorage{
				FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
				GaugeMetrics: []*me.Metric{
					{
						MType: router.MetricTypeGauge,
						ID:    "gauge",
						Value: &fVal,
					},
				},
				CounterMetrics: []*me.Metric{
					{
						MType: router.MetricTypeCounter,
						ID:    "counter",
						Delta: &iVal,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ServerApp{
				config: &config.ServerConfig{
					FileStoragePath: tt.fileStoragePath,
					RetryCount:      2,
					ServerAddress:   &config.ServerAddress{Address: "127.0.0.1:8080"}},
				repository: tt.fileStorage,
			}
			a.Stop()

			file, err := os.Open(tt.fileStoragePath)
			assert.NoError(t, err)
			defer file.Close()

			info, err := file.Stat()

			assert.NoError(t, err)
			assert.NotEqual(t, 0, info.Size())

			os.Remove(tt.fileStoragePath)
		})
	}
}
