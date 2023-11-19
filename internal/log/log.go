package log

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

var (
	Logger *zap.Logger        = zap.NewNop()
	sugar  *zap.SugaredLogger = Logger.Sugar()
)

func Initialize(level string) error {
	const op = "initializing logger"

	lvl, err := zap.ParseAtomicLevel(level)

	if err != nil {
		return errorf(op, err)
	}

	config := zap.NewProductionConfig()
	config.Level = lvl
	logger, err := config.Build()

	if err != nil {
		return errorf(op, err)
	}

	Logger = logger
	sugar = logger.Sugar()
	return nil
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

func RequestResponseLogger(h http.Handler) http.Handler {
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
		sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", resp.status,
			"duration", duration,
			"size", resp.size,
		)
	}
	return http.HandlerFunc(log)
}
