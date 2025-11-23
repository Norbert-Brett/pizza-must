package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// respondWithError sends a structured error response
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithErrorDetails(w, statusCode, message, nil)
}

// respondWithErrorDetails sends a structured error response with additional details
func respondWithErrorDetails(w http.ResponseWriter, statusCode int, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:      http.StatusText(statusCode),
			Message:   message,
			Details:   details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// RespondWithValidationErrors sends validation error response
func RespondWithValidationErrors(w http.ResponseWriter, errors []ValidationError) {
	details := make(map[string]interface{})
	details["validation_errors"] = errors

	respondWithErrorDetails(w, http.StatusBadRequest, "validation failed", details)
}

// ErrorHandlingMiddleware catches panics and converts them to 500 errors
func ErrorHandlingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
					)

					respondWithError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
