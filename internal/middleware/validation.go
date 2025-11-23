package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateRequest validates the request body against a struct with validation tags
func ValidateRequest(v interface{}) error {
	return validate.Struct(v)
}

// ValidationMiddleware provides request validation
func ValidationMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This middleware can be used to add custom validation logic
			// For now, it just passes through
			// Actual validation happens in handlers using ValidateRequest
			next.ServeHTTP(w, r)
		})
	}
}

// DecodeAndValidate decodes JSON request body and validates it
func DecodeAndValidate(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return ValidateRequest(v)
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FormatValidationErrors converts validator errors to a readable format
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   e.Field(),
				Message: getErrorMessage(e),
			})
		}
	}

	return errors
}

func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "gte":
		return "Value must be greater than or equal to " + e.Param()
	case "lte":
		return "Value must be less than or equal to " + e.Param()
	case "gt":
		return "Value must be greater than " + e.Param()
	case "lt":
		return "Value must be less than " + e.Param()
	default:
		return "Invalid value"
	}
}
