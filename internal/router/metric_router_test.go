package router

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/metric"
	"github.com/Vidkin/metrics/internal/repository/mock"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func (s *MetricRouterTestSuite) TestMetricRouter_GzipCompression() {
	requestBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	successBody := `{
		"id": "test",
		"type": "gauge",
		"value": 13.5
	}`

	s.Run("sends_gzip", func() {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		s.Require().NoError(err)
		err = zb.Close()
		s.Require().NoError(err)

		s.mockRepository.EXPECT().
			UpdateMetric(gomock.Any(), gomock.Any()).
			Return(nil)
		s.mockRepository.EXPECT().
			Dump(gomock.Any()).
			Return(nil)

		value := 13.5
		s.mockRepository.EXPECT().
			GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&metric.Metric{ID: "test", MType: MetricTypeGauge, Value: &value}, nil)

		resp, respBody := s.RequestTest(http.MethodPost, "/update", buf.String(), "application/json", false, true)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().JSONEq(successBody, string(respBody))
	})

	s.Run("accepts_gzip", func() {
		s.mockRepository.EXPECT().
			UpdateMetric(gomock.Any(), gomock.Any()).
			Return(nil)
		s.mockRepository.EXPECT().
			Dump(gomock.Any()).
			Return(nil)

		value := 13.5
		s.mockRepository.EXPECT().
			GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&metric.Metric{ID: "test", MType: MetricTypeGauge, Value: &value}, nil)
		resp, respBody := s.RequestTest(http.MethodPost, "/update", requestBody, "application/json", true, false)
		s.Require().Equal(http.StatusOK, resp.StatusCode)

		s.Require().JSONEq(successBody, string(respBody))
	})
}

