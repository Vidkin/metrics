package middleware

import (
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/pkg/compress"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func Gzip(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w
		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			cw := compress.NewCompressWriter(w)
			ow = cw
			defer func() {
				if err := cw.Close(); err != nil {
					logger.Log.Error("error close compress writer", zap.Error(err))
				}
			}()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			defer func() {
				if err := r.Body.Close(); err != nil {
					logger.Log.Error("error close body", zap.Error(err))
				}
			}()

			cr, err := compress.NewCompressReader(r.Body)
			if err != nil {
				logger.Log.Error("error init compress reader", zap.Error(err))
				ow.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func() {
				if err := cr.Close(); err != nil {
					logger.Log.Error("error close compress reader", zap.Error(err))
				}
			}()
		}

		h.ServeHTTP(ow, r)
	})
}
