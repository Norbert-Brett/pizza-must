package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pizza-must/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryAlreadyExists = errors.New("category with this name already exists")
)

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	List(ctx context.Context) ([]*domain.Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
}

type categoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new instance of CategoryRepository
func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

// Create inserts a new category into the database using parameterized queries
func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) error {
	query := `
		INSERT INTO categories (id, name, description, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		category.ID,
		category.Name,
		category.Description,
		category.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate name)
		if err.Error() == "pq: duplicate key value violates unique constraint \"categories_name_key\"" ||
			err.Error() == "ERROR: duplicate key value violates unique constraint \"categories_name_key\" (SQLSTATE 23505)" {
			return ErrCategoryAlreadyExists
		}
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// List retrieves all categories
func (r *categoryRepository) List(ctx context.Context) ([]*domain.Category, error) {
	query := `
		SELECT id, name, description, created_at
		FROM categories
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	categories := []*domain.Category{}
	for rows.Next() {
		category := &domain.Category{}
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return categories, nil
}

// FindByID retrieves a category by ID using parameterized queries
func (r *categoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, name, description, created_at
		FROM categories
		WHERE id = $1
	`

	category := &domain.Category{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to find category by ID: %w", err)
	}

	return category, nil
}
