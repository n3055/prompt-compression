package api

import (
	"log/slog"
	"net/http"

	"github.com/n3055/backend-project/internal/middleware"
)

// NewRouter creates the HTTP router with all routes and middleware wired up.
// Uses Go 1.22+ enhanced ServeMux with method+pattern routing.
func NewRouter(handler *Handler, log *slog.Logger, limiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()

	// --- Routes ---

	// Health check (no auth/rate-limit needed).
	mux.HandleFunc("GET /health", handler.HealthCheck)

	// Chat endpoint — the main API.
	mux.HandleFunc("POST /api/v1/chat", handler.Chat)

	// Session management.
	mux.HandleFunc("GET /api/v1/sessions/{id}", handler.GetSession)
	mux.HandleFunc("GET /api/v1/sessions/{id}/history", handler.GetHistory)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", handler.DeleteSession)

	// --- Middleware Chain (applied in reverse order) ---
	// Request flow: CORS → RateLimit → RequestID → Recovery → Logging → Handler
	var h http.Handler = mux
	h = middleware.Logging(log)(h)
	h = middleware.Recovery(log)(h)
	h = middleware.RequestID(h)
	h = middleware.RateLimit(limiter)(h)
	h = middleware.CORS(h)

	return h
}
