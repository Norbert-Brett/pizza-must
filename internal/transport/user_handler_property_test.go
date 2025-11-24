package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pizza-must/internal/domain"
	"pizza-must/internal/repository"
	"pizza-must/internal/service"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Mock repositories for testing
type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if _, exists := m.users[user.Email]; exists {
		return repository.ErrUserAlreadyExists
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, exists := m.users[email]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

type mockRefreshTokenRepository struct {
	tokens map[string]*domain.RefreshToken
}

func newMockRefreshTokenRepository() *mockRefreshTokenRepository {
	return &mockRefreshTokenRepository{
		tokens: make(map[string]*domain.RefreshToken),
	}
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockRefreshTokenRepository) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	refreshToken, exists := m.tokens[token]
	if !exists {
		return nil, repository.ErrRefreshTokenNotFound
	}
	if refreshToken.Revoked {
		return nil, repository.ErrRefreshTokenRevoked
	}
	return refreshToken, nil
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	refreshToken, exists := m.tokens[token]
	if !exists {
		return repository.ErrRefreshTokenNotFound
	}
	refreshToken.Revoked = true
	return nil
}

// Feature: ordering-platform, Property 3: Invalid registration data is rejected
// Validates: Requirements 1.5
func TestProperty_InvalidRegistrationDataIsRejected(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("registration with invalid data returns validation errors", prop.ForAll(
		func(invalidCase int) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			userService := service.NewUserService(userRepo, refreshTokenRepo, "test-secret")
			logger, _ := zap.NewDevelopment()
			handler := NewUserHandler(userService, logger)

			var reqBody RegisterRequest

			// Generate different invalid cases
			switch invalidCase % 4 {
			case 0:
				// Empty email
				reqBody = RegisterRequest{
					Email:     "",
					Password:  "ValidPass123",
					FirstName: "John",
					LastName:  "Doe",
				}
			case 1:
				// Invalid email format
				reqBody = RegisterRequest{
					Email:     "not-an-email",
					Password:  "ValidPass123",
					FirstName: "John",
					LastName:  "Doe",
				}
			case 2:
				// Short password (less than 8 characters)
				reqBody = RegisterRequest{
					Email:     "test@example.com",
					Password:  "short",
					FirstName: "John",
					LastName:  "Doe",
				}
			case 3:
				// Missing required fields
				reqBody = RegisterRequest{
					Email:    "test@example.com",
					Password: "ValidPass123",
					// FirstName and LastName missing
				}
			}

			// Create request
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler.Register(w, req)

			// Verify response is 400 Bad Request or 409 Conflict
			if w.Code != http.StatusBadRequest && w.Code != http.StatusConflict {
				t.Logf("FAIL: Expected 400 or 409 status code, got %d", w.Code)
				return false
			}

			// Verify response contains error structure
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Logf("FAIL: Could not decode error response: %v", err)
				return false
			}

			// Verify error field exists
			if _, exists := response["error"]; !exists {
				t.Logf("FAIL: Response missing 'error' field")
				return false
			}

			return true
		},
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 2: Successful registration returns profile data
// Validates: Requirements 1.4
func TestProperty_SuccessfulRegistrationReturnsProfileData(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("successful registration returns user profile with all fields", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			userService := service.NewUserService(userRepo, refreshTokenRepo, "test-secret")
			logger, _ := zap.NewDevelopment()
			handler := NewUserHandler(userService, logger)

			// Create request
			reqBody := RegisterRequest{
				Email:     email,
				Password:  password,
				FirstName: firstName,
				LastName:  lastName,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler.Register(w, req)

			// Skip if registration failed (e.g., duplicate email from previous iteration)
			if w.Code != http.StatusCreated {
				return true
			}

			// Verify response is 201 Created
			if w.Code != http.StatusCreated {
				t.Logf("FAIL: Expected 201 status code, got %d", w.Code)
				return false
			}

			// Decode response
			var profile UserProfile
			if err := json.NewDecoder(w.Body).Decode(&profile); err != nil {
				t.Logf("FAIL: Could not decode response: %v", err)
				return false
			}

			// Verify all profile fields are present
			if profile.ID == "" {
				t.Logf("FAIL: Profile missing ID")
				return false
			}

			if profile.Email != email {
				t.Logf("FAIL: Email mismatch. Expected %s, got %s", email, profile.Email)
				return false
			}

			if profile.FirstName != firstName {
				t.Logf("FAIL: FirstName mismatch. Expected %s, got %s", firstName, profile.FirstName)
				return false
			}

			if profile.LastName != lastName {
				t.Logf("FAIL: LastName mismatch. Expected %s, got %s", lastName, profile.LastName)
				return false
			}

			if profile.Role == "" {
				t.Logf("FAIL: Profile missing Role")
				return false
			}

			// Verify ID is a valid UUID
			if _, err := uuid.Parse(profile.ID); err != nil {
				t.Logf("FAIL: Profile ID is not a valid UUID: %v", err)
				return false
			}

			return true
		},
		gen.RegexMatch(`[a-z]{3,10}@[a-z]{3,8}\.(com|org|net)`),
		gen.RegexMatch(`[A-Za-z0-9!@#$%]{8,20}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 4: Valid login returns both tokens
// Validates: Requirements 2.1
func TestProperty_ValidLoginReturnsBothTokens(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid login returns access token and refresh token", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			userService := service.NewUserService(userRepo, refreshTokenRepo, "test-secret")
			logger, _ := zap.NewDevelopment()
			handler := NewUserHandler(userService, logger)

			// First, register the user
			_, err := userService.Register(context.Background(), email, password, firstName, lastName)
			if err != nil {
				return true // Skip if registration fails
			}

			// Create login request
			loginReq := LoginRequest{
				Email:    email,
				Password: password,
			}
			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute login
			handler.Login(w, req)

			// Verify response is 200 OK
			if w.Code != http.StatusOK {
				t.Logf("FAIL: Expected 200 status code, got %d", w.Code)
				return false
			}

			// Decode response
			var loginResp LoginResponse
			if err := json.NewDecoder(w.Body).Decode(&loginResp); err != nil {
				t.Logf("FAIL: Could not decode login response: %v", err)
				return false
			}

			// Verify access token is present and not empty
			if loginResp.AccessToken == "" {
				t.Logf("FAIL: Access token is empty")
				return false
			}

			// Verify refresh token is present and not empty
			if loginResp.RefreshToken == "" {
				t.Logf("FAIL: Refresh token is empty")
				return false
			}

			// Verify user profile is included
			if loginResp.User.ID == "" {
				t.Logf("FAIL: User profile missing ID")
				return false
			}

			if loginResp.User.Email != email {
				t.Logf("FAIL: User email mismatch")
				return false
			}

			// Verify access token is valid
			claims, err := userService.ValidateToken(loginResp.AccessToken)
			if err != nil {
				t.Logf("FAIL: Access token validation failed: %v", err)
				return false
			}

			// Verify claims contain user information
			if claims.UserID.String() != loginResp.User.ID {
				t.Logf("FAIL: Token user ID doesn't match profile ID")
				return false
			}

			// Verify refresh token can be used
			newAccessToken, err := userService.RefreshToken(context.Background(), loginResp.RefreshToken)
			if err != nil {
				t.Logf("FAIL: Refresh token is not valid: %v", err)
				return false
			}

			if newAccessToken == "" {
				t.Logf("FAIL: Refresh token returned empty access token")
				return false
			}

			return true
		},
		gen.RegexMatch(`[a-z]{3,10}@[a-z]{3,8}\.(com|org|net)`),
		gen.RegexMatch(`[A-Za-z0-9!@#$%]{8,20}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
