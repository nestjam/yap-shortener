package middleware

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	contentLengthHeader   = "Content-Length"
	gzipEncoding          = "gzip"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.Header().Del(contentLengthHeader)
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)

	if err != nil {
		return 0, fmt.Errorf("write compressed: %w", err)
	}

	return n, nil
}

type gzipRequestReader struct {
	io.ReadCloser
	gz *gzip.Reader
}

func newGzipRequestReader(r io.ReadCloser) (*gzipRequestReader, error) {
	gzipReader, err := gzip.NewReader(r)

	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}

	return &gzipRequestReader{
		r,
		gzipReader,
	}, nil
}

func (g *gzipRequestReader) Read(p []byte) (int, error) {
	n, err := g.gz.Read(p)

	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	if err != nil {
		return 0, fmt.Errorf("read compressed: %w", err)
	}

	return n, nil
}

func (g *gzipRequestReader) Close() error {
	if err := g.ReadCloser.Close(); err != nil {
		return fmt.Errorf("close reader: %w", err)
	}

	if err := g.gz.Close(); err != nil {
		return fmt.Errorf("close gzip reader: %w", err)
	}

	return nil
}

func ResponseEncoder(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get(acceptEncodingHeader), gzipEncoding) {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set(contentEncodingHeader, gzipEncoding)

		gz, ok := gzipWriterPool.Get().(*gzip.Writer)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer gzipWriterPool.Put(gz)

		gz.Reset(w)
		defer func() {
			_ = gz.Close()
		}()

		h.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func RequestDecoder(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding := r.Header.Get(contentEncodingHeader)
		if strings.Contains(encoding, gzipEncoding) {
			reader, err := newGzipRequestReader(r.Body)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = reader

			defer func() {
				_ = reader.Close()
			}()
		}

		h.ServeHTTP(w, r)
	})
}
