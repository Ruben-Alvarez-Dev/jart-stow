// Package sqlite implements the SQLite adapter for Jart-Stow.
// It provides repository implementations and migration management.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a single numbered migration file.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// LoadMigrations reads all embedded migration files and returns them sorted by version.
func LoadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("parsing version from %s: %w", entry.Name(), err)
		}
		content, err := migrationsFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading migration file %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// parseVersion extracts the leading integer from a migration filename like "001_initial_schema.sql".
func parseVersion(filename string) (int, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}
	return strconv.Atoi(parts[0])
}

// Migrate ensures the database schema is up to date by applying all pending migrations.
func Migrate(ctx context.Context, db *sql.DB) error {
	currentVersion, err := getSchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("getting schema version: %w", err)
	}

	migrations, err := LoadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return fmt.Errorf("applying migration %d (%s): %w", m.Version, m.Name, err)
		}
	}

	return nil
}

// getSchemaVersion reads the current schema version from the database.
func getSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int
	err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("reading user_version: %w", err)
	}
	return version, nil
}

// applyMigration executes a single migration and updates the schema version.
func applyMigration(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("executing migration SQL: %w", err)
	}

	setVersionSQL := fmt.Sprintf("PRAGMA user_version = %d", m.Version)
	if _, err := tx.ExecContext(ctx, setVersionSQL); err != nil {
		return fmt.Errorf("setting user_version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}

	return nil
}
