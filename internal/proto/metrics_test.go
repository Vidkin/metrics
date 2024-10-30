package proto

import (
	"context"
	"encoding/base64"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	pb "google.golang.org/protobuf/proto"

	"github.com/Vidkin/metrics/internal/repository/mock"
	"github.com/Vidkin/metrics/internal/repository/storage"
	"github.com/Vidkin/metrics/internal/router"
	"github.com/Vidkin/metrics/pkg/hash"
	"github.com/Vidkin/metrics/pkg/interceptors"
	"github.com/Vidkin/metrics/proto"
)

func TestMetricsServer_UpdateMetrics(t *testing.T) {
	type params struct {
		MockRepository *mock.MockRepository
		Repository     router.Repository
		LastStoreTime  time.Time
		TrustedSubnet  string
		Key            string
		ClientKey      string
		ServerAddress  string
		RetryCount     int
		StoreInterval  int
	}
	tests := []struct {
		params    *params
		in        *proto.UpdateMetricsRequest
		want      *proto.UpdateMetricsResponse
		updateErr error
		getErr    error
		name      string
		wantErr   bool
	}{
		{
			name: "update metrics ok",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "127.0.0.0/24",
				ServerAddress: "127.0.0.1:8080",
			},

			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update metrics ok with hash",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				Key:           "test",
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "127.0.0.0/24",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update metrics with bad hash",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				Key:           "test",
				ClientKey:     "badKey",
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "127.0.0.0/24",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update metrics with empty",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				Key:           "test",
				ClientKey:     "",
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "127.0.0.0/24",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update metrics not in trusted subnet",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "192.168.1.0/24",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update metrics err parsing CIDR",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 10,
				TrustedSubnet: "badCIDR",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update metrics bad storage path",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: "/badPath//",
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 0,
				TrustedSubnet: "127.0.0.0/24",
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update metrics ok without subnet",
			params: &params{
				Repository: &storage.FileStorage{
					FileStoragePath: filepath.Join(os.TempDir(), "metricsTestFile.test"),
					Gauge:           make(map[string]float64),
					Counter:         make(map[string]int64),
				},
				LastStoreTime: time.Now(),
				RetryCount:    2,
				StoreInterval: 0,
				ServerAddress: "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update metrics with unavailable postgresql storage on update metrics op",
			params: &params{
				MockRepository: mock.NewMockRepository(gomock.NewController(t)),
				LastStoreTime:  time.Now(),
				RetryCount:     2,
				StoreInterval:  20,
				ServerAddress:  "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr:   true,
			updateErr: &pgconn.PgError{Code: pgerrcode.ConnectionException},
		},
		{
			name: "update metrics with unavailable postgresql storage on get metric op",
			params: &params{
				MockRepository: mock.NewMockRepository(gomock.NewController(t)),
				LastStoreTime:  time.Now(),
				RetryCount:     2,
				StoreInterval:  20,
				ServerAddress:  "127.0.0.1:8080",
			},
			in: &proto.UpdateMetricsRequest{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			want: &proto.UpdateMetricsResponse{
				Metrics: []*proto.Metric{
					{
						Delta: 12,
						Id:    "c1",
						Type:  proto.Metric_COUNTER,
					},
					{
						Value: 12.2,
						Id:    "g1",
						Type:  proto.Metric_GAUGE,
					},
				},
			},
			wantErr:   true,
			updateErr: nil,
			getErr:    &pgconn.PgError{Code: pgerrcode.ConnectionException},
		},
	}

	defer os.Remove(filepath.Join(os.TempDir(), "metricsTestFile.test"))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MetricsServer{
				Repository:    tt.params.Repository,
				LastStoreTime: tt.params.LastStoreTime,
				RetryCount:    tt.params.RetryCount,
				StoreInterval: tt.params.StoreInterval,
			}
			var s *grpc.Server
			s = grpc.NewServer(
				grpc.ChainUnaryInterceptor(
					interceptors.LoggingInterceptor,
					interceptors.TrustedSubnetInterceptor(tt.params.TrustedSubnet),
					interceptors.HashInterceptor(tt.params.Key)))
			proto.RegisterMetricsServer(s, ms)

			listen, err := net.Listen("tcp", tt.params.ServerAddress)
			require.NoError(t, err)

			go func() {
				err = s.Serve(listen)
				require.NoError(t, err)
			}()

			conn, err := grpc.NewClient(tt.params.ServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer func(conn *grpc.ClientConn) {
				err = conn.Close()
				require.NoError(t, err)
			}(conn)

			clientGRPC := proto.NewMetricsClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			if !tt.wantErr {
				if tt.params.Key != "" {
					data, errM := pb.Marshal(tt.in)
					require.NoError(t, errM)
					h := hash.GetHashSHA256(tt.params.Key, data)
					hEnc := base64.StdEncoding.EncodeToString(h)
					md := metadata.New(map[string]string{"HashSHA256": hEnc})
					ctx = metadata.NewOutgoingContext(ctx, md)
				}

				got, errU := clientGRPC.UpdateMetrics(ctx, tt.in)
				require.NoError(t, errU)
				assert.Equal(t, tt.want.Metrics, got.Metrics)
			} else {
				if tt.params.ClientKey != "" {
					data, errM := pb.Marshal(tt.in)
					require.NoError(t, errM)
					h := hash.GetHashSHA256(tt.params.ClientKey, data)
					hEnc := base64.StdEncoding.EncodeToString(h)
					md := metadata.New(map[string]string{"HashSHA256": hEnc})
					ctx = metadata.NewOutgoingContext(ctx, md)
				}

				if tt.params.MockRepository != nil {
					ms.Repository = tt.params.MockRepository
					tt.params.MockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(tt.updateErr)
					if tt.getErr != nil {
						tt.params.MockRepository.EXPECT().
							GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
							Return(nil, tt.getErr)
					}
				}
				_, err = clientGRPC.UpdateMetrics(ctx, tt.in)
				require.Error(t, err)
			}

			s.Stop()
		})
	}
}
