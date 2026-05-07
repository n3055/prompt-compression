package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/n3055/backend-project/internal/service"
	"github.com/n3055/backend-project/internal/store"
)

// Handler holds HTTP handlers and their dependencies.
type Handler struct {
	svc *service.ConversationService
	log *slog.Logger
}

// NewHandler creates a new handler with injected dependencies.
func NewHandler(svc *service.ConversationService, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// ChatRequest is the expected JSON body for POST /api/v1/chat.
type ChatRequest struct {
	SessionID    string `json:"session_id"`    // Optional — omit to create new session.
	Instructions string `json:"instructions"`  // System prompt / instructions.
	Query        string `json:"query"`         // User's current message.
}

// HealthCheck handles GET /health.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "prompt-compression-engine",
	})
}

// Chat handles POST /api/v1/chat.
// This is the main endpoint — it processes a message through the compression pipeline.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	result, err := h.svc.ProcessMessage(req.SessionID, req.Instructions, req.Query)
	if err != nil {
		h.log.Error("failed to process message", "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(w, http.StatusOK, result)
}

// GetSession handles GET /api/v1/sessions/{id}.
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session ID is required")
		return
	}

	info, err := h.svc.GetSessionInfo(id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "session not found: "+id)
			return
		}
		h.log.Error("failed to get session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeSuccess(w, http.StatusOK, info)
}

// GetHistory handles GET /api/v1/sessions/{id}/history.
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session ID is required")
		return
	}

	history, err := h.svc.GetHistory(id)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "session not found: "+id)
			return
		}
		h.log.Error("failed to get history", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeSuccess(w, http.StatusOK, map[string]interface{}{
		"session_id": id,
		"history":    history,
	})
}

// DeleteSession handles DELETE /api/v1/sessions/{id}.
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session ID is required")
		return
	}

	if err := h.svc.DeleteSession(id); err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "session not found: "+id)
			return
		}
		h.log.Error("failed to delete session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeSuccess(w, http.StatusOK, map[string]string{
		"message": "session deleted",
	})
}

// isNotFound checks if an error is a store.ErrNotFound.
func isNotFound(err error) bool {
	var notFound *store.ErrNotFound
	return errors.As(err, &notFound)
}
