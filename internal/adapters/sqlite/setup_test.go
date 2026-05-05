package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupTestDB creates a temporary SQLite database for testing, runs migrations,
// and returns the connection.
func setupTestDB(t *testing.T) *Connection {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	conn, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return conn
}

// freshContext returns a background context for tests.
func freshContext() context.Context {
	return context.Background()
}
