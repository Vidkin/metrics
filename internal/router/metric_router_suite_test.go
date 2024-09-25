package router

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/config"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/internal/repository/mock"
)

type MetricRouterTestSuite struct {
	suite.Suite
	metricRouter   *MetricRouter
	key            string
	mockController *gomock.Controller
	mockRepository *mock.MockRepository
	server         *httptest.Server
}

func (s *MetricRouterTestSuite) SetupTest() {
	chiRouter := chi.NewRouter()
	serverConfig := config.ServerConfig{StoreInterval: 0, RetryCount: 2}

	s.mockController = gomock.NewController(s.T())
	s.mockRepository = mock.NewMockRepository(s.mockController)
	s.metricRouter = NewMetricRouter(chiRouter, s.mockRepository, &serverConfig)
	s.server = httptest.NewServer(s.metricRouter.Router)
}

func (s *MetricRouterTestSuite) TearDownTest() {
	s.server.Close()
}

func (s *MetricRouterTestSuite) RequestTest(method, path string, body string, contentType string, acceptEncoding bool, contentEncoding bool) (*http.Response, []byte) {
	bodyBytes := []byte(body)
	req, err := http.NewRequest(method, s.server.URL+path, bytes.NewBuffer(bodyBytes))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", contentType)
	if acceptEncoding == true {
		req.Header.Set("Accept-Encoding", "gzip")
	} else {
		req.Header.Set("Accept-Encoding", "")
	}
	if contentEncoding == true {
		req.Header.Set("Content-Encoding", "gzip")
	} else {
		req.Header.Set("Content-Encoding", "")
	}

	resp, err := s.server.Client().Do(req)
	s.Require().NoError(err)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Error("error close request body", zap.Error(err))
		}
	}(resp.Body)

	if acceptEncoding {
		dec, err := gzip.NewReader(resp.Body)
		s.Require().NoError(err)
		respBody, err := io.ReadAll(dec)
		s.Require().NoError(err)
		return resp, respBody
	} else {
		respBody, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		return resp, respBody
	}
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(MetricRouterTestSuite))
}
