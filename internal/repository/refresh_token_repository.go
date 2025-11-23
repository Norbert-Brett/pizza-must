package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pizza-must/internal/domain"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked  = errors.New("refresh token has been revoked")
)

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
}

type refreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository creates a new instance of RefreshTokenRepository
func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// Create inserts a new refresh token into the database using parameterized queries
func (r *refreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		token.CreatedAt,
		token.Revoked,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// FindByToken retrieves a refresh token by its token string using parameterized queries
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, revoked
		FROM refresh_tokens
		WHERE token = $1
	`

	refreshToken := &domain.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.Token,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&refreshToken.Revoked,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	if refreshToken.Revoked {
		return nil, ErrRefreshTokenRevoked
	}

	return refreshToken, nil
}

// Revoke marks a refresh token as revoked using parameterized queries
func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE token = $1
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
