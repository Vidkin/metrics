package middleware

import (
	"github.com/Vidkin/metrics/internal/logger"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (rw *loggingResponseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.responseData.size += size
	return size, err
}

func (rw *loggingResponseWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.responseData.status = statusCode
}

func Logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		rData := &responseData{
			status: 0,
			size:   0,
		}
		loggingRW := loggingResponseWriter{
			responseData:   rData,
			ResponseWriter: rw,
		}

		h.ServeHTTP(&loggingRW, req)
		duration := time.Since(startTime)

		logger.Log.Info(
			"Request data",
			zap.String("method", req.Method),
			zap.String("URI", req.RequestURI),
			zap.Duration("duration", duration),
		)

		logger.Log.Info(
			"Response data",
			zap.Int("status", rData.status),
			zap.Int("size", rData.size),
		)
	})
}
