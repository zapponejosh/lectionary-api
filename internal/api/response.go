package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains error details.
type ErrorInfo struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ResponseWriter wraps response writing with consistent error handling.
// It logs errors but doesn't expose internal details to clients.
type ResponseWriter struct {
	logger *slog.Logger
}

// NewResponseWriter creates a new ResponseWriter with the given logger.
func NewResponseWriter(logger *slog.Logger) *ResponseWriter {
	return &ResponseWriter{logger: logger}
}

// WriteJSON writes a JSON response with the given status code.
// If encoding fails, it logs the error and writes a 500 response.
func (rw *ResponseWriter) WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// At this point headers are already sent, so we can only log the error
		if rw.logger != nil {
			rw.logger.Error("failed to encode JSON response",
				slog.Any("error", err),
				slog.Int("status", status),
			)
		}
	}
}

// WriteSuccess writes a successful JSON response.
func (rw *ResponseWriter) WriteSuccess(w http.ResponseWriter, data interface{}) {
	rw.WriteJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error JSON response.
func (rw *ResponseWriter) WriteError(w http.ResponseWriter, status int, message string, code string) {
	rw.WriteJSON(w, status, Response{
		Success: false,
		Error: &ErrorInfo{
			Message: message,
			Code:    code,
		},
	})
}

// WriteNotFound writes a 404 Not Found response.
func (rw *ResponseWriter) WriteNotFound(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusNotFound, message, "NOT_FOUND")
}

// WriteBadRequest writes a 400 Bad Request response.
func (rw *ResponseWriter) WriteBadRequest(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusBadRequest, message, "BAD_REQUEST")
}

// WriteInternalError writes a 500 Internal Server Error response.
func (rw *ResponseWriter) WriteInternalError(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusInternalServerError, message, "INTERNAL_ERROR")
}

// WriteUnauthorized writes a 401 Unauthorized response.
func (rw *ResponseWriter) WriteUnauthorized(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusUnauthorized, message, "UNAUTHORIZED")
}

// WriteConflict writes a 409 Conflict response.
func (rw *ResponseWriter) WriteConflict(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusConflict, message, "CONFLICT")
}

// WriteServiceUnavailable writes a 503 Service Unavailable response.
func (rw *ResponseWriter) WriteServiceUnavailable(w http.ResponseWriter, message string) {
	rw.WriteError(w, http.StatusServiceUnavailable, message, "SERVICE_UNAVAILABLE")
}

// =============================================================================
// Standalone functions for backward compatibility and simpler use cases
// =============================================================================

// WriteJSON writes a JSON response with the given status code.
// Deprecated: Use ResponseWriter.WriteJSON for proper error logging.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a successful JSON response.
// Deprecated: Use ResponseWriter.WriteSuccess for proper error logging.
func WriteSuccess(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error JSON response.
// Deprecated: Use ResponseWriter.WriteError for proper error logging.
func WriteError(w http.ResponseWriter, status int, message string, code ...string) error {
	errInfo := ErrorInfo{
		Message: message,
	}
	if len(code) > 0 {
		errInfo.Code = code[0]
	}

	return WriteJSON(w, status, Response{
		Success: false,
		Error:   &errInfo,
	})
}

// WriteNotFound writes a 404 Not Found response.
// Deprecated: Use ResponseWriter.WriteNotFound for proper error logging.
func WriteNotFound(w http.ResponseWriter, message string) error {
	return WriteError(w, http.StatusNotFound, message, "NOT_FOUND")
}

// WriteBadRequest writes a 400 Bad Request response.
// Deprecated: Use ResponseWriter.WriteBadRequest for proper error logging.
func WriteBadRequest(w http.ResponseWriter, message string) error {
	return WriteError(w, http.StatusBadRequest, message, "BAD_REQUEST")
}

// WriteInternalError writes a 500 Internal Server Error response.
// Deprecated: Use ResponseWriter.WriteInternalError for proper error logging.
func WriteInternalError(w http.ResponseWriter, message string) error {
	return WriteError(w, http.StatusInternalServerError, message, "INTERNAL_ERROR")
}

// WriteUnauthorized writes a 401 Unauthorized response.
// Deprecated: Use ResponseWriter.WriteUnauthorized for proper error logging.
func WriteUnauthorized(w http.ResponseWriter, message string) error {
	return WriteError(w, http.StatusUnauthorized, message, "UNAUTHORIZED")
}
func WriteForbidden(w http.ResponseWriter, message string) error {
	return WriteError(w, http.StatusForbidden,
		message, "FORBIDDEN")
}
