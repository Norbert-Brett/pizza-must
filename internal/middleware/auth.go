package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

// AuthMiddleware validates JWT tokens and extracts user claims
func AuthMiddleware(jwtSecret string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Debug("Missing authorization header")
				respondWithError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			// Check for Bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.Debug("Invalid authorization header format")
				respondWithError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				logger.Debug("Token validation failed", zap.Error(err))
				if err == jwt.ErrTokenExpired {
					respondWithError(w, http.StatusUnauthorized, "token expired")
				} else {
					respondWithError(w, http.StatusUnauthorized, "invalid token")
				}
				return
			}

			if !token.Valid {
				logger.Debug("Invalid token")
				respondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				logger.Error("Failed to extract claims from token")
				respondWithError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// Extract user ID
			userID, ok := claims["user_id"].(string)
			if !ok {
				logger.Error("Missing user_id in token claims")
				respondWithError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// Extract role
			role, ok := claims["role"].(string)
			if !ok {
				logger.Error("Missing role in token claims")
				respondWithError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UserRoleKey, role)

			logger.Debug("User authenticated",
				zap.String("user_id", userID),
				zap.String("role", role),
			)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts user ID from request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetUserRole extracts user role from request context
func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}
