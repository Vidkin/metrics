// Package compress provides a Reader and Writer for gzip-compressed data.
// When writing a response, it automatically adds header “Content-Encoding”: “gzip”.
package compress

import (
	"compress/gzip"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/Vidkin/metrics/internal/logger"
)

// Writer is a wrapper over http.ResponseWriter,
// which adds support for data compression using gzip.
type Writer struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// NewCompressWriter creates a new instance of Writer,
// initializing it with the given http.ResponseWriter.
// This constructor also creates a new gzip.Writer,
// which will be used to compress the data before sending it.
//
// Parameters:
// - w: the http.ResponseWriter that will be used to send the compressed data.
//
// Returns:
// - Pointer to a new Writer ready to be used.
func NewCompressWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header returns the header map that will be sent by the ResponseWriter.
// It allows setting HTTP headers before writing the response.
func (c *Writer) Header() http.Header {
	return c.w.Header()
}

// Write writes the compressed data to the underlying ResponseWriter.
// It sets the "Content-Encoding" header to "gzip" before writing the data.
//
// Parameters:
// - p: a byte slice containing the data to be written.
//
// Returns:
// - The number of bytes written and any error encountered.
func (c *Writer) Write(p []byte) (int, error) {
	c.w.Header().Set("Content-Encoding", "gzip")
	return c.zw.Write(p)
}

// WriteHeader sends an HTTP response header with the provided status code.
// It allows setting the status code for the response.
func (c *Writer) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

// Close closes the gzip.Writer, flushing any buffered data to the underlying writer.
// It should be called to ensure all data is sent before the response is completed.
func (c *Writer) Close() error {
	return c.zw.Close()
}

// Reader is a wrapper over io.ReadCloser,
// which adds support for reading gzip-compressed data.
type Reader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader creates a new instance of Reader,
// initializing it with the given io.ReadCloser.
// This constructor also creates a new gzip.Reader,
// which will be used to decompress the data being read.
//
// Parameters:
// - r: the io.ReadCloser that will be used to read the compressed data.
//
// Returns:
// - Pointer to a new Reader ready to be used, or an error if initialization fails.
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

// Read reads the decompressed data from the gzip.Reader into the provided byte slice.
//
// Parameters:
// - p: a byte slice to hold the read data.
//
// Returns:
// - The number of bytes read and any error encountered.
func (c Reader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close closes the gzip.Reader and the underlying ReadCloser.
// It should be called to release resources associated with the reader.
func (c *Reader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