func (s *MetricRouterTestSuite) TestMetricRouter_PingDBHandler() {
	type want struct {
		statusCode int
		response   string
	}
	tests := []struct {
		name        string
		want        want
		contentType string
	}{
		{
			name:        "test db is not available",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "test ok",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusOK,
				response:   "",
			},
		},
	}
	for _, test := range tests {
		if test.want.statusCode != http.StatusOK {
			s.mockRepository.EXPECT().
				Ping(gomock.Any()).
				Return(errors.New("can't connect to DB"))
		} else {
			s.mockRepository.EXPECT().
				Ping(gomock.Any()).
				Return(nil)
		}
		s.Run(test.name, func() {
			resp, respBody := s.RequestTest(http.MethodGet, "/ping", "", test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				s.Assert().Equal(test.want.response, string(respBody))
			}
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_RootHandler() {
	type want struct {
		statusCode int
		err        string
		response   []*metric.Metric
	}

	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)

	const (
		errGetMetrics = "errGetMetrics"
	)
	tests := []struct {
		name        string
		want        want
		contentType string
	}{
		{
			name:        "db is not available",
			contentType: "text/html",
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error get metrics from db",
			contentType: "text/html",
			want: want{
				err:        errGetMetrics,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "test ok",
			contentType: "text/html",
			want: want{
				statusCode: http.StatusOK,
				response: []*metric.Metric{
					{
						ID:    "counter",
						MType: MetricTypeCounter,
						Delta: &intValue,
					},
					{
						ID:    "gauge",
						MType: MetricTypeGauge,
						Value: &floatValue,
					},
				},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK {
				if test.want.err == errGetMetrics {
					s.mockRepository.EXPECT().
						GetMetrics(gomock.Any()).
						Return(nil, errors.New(errGetMetrics))
				} else {
					s.mockRepository.EXPECT().
						GetMetrics(gomock.Any()).
						Return(nil, &pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				}
			} else {
				s.mockRepository.EXPECT().
					GetMetrics(gomock.Any()).
					Return(test.want.response, nil)
			}
			resp, respBody := s.RequestTest(http.MethodGet, "/", "", test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				s.Equal(
					fmt.Sprintf(
						"%s = %d\n%s = %.2f\n",
						test.want.response[0].ID, *test.want.response[0].Delta,
						test.want.response[1].ID, *test.want.response[1].Value),
					string(respBody))
			}
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_GetMetricValueHandlerJSON() {
	type want struct {
		statusCode int
		err        string
		response   *metric.Metric
	}

	const (
		errGetMetrics = "errGetMetrics"
	)
	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)
	tests := []struct {
		name        string
		want        want
		body        string
		contentType string
	}{
		{
			name:        "bad content type",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "error decode request body",
			contentType: "application/json",
			body:        `bad_body`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "bad metric type",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "badType",
				"value": 13.5
			}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "db is not available",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "error get metric from db",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errGetMetrics,
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "test get counter ok",
			contentType: "application/json",
			body: `{
				"id": "cou",
				"type": "counter"
			}`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "cou",
					MType: MetricTypeCounter,
					Delta: &intValue},
			},
		},
		{
			name:        "test get gauge ok",
			contentType: "application/json",
			body: `{
				"id": "gau",
				"type": "gauge"
			}`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "gau",
					MType: MetricTypeGauge,
					Value: &floatValue},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK && test.want.statusCode != http.StatusBadRequest {
				if test.want.err == errGetMetrics {
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New(errGetMetrics))
				} else {
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, &pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				}
			} else if test.want.statusCode == http.StatusOK {
				s.mockRepository.EXPECT().
					GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(test.want.response, nil)
			}
			resp, respBody := s.RequestTest(http.MethodPost, "/value", test.body, test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				var actualMetric metric.Metric
				s.Require().NoError(json.Unmarshal(respBody, &actualMetric))
				s.Assert().Equal(test.want.response.ID, actualMetric.ID)
				s.Assert().Equal(test.want.response.MType, actualMetric.MType)
				if test.want.response.MType == MetricTypeCounter {
					s.Assert().Equal(*test.want.response.Delta, *actualMetric.Delta)
				} else {
					s.Assert().Equal(*test.want.response.Value, *actualMetric.Value)
				}
			}
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_GetMetricValueHandler() {
	type want struct {
		statusCode    int
		err           string
		response      *metric.Metric
		responseValue string
	}

	const (
		errGetMetrics = "errGetMetrics"
	)
	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)
	tests := []struct {
		name        string
		want        want
		contentType string
		mType       string
		mName       string
	}{
		{
			name:        "bad metric type",
			contentType: "text/plain",
			mType:       "badMType",
			mName:       "test",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "db is not available",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "test",
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "error get metric from db",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "test",
			want: want{
				err:        errGetMetrics,
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "test get counter ok",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "test",
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "test",
					MType: MetricTypeCounter,
					Delta: &intValue},
				responseValue: "1",
			},
		},
		{
			name:        "test get gauge ok",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "test",
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "test",
					MType: MetricTypeGauge,
					Value: &floatValue},
				responseValue: "1.24",
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK && test.want.statusCode != http.StatusBadRequest {
				if test.want.err == errGetMetrics {
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New(errGetMetrics))
				} else {
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, &pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				}
			} else if test.want.statusCode == http.StatusOK {
				s.mockRepository.EXPECT().
					GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(test.want.response, nil)
			}
			resp, respBody := s.RequestTest(http.MethodGet, fmt.Sprintf("/value/%s/%s", test.mType, test.mName), "", test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				s.Assert().Equal(test.want.responseValue, string(respBody))
			}
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_UpdateMetricHandlerJSON() {
	type want struct {
		statusCode int
		err        string
		response   *metric.Metric
	}

	const (
		errConnUpdateMetric = "errConnUpdateMetric"
		errUpdateMetric     = "errUpdateMetric"
		errOSDumpMetric     = "errOSDumpMetric"
		errDumpMetric       = "errDumpMetric"
		errConnGetMetric    = "errConnGetMetric"
	)
	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)
	tests := []struct {
		name        string
		want        want
		body        string
		contentType string
	}{
		{
			name:        "bad content type",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "error decode request body",
			contentType: "application/json",
			body:        `bad_body`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "bad metric type",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "badType",
				"value": 13.5
			}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty gauge metric value",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge"
			}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty counter metric value",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "counter"
			}`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "db is not available",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errConnUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "db error update metric",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error OS dump metric",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errOSDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error dump metric",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error get metric: db is not available",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				err:        errConnGetMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error get metric",
			contentType: "application/json",
			body: `{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}`,
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "test update counter ok",
			contentType: "application/json",
			body: `{
				"id": "cou",
				"delta": 1,
				"type": "counter"
			}`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "cou",
					MType: MetricTypeCounter,
					Delta: &intValue},
			},
		},
		{
			name:        "test update gauge ok",
			contentType: "application/json",
			body: `{
				"id": "gau",
				"value": 1.24,
				"type": "gauge"
			}`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "gau",
					MType: MetricTypeGauge,
					Value: &floatValue},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK && test.want.statusCode != http.StatusBadRequest {
				if test.want.err == errConnUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(&pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				} else if test.want.err == errUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(errors.New(errUpdateMetric))
				} else if test.want.err == errOSDumpMetric || test.want.err == errDumpMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(nil)
					if test.want.err == errOSDumpMetric {
						pathError := &os.PathError{
							Op:   "open",
							Path: "nonexistent_file.txt",
							Err:  os.ErrNotExist,
						}
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(pathError).Times(s.metricRouter.RetryCount + 1)
					} else {
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(errors.New("error dump metric"))
					}
				} else if test.want.err == errConnGetMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						Dump(gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, &pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				} else {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						Dump(gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New("error get metric"))
				}
			} else if test.want.statusCode == http.StatusOK {
				s.mockRepository.EXPECT().
					UpdateMetric(gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockRepository.EXPECT().
					Dump(gomock.Any()).
					Return(nil)
				s.mockRepository.EXPECT().
					GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(test.want.response, nil)
			}
			resp, respBody := s.RequestTest(http.MethodPost, "/update", test.body, test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				fmt.Println(string(respBody))
				var actualMetric metric.Metric
				s.Require().NoError(json.Unmarshal(respBody, &actualMetric))
				s.Assert().Equal(test.want.response.ID, actualMetric.ID)
				s.Assert().Equal(test.want.response.MType, actualMetric.MType)
				if test.want.response.MType == MetricTypeCounter {
					s.Assert().Equal(*test.want.response.Delta, *actualMetric.Delta)
				} else {
					s.Assert().Equal(*test.want.response.Value, *actualMetric.Value)
				}
			}
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_UpdateMetricHandler() {
	type want struct {
		statusCode int
		err        string
		response   *metric.Metric
	}

	const (
		errConnUpdateMetric = "errConnUpdateMetric"
		errUpdateMetric     = "errUpdateMetric"
		errOSDumpMetric     = "errOSDumpMetric"
		errDumpMetric       = "errDumpMetric"
	)

	tests := []struct {
		name        string
		want        want
		mType       string
		mName       string
		mValue      string
		contentType string
	}{
		{
			name:        "empty metric name",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "",
			mValue:      "1.24",
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "bad metric type",
			contentType: "text/plain",
			mType:       "badType",
			mName:       "gau",
			mValue:      "1.24",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "bad gauge metric value",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "gau",
			mValue:      "badValue",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "bad counter metric value",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "badValue",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "db is not available",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "1",
			want: want{
				err:        errConnUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error update metric db",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "1",
			want: want{
				err:        errUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error OS dump metric",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "1",
			want: want{
				err:        errOSDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error dump metric",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "1",
			want: want{
				err:        errDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "test update counter ok",
			contentType: "text/plain",
			mType:       "counter",
			mName:       "cou",
			mValue:      "1",
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:        "test update gauge ok",
			contentType: "text/plain",
			mType:       "gauge",
			mName:       "gau",
			mValue:      "1.24",
			want: want{
				statusCode: http.StatusOK,
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK && test.want.statusCode != http.StatusBadRequest {
				if test.want.err == errUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(errors.New(errUpdateMetric))
				} else if test.want.err == errConnUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(&pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				} else if test.want.err == errOSDumpMetric || test.want.err == errDumpMetric {
					s.mockRepository.EXPECT().
						UpdateMetric(gomock.Any(), gomock.Any()).
						Return(nil)
					if test.want.err == errOSDumpMetric {
						pathError := &os.PathError{
							Op:   "open",
							Path: "nonexistent_file.txt",
							Err:  os.ErrNotExist,
						}
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(pathError).Times(s.metricRouter.RetryCount + 1)
					} else {
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(errors.New("error dump metric"))
					}
				}
			} else if test.want.statusCode == http.StatusOK {
				s.mockRepository.EXPECT().
					UpdateMetric(gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockRepository.EXPECT().
					Dump(gomock.Any()).
					Return(nil)
			}

			path := ""
			if test.mName == "" {
				path = fmt.Sprintf("/update/%s/%s", test.mType, test.mValue)
			} else {
				path = fmt.Sprintf("/update/%s/%s/%s", test.mType, test.mName, test.mValue)
			}
			resp, _ := s.RequestTest(http.MethodPost, path, "", test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
		})
	}
}

func (s *MetricRouterTestSuite) TestMetricRouter_UpdateMetricsHandlerJSON() {
	type want struct {
		statusCode int
		err        string
		response   *metric.Metric
	}

	const (
		errConnUpdateMetric = "errConnUpdateMetric"
		errUpdateMetric     = "errUpdateMetric"
		errOSDumpMetric     = "errOSDumpMetric"
		errDumpMetric       = "errDumpMetric"
		errConnGetMetric    = "errConnGetMetric"
	)
	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)
	tests := []struct {
		name        string
		want        want
		body        string
		contentType string
	}{
		{
			name:        "bad content type",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "error decode request body",
			contentType: "application/json",
			body:        `bad_body`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "bad metric type",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "badType",
				"value": 13.5
			}]`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty gauge metric value",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge"
			}]`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "empty counter metric value",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "counter"
			}]`,
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "db is not available",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				err:        errConnUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "db error update metric",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				err:        errUpdateMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error OS dump metric",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				err:        errOSDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error dump metric",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				err:        errDumpMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error get metric: db is not available",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				err:        errConnGetMetric,
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "error get metric",
			contentType: "application/json",
			body: `[{
				"id": "test",
				"type": "gauge",
				"value": 13.5
			}]`,
			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "test update counter ok",
			contentType: "application/json",
			body: `[{
				"id": "cou",
				"delta": 1,
				"type": "counter"
			}]`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "cou",
					MType: MetricTypeCounter,
					Delta: &intValue},
			},
		},
		{
			name:        "test update gauge ok",
			contentType: "application/json",
			body: `[{
				"id": "gau",
				"value": 1.24,
				"type": "gauge"
			}]`,
			want: want{
				statusCode: http.StatusOK,
				response: &metric.Metric{
					ID:    "gau",
					MType: MetricTypeGauge,
					Value: &floatValue},
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			if test.want.statusCode != http.StatusOK && test.want.statusCode != http.StatusBadRequest {
				if test.want.err == errConnUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(&pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				} else if test.want.err == errUpdateMetric {
					s.mockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(errors.New(errUpdateMetric))
				} else if test.want.err == errOSDumpMetric || test.want.err == errDumpMetric {
					s.mockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(nil)
					if test.want.err == errOSDumpMetric {
						pathError := &os.PathError{
							Op:   "open",
							Path: "nonexistent_file.txt",
							Err:  os.ErrNotExist,
						}
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(pathError).Times(s.metricRouter.RetryCount + 1)
					} else {
						s.mockRepository.EXPECT().
							Dump(gomock.Any()).
							Return(errors.New("error dump metric"))
					}
				} else if test.want.err == errConnGetMetric {
					s.mockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						Dump(gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, &pgconn.PgError{Code: pgerrcode.ConnectionException}).Times(s.metricRouter.RetryCount + 1)
				} else {
					s.mockRepository.EXPECT().
						UpdateMetrics(gomock.Any(), gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						Dump(gomock.Any()).
						Return(nil)
					s.mockRepository.EXPECT().
						GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New("error get metric"))
				}
			} else if test.want.statusCode == http.StatusOK {
				s.mockRepository.EXPECT().
					UpdateMetrics(gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockRepository.EXPECT().
					Dump(gomock.Any()).
					Return(nil)
				s.mockRepository.EXPECT().
					GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(test.want.response, nil)
			}
			resp, respBody := s.RequestTest(http.MethodPost, "/updates", test.body, test.contentType, false, false)
			s.Assert().Equal(test.want.statusCode, resp.StatusCode)
			if test.want.statusCode == http.StatusOK {
				fmt.Println(string(respBody))
				var actualMetric []*metric.Metric
				s.Require().NoError(json.Unmarshal(respBody, &actualMetric))
				s.Assert().Equal(test.want.response.ID, actualMetric[0].ID)
				s.Assert().Equal(test.want.response.MType, actualMetric[0].MType)
				if test.want.response.MType == MetricTypeCounter {
					s.Assert().Equal(*test.want.response.Delta, *actualMetric[0].Delta)
				} else {
					s.Assert().Equal(*test.want.response.Value, *actualMetric[0].Value)
				}
			}
		})
	}
}

