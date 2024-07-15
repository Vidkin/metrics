package compress

import (
	"compress/gzip"
	"github.com/Vidkin/metrics/internal/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Writer struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewCompressWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *Writer) Header() http.Header {
	return c.w.Header()
}

func (c *Writer) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *Writer) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *Writer) Close() error {
	return c.zw.Close()
}

type Reader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func NewCompressReader(r io.ReadCloser) (*Reader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		logger.Log.Info("error init compress reader", zap.Error(err))
		return nil, err
	}

	return &Reader{
		r:  r,
		zr: zr,
	}, nil
}

func (c Reader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}
func (c *Reader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
