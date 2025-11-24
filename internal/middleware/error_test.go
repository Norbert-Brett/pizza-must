package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: ordering-platform, Property 51: Errors have consistent structure
// Validates: Requirements 19.1
func TestProperty_ErrorsHaveConsistentStructure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("all error responses have consistent structure", prop.ForAll(
		func(message string) bool {
			// Use standard HTTP status codes that have defined text
			standardCodes := []int{
				http.StatusBadRequest,          // 400
				http.StatusUnauthorized,        // 401
				http.StatusForbidden,           // 403
				http.StatusNotFound,            // 404
				http.StatusConflict,            // 409
				http.StatusTooManyRequests,     // 429
				http.StatusInternalServerError, // 500
				http.StatusServiceUnavailable,  // 503
			}

			// Pick a random standard status code
			statusCode := standardCodes[len(message)%len(standardCodes)]

			// Ensure non-empty message
			if len(message) == 0 {
				message = "test error"
			}

			w := httptest.NewRecorder()
			RespondWithError(w, statusCode, message)

			// Check status code
			if w.Code != statusCode {
				return false
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				return false
			}

			// Parse response
			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Verify structure - all required fields must be present
			if response.Error.Code == "" {
				return false
			}
			if response.Error.Message != message {
				return false
			}
			if response.Error.Timestamp == "" {
				return false
			}

			// Verify timestamp is valid RFC3339
			if _, err := time.Parse(time.RFC3339, response.Error.Timestamp); err != nil {
				return false
			}

			return true
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that error responses include proper HTTP status codes
func TestProperty_ErrorStatusCodesAreCorrect(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("error responses use correct HTTP status codes", prop.ForAll(
		func(useCode int) bool {
			// Use standard HTTP status codes
			standardCodes := []int{
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusConflict,
				http.StatusTooManyRequests,
				http.StatusInternalServerError,
				http.StatusServiceUnavailable,
			}

			// Handle negative codes
			if useCode < 0 {
				useCode = -useCode
			}

			statusCode := standardCodes[useCode%len(standardCodes)]

			w := httptest.NewRecorder()
			RespondWithError(w, statusCode, "test error")

			// Status code should match what was requested
			return w.Code == statusCode
		},
		gen.Int(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that error responses with details include them in the structure
func TestProperty_ErrorDetailsAreIncluded(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("error responses with details include them", prop.ForAll(
		func(message string, detailKey string, detailValue string) bool {
			if message == "" {
				message = "test error"
			}
			if detailKey == "" {
				detailKey = "field"
			}
			if detailValue == "" {
				detailValue = "error detail"
			}

			details := map[string]interface{}{
				detailKey: detailValue,
			}

			w := httptest.NewRecorder()
			RespondWithErrorDetails(w, http.StatusBadRequest, message, details)

			// Parse response
			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Verify details are present
			if response.Error.Details == nil {
				return false
			}

			// Verify the detail we added is present
			if val, ok := response.Error.Details[detailKey]; !ok || val != detailValue {
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that validation errors are properly formatted
func TestProperty_ValidationErrorsHaveConsistentStructure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("validation errors have consistent structure", prop.ForAll(
		func(fieldName string, errorMessage string) bool {
			if fieldName == "" {
				fieldName = "testField"
			}
			if errorMessage == "" {
				errorMessage = "test error"
			}

			errors := []ValidationError{
				{
					Field:   fieldName,
					Message: errorMessage,
				},
			}

			w := httptest.NewRecorder()
			RespondWithValidationErrors(w, errors)

			// Check status code
			if w.Code != http.StatusBadRequest {
				return false
			}

			// Parse response
			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// Verify structure
			if response.Error.Code == "" {
				return false
			}
			if response.Error.Message == "" {
				return false
			}
			if response.Error.Details == nil {
				return false
			}

			// Verify validation errors are in details
			if _, ok := response.Error.Details["validation_errors"]; !ok {
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that JSON responses are properly formatted
func TestProperty_JSONResponsesAreValid(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("JSON responses are valid and parseable", prop.ForAll(
		func(useCode int, data map[string]string) bool {
			// Use standard HTTP status codes
			standardCodes := []int{
				http.StatusOK,
				http.StatusCreated,
				http.StatusAccepted,
				http.StatusBadRequest,
				http.StatusUnauthorized,
				http.StatusForbidden,
				http.StatusNotFound,
				http.StatusInternalServerError,
			}

			// Handle negative codes
			if useCode < 0 {
				useCode = -useCode
			}

			statusCode := standardCodes[useCode%len(standardCodes)]

			w := httptest.NewRecorder()
			RespondWithJSON(w, statusCode, data)

			// Check status code
			if w.Code != statusCode {
				return false
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				return false
			}

			// Verify JSON is parseable
			var result map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				return false
			}

			// Verify data matches
			for k, v := range data {
				if result[k] != v {
					return false
				}
			}

			return true
		},
		gen.Int(),
		gen.MapOf(gen.AlphaString(), gen.AlphaString()),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