func BenchmarkPingDBHandler(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	b.ResetTimer()
	b.Run("ping", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				Ping(gomock.Any()).
				Return(nil).AnyTimes()
			resp, err := http.Get(server.URL + "/ping")
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkRootHandler(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue   int64 = 1
		floatValue       = 1.24
	)
	response := []*metric.Metric{
		{
			ID:    "counter",
			MType: MetricTypeCounter,
			Delta: &intValue,
		},
		{
			ID:    "gauge",
			MType: MetricTypeGauge,
			Value: &floatValue,
		},
	}

	b.ResetTimer()
	b.Run("root", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				GetMetrics(gomock.Any()).
				Return(response, nil)
			resp, err := http.Get(server.URL + "/")
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkGetMetricValueHandlerJSON(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue int64 = 1
	)
	response := &metric.Metric{
		ID:    "cou",
		MType: MetricTypeCounter,
		Delta: &intValue}

	body := []byte(`{
				"id": "cou",
				"type": "counter"
			}`)
	b.ResetTimer()
	b.Run("value json", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(response, nil).AnyTimes()
			resp, err := http.Post(server.URL+"/value", "application/json", bytes.NewBuffer(body))
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkGetMetricValueHandler(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue int64 = 1
	)
	response := &metric.Metric{
		ID:    "cou",
		MType: MetricTypeCounter,
		Delta: &intValue}

	mName := "cou"
	mType := "counter"

	b.ResetTimer()
	b.Run("value", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(response, nil).AnyTimes()
			resp, err := http.Get(server.URL + "/value/" + mType + "/" + mName)
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkUpdateMetricHandlerJSON(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue int64 = 1
	)
	response := &metric.Metric{
		ID:    "cou",
		MType: MetricTypeCounter,
		Delta: &intValue}

	body := []byte(`{
				"id": "cou",
				"delta": 1,
				"type": "counter"
			}`)
	b.ResetTimer()
	b.Run("update json", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				UpdateMetric(gomock.Any(), gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				Dump(gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(response, nil).AnyTimes()
			resp, err := http.Post(server.URL+"/update", "application/json", bytes.NewBuffer(body))
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkUpdateMetricHandler(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue int64 = 1
	)
	response := &metric.Metric{
		ID:    "cou",
		MType: MetricTypeCounter,
		Delta: &intValue}

	mName := "cou"
	mType := "counter"

	b.ResetTimer()
	b.Run("update", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				UpdateMetric(gomock.Any(), gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				Dump(gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(response, nil).AnyTimes()
			resp, err := http.Get(server.URL + "/update/" + mType + "/" + mName + "/1")
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}

func BenchmarkUpdateMetricsHandlerJSON(b *testing.B) {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}
	mockController := gomock.NewController(b)
	mockRepository := mock.NewMockRepository(mockController)
	metricRouter := NewMetricRouter(chiRouter, mockRepository, &serverConfig)
	server := httptest.NewServer(metricRouter.Router)
	defer server.Close()

	var (
		intValue int64 = 1
	)
	response := &metric.Metric{
		ID:    "cou",
		MType: MetricTypeCounter,
		Delta: &intValue}

	body := []byte(`[{
				"id": "cou",
				"delta": 1,
				"type": "counter"
			}]`)
	b.ResetTimer()
	b.Run("update metrics json", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			mockRepository.EXPECT().
				UpdateMetrics(gomock.Any(), gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				Dump(gomock.Any()).
				Return(nil).AnyTimes()
			mockRepository.EXPECT().
				GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(response, nil).AnyTimes()
			resp, err := http.Post(server.URL+"/updates", "application/json", bytes.NewBuffer(body))
			if err != nil {
				b.Fatalf("failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
	mockController.Finish()
}
