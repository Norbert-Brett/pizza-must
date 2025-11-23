package database

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// RunMigrations executes all pending database migrations
func RunMigrations(db *sql.DB, migrationsDir string, logger *zap.Logger) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	logger.Info("Checking for pending migrations...", zap.String("dir", migrationsDir))

	if err := goose.Up(db, migrationsDir); err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Migrations completed successfully")
	return nil
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(db *sql.DB, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	return goose.Status(db, migrationsDir)
}
