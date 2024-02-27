package middleware

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

// Write выполняет запись данных в HTTP ответ и сохраняет информацию о размере данных.
func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	const op = "logging response"
	size, err := w.ResponseWriter.Write(b)

	if err != nil {
		return size, fmt.Errorf("%s: %w", op, err)
	}

	w.responseData.size += size
	return size, nil
}

// WriteHeader отправляет заголовок HTTP ответа с указанным кодом и сохраняет отправленый статус.
func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

// ResponseLogger возвращает посредника, который логирует сведения из HTTP ответа.
func ResponseLogger(logger *zap.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		methodCache := make(map[string]zap.Field)
		statusCache := make(map[int]zap.Field)

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

			if _, ok := methodCache[r.Method]; !ok {
				methodCache[r.Method] = zap.String("method", r.Method)
			}
			if _, ok := statusCache[resp.status]; !ok {
				statusCache[resp.status] = zap.Int("status", resp.status)
			}

			duration := time.Since(start)
			logger.Info("",
				zap.String("uri", r.RequestURI),
				methodCache[r.Method],
				statusCache[resp.status],
				zap.Duration("duration", duration),
				zap.Int("size", resp.size),
			)
		}
		return http.HandlerFunc(log)
	}
}
