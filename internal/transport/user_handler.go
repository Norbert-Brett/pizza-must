package transport

import (
	"encoding/json"
	"net/http"

	"pizza-must/internal/middleware"
	"pizza-must/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents the token refresh request payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         UserProfile `json:"user"`
}

// RefreshResponse represents the token refresh response
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// UserProfile represents user profile data
type UserProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService service.UserService
	logger      *zap.Logger
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService service.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// RegisterRoutes registers all user routes
func (h *UserHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/users", func(r chi.Router) {
		// Public routes
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/logout", h.Logout)
			r.Get("/profile", h.GetProfile)
		})
	})
}

// Register handles user registration
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// Decode and validate request
	if err := middleware.DecodeAndValidate(r, &req); err != nil {
		h.logger.Debug("Registration validation failed", zap.Error(err))

		// Check if it's a validation error
		if validationErrors := middleware.FormatValidationErrors(err); len(validationErrors) > 0 {
			middleware.RespondWithValidationErrors(w, validationErrors)
			return
		}

		// JSON decode error
		middleware.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call service
	user, err := h.userService.Register(r.Context(), req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		h.logger.Error("Registration failed", zap.Error(err))

		// Check for specific errors
		if err.Error() == "user with this email already exists" {
			middleware.RespondWithError(w, http.StatusConflict, "user with this email already exists")
			return
		}

		middleware.RespondWithError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	// Return user profile
	profile := UserProfile{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}

	h.logger.Info("User registered successfully", zap.String("user_id", user.ID.String()))
	middleware.RespondWithJSON(w, http.StatusCreated, profile)
}

// Login handles user authentication
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// Decode and validate request
	if err := middleware.DecodeAndValidate(r, &req); err != nil {
		h.logger.Debug("Login validation failed", zap.Error(err))

		// Check if it's a validation error
		if validationErrors := middleware.FormatValidationErrors(err); len(validationErrors) > 0 {
			middleware.RespondWithValidationErrors(w, validationErrors)
			return
		}

		middleware.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call service
	accessToken, refreshToken, user, err := h.userService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Debug("Login failed", zap.Error(err))

		// Check for invalid credentials
		if err == service.ErrInvalidCredentials {
			middleware.RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		middleware.RespondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	// Return tokens and user profile
	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserProfile{
			ID:        user.ID.String(),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
		},
	}

	h.logger.Info("User logged in successfully", zap.String("user_id", user.ID.String()))
	middleware.RespondWithJSON(w, http.StatusOK, response)
}

// Logout handles user logout
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	// Decode request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("Logout decode failed", zap.Error(err))
		middleware.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call service
	if err := h.userService.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		middleware.RespondWithError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	h.logger.Info("User logged out successfully")
	middleware.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	// Decode and validate request
	if err := middleware.DecodeAndValidate(r, &req); err != nil {
		h.logger.Debug("Refresh token validation failed", zap.Error(err))

		// Check if it's a validation error
		if validationErrors := middleware.FormatValidationErrors(err); len(validationErrors) > 0 {
			middleware.RespondWithValidationErrors(w, validationErrors)
			return
		}

		middleware.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call service
	newAccessToken, err := h.userService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Debug("Token refresh failed", zap.Error(err))

		// Check for specific errors
		if err == service.ErrInvalidToken {
			middleware.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		if err == service.ErrTokenExpired {
			middleware.RespondWithError(w, http.StatusUnauthorized, "refresh token expired")
			return
		}

		middleware.RespondWithError(w, http.StatusInternalServerError, "failed to refresh token")
		return
	}

	// Return new access token
	response := RefreshResponse{
		AccessToken: newAccessToken,
	}

	h.logger.Info("Token refreshed successfully")
	middleware.RespondWithJSON(w, http.StatusOK, response)
}

// GetProfile handles getting user profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.logger.Error("User ID not found in context")
		middleware.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Invalid user ID format", zap.Error(err))
		middleware.RespondWithError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Get user from service
	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", zap.Error(err))
		middleware.RespondWithError(w, http.StatusInternalServerError, "failed to get user profile")
		return
	}

	// Return user profile
	profile := UserProfile{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}

	middleware.RespondWithJSON(w, http.StatusOK, profile)
}
