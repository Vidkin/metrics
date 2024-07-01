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
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			cr, err := compress.NewCompressReader(r.Body)
			if err != nil {
				logger.Log.Error("error init compress reader", zap.Error(err))
				ow.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}
