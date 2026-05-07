// Package api handles HTTP routing and request/response processing.
package api

import (
	"encoding/json"
	"net/http"
)

// SuccessResponse is the standard success envelope.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeSuccess writes a successful JSON response.
func writeSuccess(w http.ResponseWriter, status int, data interface{}) {
	writeJSON(w, status, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// writeError writes an error JSON response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Success: false,
		Error:   message,
	})
}
