package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerWindow int           // Number of requests allowed per window
	Window            time.Duration // Time window for rate limiting
	KeyPrefix         string        // Redis key prefix
}

// RateLimitMiddleware implements rate limiting using Redis
func RateLimitMiddleware(redisClient *redis.Client, config RateLimitConfig, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP address or user ID if authenticated)
			clientID := r.RemoteAddr
			if userID, ok := GetUserID(r.Context()); ok {
				clientID = userID
			}

			// Create Redis key
			key := fmt.Sprintf("%s:%s", config.KeyPrefix, clientID)

			ctx := context.Background()

			// Increment counter
			count, err := redisClient.Incr(ctx, key).Result()
			if err != nil {
				logger.Error("Failed to increment rate limit counter",
					zap.Error(err),
					zap.String("key", key),
				)
				// On Redis error, allow request to proceed
				next.ServeHTTP(w, r)
				return
			}

			// Set expiry on first request
			if count == 1 {
				redisClient.Expire(ctx, key, config.Window)
			}

			// Check if limit exceeded
			if count > int64(config.RequestsPerWindow) {
				// Get TTL for retry-after header
				ttl, err := redisClient.TTL(ctx, key).Result()
				if err != nil {
					ttl = config.Window
				}

				logger.Warn("Rate limit exceeded",
					zap.String("client_id", clientID),
					zap.Int64("count", count),
					zap.Int("limit", config.RequestsPerWindow),
				)

				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))

				respondWithError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			// Add rate limit headers
			remaining := config.RequestsPerWindow - int(count)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			next.ServeHTTP(w, r)
		})
	}
}
