package middleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Test struct with validation tags
type TestRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"required,gte=0,lte=150"`
}

// Feature: ordering-platform, Property 48: Required field validation works
// Validates: Requirements 18.2
func TestProperty_RequiredFieldValidationWorks(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("missing required fields are rejected", prop.ForAll(
		func(includeNameField bool, includeEmailField bool, includeAgeField bool) bool {
			// Create request with some fields missing
			reqMap := make(map[string]interface{})

			if includeNameField {
				reqMap["name"] = "John Doe"
			}
			if includeEmailField {
				reqMap["email"] = "john@example.com"
			}
			if includeAgeField {
				reqMap["age"] = 25
			}

			// If all fields are present, this should pass validation
			allFieldsPresent := includeNameField && includeEmailField && includeAgeField

			reqBody, _ := json.Marshal(reqMap)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			var testReq TestRequest
			err := DecodeAndValidate(req, &testReq)

			if allFieldsPresent {
				// Should pass validation
				return err == nil
			} else {
				// Should fail validation
				return err != nil
			}
		},
		gen.Bool(),
		gen.Bool(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that validation errors are properly formatted
func TestProperty_ValidationErrorsAreFormatted(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("validation errors include field information", prop.ForAll(
		func() bool {
			// Create request with invalid email
			reqMap := map[string]interface{}{
				"name":  "John Doe",
				"email": "invalid-email", // Invalid email format
				"age":   25,
			}

			reqBody, _ := json.Marshal(reqMap)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			var testReq TestRequest
			err := DecodeAndValidate(req, &testReq)

			if err == nil {
				return false // Should have validation error
			}

			// Format the errors
			validationErrors := FormatValidationErrors(err)

			// Should have at least one error
			if len(validationErrors) == 0 {
				return false
			}

			// Each error should have a field and message
			for _, ve := range validationErrors {
				if ve.Field == "" || ve.Message == "" {
					return false
				}
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that valid requests pass validation
func TestProperty_ValidRequestsPassValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid requests pass validation", prop.ForAll(
		func(seed int) bool {
			// Use seed to generate deterministic but varied data
			names := []string{"John Doe", "Jane Smith", "Bob Johnson", "Alice Williams"}
			ages := []int{25, 30, 45, 60, 18, 75, 100}

			// Handle negative seeds
			if seed < 0 {
				seed = -seed
			}

			name := names[seed%len(names)]
			age := ages[seed%len(ages)]

			reqMap := map[string]interface{}{
				"name":  name,
				"email": "valid@example.com",
				"age":   age,
			}

			reqBody, _ := json.Marshal(reqMap)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			var testReq TestRequest
			err := DecodeAndValidate(req, &testReq)

			// Should pass validation
			return err == nil
		},
		gen.Int(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test age range validation
func TestProperty_AgeRangeValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("age outside valid range is rejected", prop.ForAll(
		func(age int) bool {
			reqMap := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   age,
			}

			reqBody, _ := json.Marshal(reqMap)
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			var testReq TestRequest
			err := DecodeAndValidate(req, &testReq)

			// Age should be between 0 and 150
			if age >= 0 && age <= 150 {
				return err == nil // Should pass
			} else {
				return err != nil // Should fail
			}
		},
		gen.IntRange(-100, 200),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
