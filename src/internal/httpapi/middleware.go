package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
)

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		slog.InfoContext(r.Context(), "request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", metrics.Code,
			"response_bytes", metrics.Written,
			"duration", metrics.Duration,
		)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseStarted := false
		wrapped := httpsnoop.Wrap(w, httpsnoop.Hooks{
			WriteHeader: func(nextWriteHeader httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return httpsnoop.WriteHeaderFunc(func(status int) {
					responseStarted = true
					nextWriteHeader(status)
				})
			},
			Write: func(nextWrite httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return httpsnoop.WriteFunc(func(body []byte) (int, error) {
					responseStarted = true
					return nextWrite(body)
				})
			},
		})

		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(r.Context(), "panic recovered", "panic", recovered)
				if !responseStarted {
					writeJSON(wrapped, http.StatusInternalServerError, errorResponse{Message: "internal server error"})
				}
			}
		}()
		next.ServeHTTP(wrapped, r)
	})
}
