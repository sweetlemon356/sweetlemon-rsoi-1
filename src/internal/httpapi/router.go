package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/swaggest/swgui/v5emb"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func NewRouter(handler *Handler, healthChecker HealthChecker) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := healthChecker.Ping(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "health check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Message: "database unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/v1/persons", handler.List)
	mux.HandleFunc("POST /api/v1/persons", handler.Create)
	mux.HandleFunc("GET /api/v1/persons/{id}", handler.Get)
	mux.HandleFunc("PATCH /api/v1/persons/{id}", handler.Update)
	mux.HandleFunc("DELETE /api/v1/persons/{id}", handler.Delete)

	mux.HandleFunc("GET /openapi.yaml", serveOpenAPI)
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	mux.Handle("GET /swagger/", v5emb.New("Person Service API", "/openapi.yaml", "/swagger/"))

	return logMiddleware(recoverMiddleware(mux))
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}
