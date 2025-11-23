package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Feature: ordering-platform, Property 68: Pending migrations are executed
// Validates: Requirements 23.2
func TestMigrationFilesExist(t *testing.T) {
	migrationsDir := "../../migrations"

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		t.Fatal("Migrations directory does not exist")
	}

	// Expected migration files
	expectedMigrations := []string{
		"00001_create_users_table.sql",
		"00002_create_refresh_tokens_table.sql",
		"00003_create_categories_table.sql",
		"00004_create_products_table.sql",
		"00005_create_cart_items_table.sql",
		"00006_create_orders_table.sql",
		"00007_create_order_items_table.sql",
		"00008_create_updated_at_trigger.sql",
	}

	for _, migration := range expectedMigrations {
		path := filepath.Join(migrationsDir, migration)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Migration file %s does not exist", migration)
		}
	}
}

func TestMigrationFilesHaveUpAndDown(t *testing.T) {
	migrationsDir := "../../migrations"

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	sqlFileCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		sqlFileCount++
		content, err := os.ReadFile(filepath.Join(migrationsDir, file.Name()))
		if err != nil {
			t.Errorf("Failed to read migration file %s: %v", file.Name(), err)
			continue
		}

		contentStr := string(content)

		// Check for goose Up directive
		if !strings.Contains(contentStr, "-- +goose Up") {
			t.Errorf("Migration file %s missing '-- +goose Up' directive", file.Name())
		}

		// Check for goose Down directive
		if !strings.Contains(contentStr, "-- +goose Down") {
			t.Errorf("Migration file %s missing '-- +goose Down' directive", file.Name())
		}

		// Check for StatementBegin/End
		if !strings.Contains(contentStr, "-- +goose StatementBegin") {
			t.Errorf("Migration file %s missing '-- +goose StatementBegin' directive", file.Name())
		}

		if !strings.Contains(contentStr, "-- +goose StatementEnd") {
			t.Errorf("Migration file %s missing '-- +goose StatementEnd' directive", file.Name())
		}
	}

	if sqlFileCount == 0 {
		t.Error("No SQL migration files found")
	}
}

func TestMigrationFilesCreateExpectedTables(t *testing.T) {
	migrationsDir := "../../migrations"

	expectedTables := map[string]string{
		"users":          "00001_create_users_table.sql",
		"refresh_tokens": "00002_create_refresh_tokens_table.sql",
		"categories":     "00003_create_categories_table.sql",
		"products":       "00004_create_products_table.sql",
		"cart_items":     "00005_create_cart_items_table.sql",
		"orders":         "00006_create_orders_table.sql",
		"order_items":    "00007_create_order_items_table.sql",
	}

	for tableName, migrationFile := range expectedTables {
		path := filepath.Join(migrationsDir, migrationFile)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read migration file %s: %v", migrationFile, err)
			continue
		}

		contentStr := string(content)

		// Check if migration creates the table
		createTableStmt := "CREATE TABLE IF NOT EXISTS " + tableName
		if !strings.Contains(contentStr, createTableStmt) {
			t.Errorf("Migration file %s does not create table %s", migrationFile, tableName)
		}

		// Check if migration has drop table in down section
		dropTableStmt := "DROP TABLE IF EXISTS " + tableName
		if !strings.Contains(contentStr, dropTableStmt) {
			t.Errorf("Migration file %s does not drop table %s in down section", migrationFile, tableName)
		}
	}
}

func TestUsersTableHasRequiredColumns(t *testing.T) {
	migrationsDir := "../../migrations"
	path := filepath.Join(migrationsDir, "00001_create_users_table.sql")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read users migration: %v", err)
	}

	contentStr := string(content)
	requiredColumns := []string{
		"id UUID PRIMARY KEY",
		"email VARCHAR",
		"password_hash VARCHAR",
		"first_name VARCHAR",
		"last_name VARCHAR",
		"role VARCHAR",
		"created_at TIMESTAMP",
		"updated_at TIMESTAMP",
	}

	for _, column := range requiredColumns {
		if !strings.Contains(contentStr, column) {
			t.Errorf("Users table missing required column definition: %s", column)
		}
	}
}

func TestProductsTableHasRequiredColumns(t *testing.T) {
	migrationsDir := "../../migrations"
	path := filepath.Join(migrationsDir, "00004_create_products_table.sql")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read products migration: %v", err)
	}

	contentStr := string(content)
	requiredColumns := []string{
		"id UUID PRIMARY KEY",
		"name VARCHAR",
		"description TEXT",
		"price DECIMAL",
		"category_id UUID",
		"image_url VARCHAR",
		"stock INTEGER",
		"created_at TIMESTAMP",
		"updated_at TIMESTAMP",
	}

	for _, column := range requiredColumns {
		if !strings.Contains(contentStr, column) {
			t.Errorf("Products table missing required column definition: %s", column)
		}
	}

	// Check for foreign key constraint
	if !strings.Contains(contentStr, "FOREIGN KEY (category_id)") {
		t.Error("Products table missing foreign key constraint to categories")
	}
}

func TestOrdersTableHasStatusConstraint(t *testing.T) {
	migrationsDir := "../../migrations"
	path := filepath.Join(migrationsDir, "00006_create_orders_table.sql")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read orders migration: %v", err)
	}

	contentStr := string(content)

	// Check for status constraint with valid values
	requiredStatuses := []string{"pending", "confirmed", "shipped", "delivered"}
	for _, status := range requiredStatuses {
		if !strings.Contains(contentStr, status) {
			t.Errorf("Orders table status constraint missing value: %s", status)
		}
	}
}

func TestCartItemsTableHasUniqueConstraint(t *testing.T) {
	migrationsDir := "../../migrations"
	path := filepath.Join(migrationsDir, "00005_create_cart_items_table.sql")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read cart_items migration: %v", err)
	}

	contentStr := string(content)

	// Check for unique constraint on user_id and product_id
	if !strings.Contains(contentStr, "UNIQUE (user_id, product_id)") {
		t.Error("Cart items table missing unique constraint on (user_id, product_id)")
	}
}
