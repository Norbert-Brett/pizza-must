package repository

import (
	"context"
	"testing"
	"time"

	"pizza-must/internal/domain"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: ordering-platform, Property 10: Product creation preserves attributes
// Validates: Requirements 4.1
func TestProperty_ProductCreationPreservesAttributes(t *testing.T) {
	// Ensure tables exist
	_, err := testDB.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL,
			category_id UUID NOT NULL,
			image_url VARCHAR(500),
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create products table: %v", err)
	}

	productRepo := NewProductRepository(testDB)
	categoryRepo := NewCategoryRepository(testDB)

	properties := gopter.NewProperties(nil)

	properties.Property("creating and retrieving a product preserves all attributes", prop.ForAll(
		func(name string, description string, price float64, imageURL string, stock int) bool {
			ctx := context.Background()

			// Create a category first
			category := &domain.Category{
				ID:          uuid.New(),
				Name:        "Test Category " + uuid.New().String(),
				Description: "Test category description",
				CreatedAt:   time.Now(),
			}
			err := categoryRepo.Create(ctx, category)
			if err != nil {
				t.Logf("FAIL: Failed to create category: %v", err)
				return false
			}

			// Create product with generated attributes
			product := &domain.Product{
				ID:          uuid.New(),
				Name:        name,
				Description: description,
				Price:       price,
				CategoryID:  category.ID,
				ImageURL:    imageURL,
				Stock:       stock,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			// Create the product
			err = productRepo.Create(ctx, product)
			if err != nil {
				t.Logf("FAIL: Failed to create product: %v", err)
				return false
			}

			// Retrieve the product
			retrieved, err := productRepo.FindByID(ctx, product.ID)
			if err != nil {
				t.Logf("FAIL: Failed to retrieve product: %v", err)
				return false
			}

			// Verify all attributes match
			if retrieved.ID != product.ID {
				t.Logf("FAIL: ID mismatch. Expected %s, got %s", product.ID, retrieved.ID)
				return false
			}

			if retrieved.Name != product.Name {
				t.Logf("FAIL: Name mismatch. Expected %s, got %s", product.Name, retrieved.Name)
				return false
			}

			if retrieved.Description != product.Description {
				t.Logf("FAIL: Description mismatch. Expected %s, got %s", product.Description, retrieved.Description)
				return false
			}

			// Compare prices with small tolerance for floating point
			if retrieved.Price < product.Price-0.01 || retrieved.Price > product.Price+0.01 {
				t.Logf("FAIL: Price mismatch. Expected %f, got %f", product.Price, retrieved.Price)
				return false
			}

			if retrieved.CategoryID != product.CategoryID {
				t.Logf("FAIL: CategoryID mismatch. Expected %s, got %s", product.CategoryID, retrieved.CategoryID)
				return false
			}

			if retrieved.ImageURL != product.ImageURL {
				t.Logf("FAIL: ImageURL mismatch. Expected %s, got %s", product.ImageURL, retrieved.ImageURL)
				return false
			}

			if retrieved.Stock != product.Stock {
				t.Logf("FAIL: Stock mismatch. Expected %d, got %d", product.Stock, retrieved.Stock)
				return false
			}

			// Verify timestamps are set
			if retrieved.CreatedAt.IsZero() {
				t.Logf("FAIL: CreatedAt is zero")
				return false
			}

			if retrieved.UpdatedAt.IsZero() {
				t.Logf("FAIL: UpdatedAt is zero")
				return false
			}

			// Cleanup
			_ = productRepo.Delete(ctx, product.ID)
			_, _ = testDB.Exec("DELETE FROM categories WHERE id = $1", category.ID)

			return true
		},
		gen.RegexMatch(`[A-Za-z0-9 ]{3,50}`),                      // name
		gen.RegexMatch(`[A-Za-z0-9 .,!?]{10,200}`),                // description
		gen.Float64Range(0.01, 9999.99),                           // price (positive values)
		gen.RegexMatch(`https?://[a-z0-9.-]+/[a-z0-9/._-]{1,50}`), // imageURL
		gen.IntRange(0, 1000),                                     // stock (non-negative)
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 14: Product updates are reflected
// Validates: Requirements 5.1, 5.3
func TestProperty_ProductUpdatesAreReflected(t *testing.T) {
	// Ensure tables exist
	_, err := testDB.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL,
			category_id UUID NOT NULL,
			image_url VARCHAR(500),
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create products table: %v", err)
	}

	productRepo := NewProductRepository(testDB)
	categoryRepo := NewCategoryRepository(testDB)

	properties := gopter.NewProperties(nil)

	properties.Property("updating a product and retrieving it shows the updated values", prop.ForAll(
		func(name1 string, name2 string, description1 string, description2 string,
			price1 float64, price2 float64, stock1 int, stock2 int) bool {
			ctx := context.Background()

			// Create a category first
			category := &domain.Category{
				ID:          uuid.New(),
				Name:        "Test Category " + uuid.New().String(),
				Description: "Test category description",
				CreatedAt:   time.Now(),
			}
			err := categoryRepo.Create(ctx, category)
			if err != nil {
				t.Logf("FAIL: Failed to create category: %v", err)
				return false
			}

			// Create initial product
			product := &domain.Product{
				ID:          uuid.New(),
				Name:        name1,
				Description: description1,
				Price:       price1,
				CategoryID:  category.ID,
				ImageURL:    "http://example.com/image1.jpg",
				Stock:       stock1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			err = productRepo.Create(ctx, product)
			if err != nil {
				t.Logf("FAIL: Failed to create product: %v", err)
				return false
			}

			// Update the product with new values
			product.Name = name2
			product.Description = description2
			product.Price = price2
			product.Stock = stock2
			product.UpdatedAt = time.Now()

			err = productRepo.Update(ctx, product)
			if err != nil {
				t.Logf("FAIL: Failed to update product: %v", err)
				return false
			}

			// Retrieve the product
			retrieved, err := productRepo.FindByID(ctx, product.ID)
			if err != nil {
				t.Logf("FAIL: Failed to retrieve product: %v", err)
				return false
			}

			// Verify updated values are reflected
			if retrieved.Name != name2 {
				t.Logf("FAIL: Name not updated. Expected %s, got %s", name2, retrieved.Name)
				return false
			}

			if retrieved.Description != description2 {
				t.Logf("FAIL: Description not updated. Expected %s, got %s", description2, retrieved.Description)
				return false
			}

			// Compare prices with small tolerance for floating point
			if retrieved.Price < price2-0.01 || retrieved.Price > price2+0.01 {
				t.Logf("FAIL: Price not updated. Expected %f, got %f", price2, retrieved.Price)
				return false
			}

			if retrieved.Stock != stock2 {
				t.Logf("FAIL: Stock not updated. Expected %d, got %d", stock2, retrieved.Stock)
				return false
			}

			// Cleanup
			_ = productRepo.Delete(ctx, product.ID)
			_, _ = testDB.Exec("DELETE FROM categories WHERE id = $1", category.ID)

			return true
		},
		gen.RegexMatch(`[A-Za-z0-9 ]{3,50}`),       // name1
		gen.RegexMatch(`[A-Za-z0-9 ]{3,50}`),       // name2
		gen.RegexMatch(`[A-Za-z0-9 .,!?]{10,200}`), // description1
		gen.RegexMatch(`[A-Za-z0-9 .,!?]{10,200}`), // description2
		gen.Float64Range(0.01, 9999.99),            // price1
		gen.Float64Range(0.01, 9999.99),            // price2
		gen.IntRange(0, 1000),                      // stock1
		gen.IntRange(0, 1000),                      // stock2
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ordering-platform, Property 16: Product deletion removes from catalog
// Validates: Requirements 6.1
func TestProperty_ProductDeletionRemovesFromCatalog(t *testing.T) {
	// Ensure tables exist
	_, err := testDB.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id UUID PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price DECIMAL(10, 2) NOT NULL,
			category_id UUID NOT NULL,
			image_url VARCHAR(500),
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create products table: %v", err)
	}

	productRepo := NewProductRepository(testDB)
	categoryRepo := NewCategoryRepository(testDB)

	properties := gopter.NewProperties(nil)

	properties.Property("deleting a product makes it not retrievable", prop.ForAll(
		func(name string, description string, price float64, stock int) bool {
			ctx := context.Background()

			// Create a category first
			category := &domain.Category{
				ID:          uuid.New(),
				Name:        "Test Category " + uuid.New().String(),
				Description: "Test category description",
				CreatedAt:   time.Now(),
			}
			err := categoryRepo.Create(ctx, category)
			if err != nil {
				t.Logf("FAIL: Failed to create category: %v", err)
				return false
			}

			// Create product
			product := &domain.Product{
				ID:          uuid.New(),
				Name:        name,
				Description: description,
				Price:       price,
				CategoryID:  category.ID,
				ImageURL:    "http://example.com/image.jpg",
				Stock:       stock,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			err = productRepo.Create(ctx, product)
			if err != nil {
				t.Logf("FAIL: Failed to create product: %v", err)
				return false
			}

			// Verify product exists
			_, err = productRepo.FindByID(ctx, product.ID)
			if err != nil {
				t.Logf("FAIL: Product should exist before deletion: %v", err)
				return false
			}

			// Delete the product
			err = productRepo.Delete(ctx, product.ID)
			if err != nil {
				t.Logf("FAIL: Failed to delete product: %v", err)
				return false
			}

			// Attempt to retrieve the deleted product
			_, err = productRepo.FindByID(ctx, product.ID)
			if err != ErrProductNotFound {
				t.Logf("FAIL: Expected ErrProductNotFound after deletion, got: %v", err)
				return false
			}

			// Cleanup category
			_, _ = testDB.Exec("DELETE FROM categories WHERE id = $1", category.ID)

			return true
		},
		gen.RegexMatch(`[A-Za-z0-9 ]{3,50}`),       // name
		gen.RegexMatch(`[A-Za-z0-9 .,!?]{10,200}`), // description
		gen.Float64Range(0.01, 9999.99),            // price
		gen.IntRange(0, 1000),                      // stock
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
