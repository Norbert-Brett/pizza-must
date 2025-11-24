package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pizza-must/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrProductNotFound = errors.New("product not found")
)

// SortOrder represents the sort direction
type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, categoryID *uuid.UUID, page, pageSize int, sortBy string, sortOrder SortOrder) ([]*domain.Product, int, error)
	Search(ctx context.Context, query string, page, pageSize int) ([]*domain.Product, int, error)
}

type productRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new instance of ProductRepository
func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepository{db: db}
}

// Create inserts a new product into the database using parameterized queries
func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	query := `
		INSERT INTO products (id, name, description, price, category_id, image_url, stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.CategoryID,
		product.ImageURL,
		product.Stock,
		product.CreatedAt,
		product.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

// Update updates an existing product in the database using parameterized queries
func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	query := `
		UPDATE products
		SET name = $2, description = $3, price = $4, category_id = $5, 
		    image_url = $6, stock = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.CategoryID,
		product.ImageURL,
		product.Stock,
		product.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}

// Delete removes a product from the database using parameterized queries
func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM products WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}

// FindByID retrieves a product by ID using parameterized queries
func (r *productRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	query := `
		SELECT id, name, description, price, category_id, image_url, stock, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	product := &domain.Product{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.CategoryID,
		&product.ImageURL,
		&product.Stock,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to find product by ID: %w", err)
	}

	return product, nil
}

// List retrieves products with optional category filtering, pagination, and sorting
func (r *productRepository) List(ctx context.Context, categoryID *uuid.UUID, page, pageSize int, sortBy string, sortOrder SortOrder) ([]*domain.Product, int, error) {
	// Validate sort field to prevent SQL injection
	validSortFields := map[string]bool{
		"name":       true,
		"price":      true,
		"created_at": true,
		"stock":      true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Default sort field
	}

	// Validate sort order
	if sortOrder != SortOrderAsc && sortOrder != SortOrderDesc {
		sortOrder = SortOrderDesc // Default sort order
	}

	// Build the WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if categoryID != nil {
		whereClause = fmt.Sprintf("WHERE category_id = $%d", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	// Count total products
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build the main query with sorting and pagination
	query := fmt.Sprintf(`
		SELECT id, name, description, price, category_id, image_url, stock, created_at, updated_at
		FROM products
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := []*domain.Product{}
	for rows.Next() {
		product := &domain.Product{}
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.CategoryID,
			&product.ImageURL,
			&product.Stock,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating products: %w", err)
	}

	return products, total, nil
}

// Search searches for products by name or description with pagination
func (r *productRepository) Search(ctx context.Context, query string, page, pageSize int) ([]*domain.Product, int, error) {
	// If query is empty, return all products
	if strings.TrimSpace(query) == "" {
		return r.List(ctx, nil, page, pageSize, "created_at", SortOrderDesc)
	}

	// Use ILIKE for case-insensitive search
	searchPattern := "%" + query + "%"

	// Count total matching products
	countQuery := `
		SELECT COUNT(*)
		FROM products
		WHERE name ILIKE $1 OR description ILIKE $1
	`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Search products
	searchQuery := `
		SELECT id, name, description, price, category_id, image_url, stock, created_at, updated_at
		FROM products
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, searchQuery, searchPattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search products: %w", err)
	}
	defer rows.Close()

	products := []*domain.Product{}
	for rows.Next() {
		product := &domain.Product{}
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.CategoryID,
			&product.ImageURL,
			&product.Stock,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating search results: %w", err)
	}

	return products, total, nil
}
