package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pizza-must/internal/domain"
	"pizza-must/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing (10 as per requirements)
	BcryptCost = 10

	// Token expiration times
	AccessTokenExpiration  = 15 * time.Minute
	RefreshTokenExpiration = 7 * 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
)

// UserService defines the interface for user business logic
type UserService interface {
	Register(ctx context.Context, email, password, firstName, lastName string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *domain.User, err error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	ValidateToken(tokenString string) (*Claims, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// Claims represents the JWT claims
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type userService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtSecret        string
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtSecret string,
) UserService {
	return &userService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
	}
}

// Register creates a new user account with hashed password
func (s *userService) Register(ctx context.Context, email, password, firstName, lastName string) (*domain.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && err != repository.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, repository.ErrUserAlreadyExists
	}

	// Hash the password with bcrypt cost factor 10
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user entity
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         "user", // Default role
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns JWT tokens
func (s *userService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *domain.User, err error) {
	// Find user by email
	user, err = s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := s.verifyPassword(user.PasswordHash, password); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	// Generate access token
	accessToken, err = s.generateAccessToken(user)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err = s.generateRefreshToken(ctx, user)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

// Logout invalidates the refresh token
func (s *userService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.refreshTokenRepo.Revoke(ctx, refreshToken); err != nil {
		if err == repository.ErrRefreshTokenNotFound {
			// Token doesn't exist, consider it already logged out
			return nil
		}
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

// RefreshToken generates a new access token using a valid refresh token
func (s *userService) RefreshToken(ctx context.Context, refreshTokenString string) (newAccessToken string, err error) {
	// Find and validate refresh token
	refreshToken, err := s.refreshTokenRepo.FindByToken(ctx, refreshTokenString)
	if err != nil {
		if err == repository.ErrRefreshTokenNotFound || err == repository.ErrRefreshTokenRevoked {
			return "", ErrInvalidToken
		}
		return "", fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Check if token is expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return "", ErrTokenExpired
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token
	newAccessToken, err = s.generateAccessToken(user)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return newAccessToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *userService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// hashPassword hashes a password using bcrypt with cost factor 10
func (s *userService) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// verifyPassword verifies a password against a bcrypt hash
func (s *userService) verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// generateAccessToken generates a JWT access token with user ID and role claims
func (s *userService) generateAccessToken(user *domain.User) (string, error) {
	expirationTime := time.Now().Add(AccessTokenExpiration)
	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// generateRefreshToken generates a refresh token and stores it in the database
func (s *userService) generateRefreshToken(ctx context.Context, user *domain.User) (string, error) {
	// Generate a random token string
	tokenString := uuid.New().String()

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(RefreshTokenExpiration),
		CreatedAt: time.Now(),
		Revoked:   false,
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", err
	}

	return tokenString, nil
}
