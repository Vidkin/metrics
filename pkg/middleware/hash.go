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

type (
	hashResponseWriter struct {
		http.ResponseWriter
		Key        string
		HashSHA256 string
		statusCode int
		written    bool
	}
)

func (rw *hashResponseWriter) Write(data []byte) (int, error) {
	h := hash.GetHashSHA256(rw.Key, data)
	hEnc := base64.StdEncoding.EncodeToString(h)
	rw.HashSHA256 = hEnc
	rw.Header().Set("HashSHA256", rw.HashSHA256)
	return rw.ResponseWriter.Write(data)
}

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

				var buf bytes.Buffer
				tee := io.TeeReader(r.Body, &buf)
				body, err := io.ReadAll(tee)
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
				r.Body = io.NopCloser(&buf)
			}
			hashRW := hashResponseWriter{
				ResponseWriter: w,
				Key:            key,
			}
			next.ServeHTTP(&hashRW, r)
		})
	}
}
