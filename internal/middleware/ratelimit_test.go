package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Feature: ordering-platform, Property 59: Rate limiting blocks excessive requests
// Validates: Requirements 21.1
func TestProperty_RateLimitingBlocksExcessiveRequests(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("excessive requests are blocked with 429", prop.ForAll(
		func(requestsPerWindow int, excessRequests int) bool {
			// Ensure we have a reasonable limit and excess
			if requestsPerWindow < 1 {
				requestsPerWindow = 5
			}
			if requestsPerWindow > 100 {
				requestsPerWindow = 100
			}
			if excessRequests < 1 {
				excessRequests = 1
			}
			if excessRequests > 50 {
				excessRequests = 50
			}

			// Create a mock Redis server using miniredis
			mr, err := miniredis.Run()
			if err != nil {
				t.Fatalf("Failed to start miniredis: %v", err)
				return false
			}
			defer mr.Close()

			// Create Redis client connected to miniredis
			redisClient := redis.NewClient(&redis.Options{
				Addr: mr.Addr(),
			})
			defer redisClient.Close()

			logger, _ := zap.NewDevelopment()

			config := RateLimitConfig{
				RequestsPerWindow: requestsPerWindow,
				Window:            1 * time.Second,
				KeyPrefix:         "test_rate_limit",
			}

			middleware := RateLimitMiddleware(redisClient, config, logger)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Make requests up to the limit
			clientIP := "192.168.1.100"
			successCount := 0
			blockedCount := 0

			totalRequests := requestsPerWindow + excessRequests

			for i := 0; i < totalRequests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = clientIP
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					successCount++
				} else if w.Code == http.StatusTooManyRequests {
					blockedCount++
				}
			}

			// Should allow exactly requestsPerWindow requests and block the rest
			return successCount == requestsPerWindow && blockedCount == excessRequests
		},
		gen.IntRange(5, 20),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Test that rate limit headers are set correctly
func TestProperty_RateLimitHeadersAreSet(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("rate limit headers are present in responses", prop.ForAll(
		func(requestsPerWindow int) bool {
			if requestsPerWindow < 1 {
				requestsPerWindow = 10
			}
			if requestsPerWindow > 100 {
				requestsPerWindow = 100
			}

			// Create a mock Redis server using miniredis
			mr, err := miniredis.Run()
			if err != nil {
				t.Fatalf("Failed to start miniredis: %v", err)
				return false
			}
			defer mr.Close()

			redisClient := redis.NewClient(&redis.Options{
				Addr: mr.Addr(),
			})
			defer redisClient.Close()

			logger, _ := zap.NewDevelopment()

			config := RateLimitConfig{
				RequestsPerWindow: requestsPerWindow,
				Window:            1 * time.Second,
				KeyPrefix:         "test_rate_limit_headers",
			}

			middleware := RateLimitMiddleware(redisClient, config, logger)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			clientIP := "192.168.1.101"
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = clientIP
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Check that rate limit headers are present
			hasLimit := w.Header().Get("X-RateLimit-Limit") != ""
			hasRemaining := w.Header().Get("X-RateLimit-Remaining") != ""

			return hasLimit && hasRemaining
		},
		gen.IntRange(5, 50),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
