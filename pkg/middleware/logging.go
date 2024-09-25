package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
)

type (
	loggingResponseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *loggingResponseData
	}
)

// Write writes the data to the underlying ResponseWriter and updates the size
// of the response. It overrides the default Write method to keep track of the
// total size of the response body.
//
// Parameters:
// - data: A byte slice containing the data to be written to the response.
//
// Returns:
// - The number of bytes written and any error encountered during the write operation.
func (rw *loggingResponseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.responseData.size += size
	return size, err
}

// WriteHeader sends an HTTP response header with the provided status code.
// It overrides the default WriteHeader method to record the status code
// of the response.
//
// Parameters:
//   - statusCode: An integer representing the HTTP status code to be sent
//     in the response.
func (rw *loggingResponseWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.responseData.status = statusCode
}

// Logging is an HTTP middleware function that logs details about incoming
// requests and outgoing responses.
//
// This function wraps the provided HTTP handler and logs the HTTP method,
// request URI, response status code, response size, and the duration of
// the request processing. It uses a custom loggingResponseWriter to capture
// the response data.
//
// Parameters:
// - h: An http.Handler that will be wrapped by the Logging middleware.
//
// Returns:
//   - An http.Handler that includes the logging functionality for requests
//     and responses.
func Logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		rData := &loggingResponseData{
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
