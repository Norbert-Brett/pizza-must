package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: ordering-platform, Property 43: Protected endpoints reject missing tokens
// Validates: Requirements 17.1
func TestProperty_ProtectedEndpointsRejectMissingTokens(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("requests without authorization header are rejected", prop.ForAll(
		func(pathSuffix string, method string) bool {
			logger, _ := zap.NewDevelopment()
			middleware := AuthMiddleware("test-secret", logger)

			// Create a test handler
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Ensure path starts with /
			path := "/" + pathSuffix
			if path == "/" {
				path = "/test"
			}

			// Create request without authorization header
			req := httptest.NewRequest(method, path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			return w.Code == http.StatusUnauthorized
		},
		gen.AlphaString(),
		gen.OneConstOf("GET", "POST", "PUT", "DELETE"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 44: Expired tokens are rejected
// Validates: Requirements 17.2
func TestProperty_ExpiredTokensAreRejected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("expired tokens are rejected with 401", prop.ForAll(
		func(userID string, role string) bool {
			logger, _ := zap.NewDevelopment()
			secret := "test-secret"
			middleware := AuthMiddleware(secret, logger)

			// Create expired token
			claims := jwt.MapClaims{
				"user_id": userID,
				"role":    role,
				"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte(secret))

			// Create test handler
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request with expired token
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			return w.Code == http.StatusUnauthorized
		},
		gen.AnyString(),
		gen.OneConstOf("user", "admin"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 45: Valid tokens allow processing
// Validates: Requirements 17.3
func TestProperty_ValidTokensAllowProcessing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid tokens allow request processing", prop.ForAll(
		func(userID string, role string) bool {
			logger, _ := zap.NewDevelopment()
			secret := "test-secret"
			middleware := AuthMiddleware(secret, logger)

			// Create valid token
			claims := jwt.MapClaims{
				"user_id": userID,
				"role":    role,
				"exp":     time.Now().Add(1 * time.Hour).Unix(), // Expires in 1 hour
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte(secret))

			// Track if handler was called
			handlerCalled := false

			// Create test handler
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true

				// Verify user ID and role are in context
				ctxUserID, ok1 := GetUserID(r.Context())
				ctxRole, ok2 := GetUserRole(r.Context())

				if !ok1 || !ok2 {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if ctxUserID != userID || ctxRole != role {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusOK)
			}))

			// Create request with valid token
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Handler should be called and return 200
			return handlerCalled && w.Code == http.StatusOK
		},
		gen.AnyString(),
		gen.OneConstOf("user", "admin"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test invalid token format
func TestProperty_InvalidTokenFormatRejected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("invalid token formats are rejected", prop.ForAll(
		func(invalidToken string) bool {
			logger, _ := zap.NewDevelopment()
			middleware := AuthMiddleware("test-secret", logger)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request with invalid token
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+invalidToken)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			return w.Code == http.StatusUnauthorized
		},
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test missing Bearer prefix
func TestProperty_MissingBearerPrefixRejected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tokens without Bearer prefix are rejected", prop.ForAll(
		func(token string) bool {
			logger, _ := zap.NewDevelopment()
			middleware := AuthMiddleware("test-secret", logger)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request without Bearer prefix
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			return w.Code == http.StatusUnauthorized
		},
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
