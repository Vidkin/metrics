package middleware

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
	"github.com/Vidkin/metrics/pkg/hash"
)

type hashResponseWriter struct {
	http.ResponseWriter
	Key        string
	HashSHA256 string
}

// Hash is an HTTP middleware function that validates the integrity of incoming
// request bodies using SHA-256 hashes.
//
// Parameters:
//   - key: A string that serves as a key in the hash computation. This key is
//     used to generate the SHA-256 hash of the request body.
//
// Returns:
//   - A function that takes an http.Handler and returns a new http.Handler
//     that includes the hash validation logic.
func Hash(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hEnc := r.Header.Get("HashSHA256")
			if hEnc == "" {
				logger.Log.Error("client does not provide any hash")
				w.WriteHeader(http.StatusBadRequest)
				return
			} else {
				hashA, err := base64.StdEncoding.DecodeString(hEnc)
				if err != nil {
					logger.Log.Error("error decode hash from base64 string", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var buf bytes.Buffer
				tee := io.TeeReader(r.Body, &buf)

				defer func() {
					if err := r.Body.Close(); err != nil {
						logger.Log.Error("error close reader body", zap.Error(err))
					}
				}()

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
				defer func() {
					if err := r.Body.Close(); err != nil {
						logger.Log.Error("error close reader body", zap.Error(err))
					}
				}()
			}

			hashRW := hashResponseWriter{
				ResponseWriter: w,
				Key:            key,
			}
			next.ServeHTTP(hashRW, r)
		})
	}
}
