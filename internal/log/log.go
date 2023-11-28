package log

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func NewProductionLogger() (*zap.Logger, error) {
	const op = "new production logger"
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := config.Build()

	if err != nil {
		return nil, errorf(op, err)
	}

	return logger, nil
}

func errorf(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	const op = "logging response"
	size, err := w.ResponseWriter.Write(b)

	if err != nil {
		return size, errorf(op, err)
	}

	w.responseData.size += size
	return size, nil
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

func RequestResponseLogger(logger *zap.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		log := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			resp := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w,
				responseData:   resp,
			}

			h.ServeHTTP(&lw, r)

			duration := time.Since(start)
			logger.Info("",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Int("status", resp.status),
				zap.Duration("duration", duration),
				zap.Int("size", resp.size),
			)
		}
		return http.HandlerFunc(log)
	}
}
