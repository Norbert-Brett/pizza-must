package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

// RequireAdmin middleware ensures the user has admin role
func RequireAdmin(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetUserRole(r.Context())
			if !ok {
				logger.Warn("Role not found in context")
				respondWithError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			if role != "admin" {
				logger.Warn("Non-admin user attempted to access admin endpoint",
					zap.String("role", role),
				)
				respondWithError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware ensures the user has one of the specified roles
func RequireRole(allowedRoles []string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetUserRole(r.Context())
			if !ok {
				logger.Warn("Role not found in context")
				respondWithError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			// Check if user's role is in allowed roles
			allowed := false
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					allowed = true
					break
				}
			}

			if !allowed {
				logger.Warn("User role not authorized",
					zap.String("role", role),
					zap.Strings("allowed_roles", allowedRoles),
				)
				respondWithError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
