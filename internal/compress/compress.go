package compress

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	gzipEncoding          = "gzip"
)

type compressWriter struct {
	http.ResponseWriter
	gw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w,
		gzip.NewWriter(w),
	}
}

func (c *compressWriter) Write(p []byte) (int, error) {
	// return c.gw.Write(p)
	n, err := c.gw.Write(p)

	if err != nil {
		return 0, fmt.Errorf("write compressed: %w", err)
	}

	return n, nil
}

func (c *compressWriter) Close() error {
	err := c.gw.Close()

	if err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return nil
}

type compressReader struct {
	r  io.ReadCloser
	gr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	gr, err := gzip.NewReader(r)

	if err != nil {
		return nil, fmt.Errorf("new reader: %w", err)
	}

	return &compressReader{
		r,
		gr,
	}, nil
}

func (c *compressReader) Read(p []byte) (int, error) {
	// return c.gr.Read(p)
	n, err := c.gr.Read(p)

	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	if err != nil {
		return 0, fmt.Errorf("read compressed: %w", err)
	}

	return n, nil
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return fmt.Errorf("close reader: %w", err)
	}

	if err := c.gr.Close(); err != nil {
		return fmt.Errorf("close compress reader: %w", err)
	}

	return nil
}

func Compressor(h http.Handler) http.Handler {
	compress := func(w http.ResponseWriter, r *http.Request) {
		contentEncodings := r.Header.Get(contentEncodingHeader)
		if strings.Contains(contentEncodings, gzipEncoding) {
			cr, err := newCompressReader(r.Body)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer func() {
				_ = cr.Close()
			}()

			r.Body = cr
		}

		acceptEncodings := r.Header.Get(acceptEncodingHeader)
		if !strings.Contains(acceptEncodings, gzipEncoding) {
			h.ServeHTTP(w, r)
			return
		}

		cw := newCompressWriter(w)
		defer func() {
			_ = cw.Close()
		}()

		w.Header().Add(contentEncodingHeader, gzipEncoding)

		h.ServeHTTP(cw, r)
	}
	return http.HandlerFunc(compress)
}
