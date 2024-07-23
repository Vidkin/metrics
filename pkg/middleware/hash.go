package middleware

import (
	"bytes"
	"encoding/base64"
	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/pkg/hash"
	"go.uber.org/zap"
	"io"
	"net/http"
)

func Hash(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hEnc := r.Header.Get("HashSHA256")
			if hEnc != "" {
				hashA, err := base64.StdEncoding.DecodeString(hEnc)
				if err != nil {
					logger.Log.Error("error decode hash from base64 string", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					logger.Log.Error("error read request body", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				hashB := hash.GetHashSHA256(key, body)
				if !bytes.Equal(hashA, hashB) {
					logger.Log.Error("hashes don't match")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}
			next.ServeHTTP(w, r)
		})
	}
}
