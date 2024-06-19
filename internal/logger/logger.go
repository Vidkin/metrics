package logger

import (
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

var Log *zap.Logger = zap.NewNop()

func Initialize(logLevel string) error {
	lvl, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	Log = logger
	return nil
}

func LoggingHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
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

		Log.Info(
			"Request data",
			zap.String("method", req.Method),
			zap.String("URI", req.RequestURI),
			zap.Duration("duration", duration),
		)

		Log.Info(
			"Response data",
			zap.Int("status", rData.status),
			zap.Int("size", rData.size),
		)
	}
}
