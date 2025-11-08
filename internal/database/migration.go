package database

import (
	"context"
	"fmt"
	"os"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes all pending database migrations
// This should be called during application startup
func RunMigrations(dsn string) error {
	// Use versioned migrations directory
	migrationsPath := "file://migrations/versioned"

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Get current version before migration
	oldVersion, _, _ := m.Version()

	// Run all pending migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get current version after migration
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if oldVersion != version {
		logger.Infof(context.Background(), "Database migrated from version %d to %d", oldVersion, version)
	} else {
		logger.Infof(context.Background(), "Database is up to date (version: %d)", version)
	}

	if dirty {
		logger.Warnf(context.Background(), "Database is in dirty state! Manual intervention may be required.")
	}

	return nil
}

// GetMigrationVersion returns the current migration version
func GetMigrationVersion() (uint, bool, error) {
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	migrationsPath := "file://migrations/versioned"

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		return 0, false, err
	}

	return version, dirty, nil
}
