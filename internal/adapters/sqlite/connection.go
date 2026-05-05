package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Connection manages the SQLite database lifecycle.
type Connection struct {
	db     *sql.DB
	dbPath string
}

// NewConnection opens a SQLite database at the given path, applies pragmas,
// and runs all pending migrations. The parent directory is created if needed.
func NewConnection(ctx context.Context, dbPath string) (*Connection, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Connection{db: db, dbPath: dbPath}, nil
}

// DB returns the underlying *sql.DB connection.
func (c *Connection) DB() *sql.DB {
	return c.db
}

// Close closes the database connection.
func (c *Connection) Close() error {
	return c.db.Close()
}

// Path returns the database file path.
func (c *Connection) Path() string {
	return c.dbPath
}
