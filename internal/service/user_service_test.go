package service

import (
	"context"
	"testing"
	"time"

	"pizza-must/internal/domain"
	"pizza-must/internal/repository"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"golang.org/x/crypto/bcrypt"
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

// Feature: ordering-platform, Property 1: Registration creates hashed passwords
// Validates: Requirements 1.1, 1.3
func TestProperty_RegistrationCreatesHashedPasswords(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("passwords are hashed with bcrypt and not stored as plaintext", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			service := NewUserService(userRepo, refreshTokenRepo, "test-secret")
			ctx := context.Background()

			// Execute registration
			user, err := service.Register(ctx, email, password, firstName, lastName)
			if err != nil {
				// If registration fails, skip this test case
				return true
			}

			// Verify password is hashed (not equal to plaintext)
			if user.PasswordHash == password {
				t.Logf("FAIL: Password stored as plaintext for email %s", email)
				return false
			}

			// Verify password hash is a valid bcrypt hash
			err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
			if err != nil {
				t.Logf("FAIL: Password hash is not a valid bcrypt hash or doesn't match: %v", err)
				return false
			}

			// Verify the stored user has the hashed password
			storedUser, err := userRepo.FindByEmail(ctx, email)
			if err != nil {
				t.Logf("FAIL: Could not find stored user: %v", err)
				return false
			}

			if storedUser.PasswordHash != user.PasswordHash {
				t.Logf("FAIL: Stored password hash doesn't match returned password hash")
				return false
			}

			if storedUser.PasswordHash == password {
				t.Logf("FAIL: Stored password is plaintext")
				return false
			}

			return true
		},
		// Generate valid email addresses
		gen.RegexMatch(`[a-z]{3,10}@[a-z]{3,8}\.(com|org|net)`),
		// Generate passwords with at least 8 characters
		gen.RegexMatch(`[A-Za-z0-9!@#$%]{8,20}`),
		// Generate first names
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		// Generate last names
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 5: JWT tokens contain required claims
// Validates: Requirements 2.3
func TestProperty_JWTTokensContainRequiredClaims(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("access tokens contain user ID and role claims", prop.ForAll(
		func(email string, password string, firstName string, lastName string, role string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			service := NewUserService(userRepo, refreshTokenRepo, "test-secret-key")
			ctx := context.Background()

			// Register user
			user, err := service.Register(ctx, email, password, firstName, lastName)
			if err != nil {
				return true // Skip if registration fails
			}

			// Override role for testing
			user.Role = role
			userRepo.users[email] = user

			// Login to get tokens
			accessToken, _, _, err := service.Login(ctx, email, password)
			if err != nil {
				t.Logf("FAIL: Login failed: %v", err)
				return false
			}

			// Validate and decode the access token
			claims, err := service.ValidateToken(accessToken)
			if err != nil {
				t.Logf("FAIL: Token validation failed: %v", err)
				return false
			}

			// Verify user ID claim is present and matches
			if claims.UserID != user.ID {
				t.Logf("FAIL: User ID claim mismatch. Expected %s, got %s", user.ID, claims.UserID)
				return false
			}

			// Verify role claim is present and matches
			if claims.Role != role {
				t.Logf("FAIL: Role claim mismatch. Expected %s, got %s", role, claims.Role)
				return false
			}

			// Verify token has expiration
			if claims.ExpiresAt == nil {
				t.Logf("FAIL: Token missing expiration claim")
				return false
			}

			// Verify token has issued at
			if claims.IssuedAt == nil {
				t.Logf("FAIL: Token missing issued at claim")
				return false
			}

			return true
		},
		gen.RegexMatch(`[a-z]{3,10}@[a-z]{3,8}\.(com|org|net)`),
		gen.RegexMatch(`[A-Za-z0-9!@#$%]{8,20}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		gen.OneConstOf("user", "admin"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 7: Token refresh round trip
// Validates: Requirements 2.5
func TestProperty_TokenRefreshRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid refresh token returns new valid access token", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			service := NewUserService(userRepo, refreshTokenRepo, "test-secret-key")
			ctx := context.Background()

			// Register and login
			_, err := service.Register(ctx, email, password, firstName, lastName)
			if err != nil {
				return true // Skip if registration fails
			}

			_, refreshToken, user, err := service.Login(ctx, email, password)
			if err != nil {
				t.Logf("FAIL: Login failed: %v", err)
				return false
			}

			// Use refresh token to get new access token
			newAccessToken, err := service.RefreshToken(ctx, refreshToken)
			if err != nil {
				t.Logf("FAIL: Token refresh failed: %v", err)
				return false
			}

			// Verify new access token is valid
			claims, err := service.ValidateToken(newAccessToken)
			if err != nil {
				t.Logf("FAIL: New access token validation failed: %v", err)
				return false
			}

			// Verify claims match the user
			if claims.UserID != user.ID {
				t.Logf("FAIL: User ID mismatch in refreshed token")
				return false
			}

			if claims.Role != user.Role {
				t.Logf("FAIL: Role mismatch in refreshed token")
				return false
			}

			// Verify token is not expired
			if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
				t.Logf("FAIL: Refreshed token is already expired")
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

// Feature: ordering-platform, Property 8: Logout invalidates refresh token
// Validates: Requirements 3.1
func TestProperty_LogoutInvalidatesRefreshToken(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("logout marks refresh token as revoked", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Setup
			userRepo := newMockUserRepository()
			refreshTokenRepo := newMockRefreshTokenRepository()
			service := NewUserService(userRepo, refreshTokenRepo, "test-secret-key")
			ctx := context.Background()

			// Register and login
			_, err := service.Register(ctx, email, password, firstName, lastName)
			if err != nil {
				return true // Skip if registration fails
			}

			_, refreshToken, _, err := service.Login(ctx, email, password)
			if err != nil {
				t.Logf("FAIL: Login failed: %v", err)
				return false
			}

			// Verify refresh token works before logout
			_, err = service.RefreshToken(ctx, refreshToken)
			if err != nil {
				t.Logf("FAIL: Refresh token should work before logout: %v", err)
				return false
			}

			// Logout
			err = service.Logout(ctx, refreshToken)
			if err != nil {
				t.Logf("FAIL: Logout failed: %v", err)
				return false
			}

			// Verify refresh token is now invalid
			_, err = service.RefreshToken(ctx, refreshToken)
			if err == nil {
				t.Logf("FAIL: Refresh token should be invalid after logout")
				return false
			}

			// Verify the error is the expected one (invalid token)
			if err != ErrInvalidToken {
				t.Logf("FAIL: Expected ErrInvalidToken, got: %v", err)
				return false
			}

			// Verify token is marked as revoked in repository
			storedToken, err := refreshTokenRepo.FindByToken(ctx, refreshToken)
			if err != repository.ErrRefreshTokenRevoked {
				t.Logf("FAIL: Token should be revoked in repository, got error: %v", err)
				return false
			}

			// storedToken should be nil when revoked
			if storedToken != nil {
				t.Logf("FAIL: Revoked token should not be returned")
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
