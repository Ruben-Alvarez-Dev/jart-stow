package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnection_OpenAndClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	conn, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	assert.Equal(t, dbPath, conn.Path())
	assert.NotNil(t, conn.DB())

	err = conn.Close()
	require.NoError(t, err)
}

func TestConnection_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/subdir/nested/test.db"

	conn, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	defer conn.Close()

	assert.Equal(t, dbPath, conn.Path())
}

func TestMigration_RunsOnOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	conn, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	defer conn.Close()

	// Verify schema version is set
	var version int
	err = conn.DB().QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)

	// Verify tables exist
	rows, err := conn.DB().QueryContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	expectedTables := []string{
		"cleanup_jobs", "daemon_events", "exclusions",
		"junk_categories", "junk_items", "projects",
		"rules", "scan_jobs", "watch_roots",
	}
	for _, expected := range expectedTables {
		assert.Contains(t, tables, expected, "table %s should exist", expected)
	}

	// Verify junk categories were seeded
	var count int
	err = conn.DB().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM junk_categories").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestConnection_ReopenDoesNotReapplyMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Open and close
	conn1, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	conn1.Close()

	// Reopen - should not fail (idempotent migration)
	conn2, err := NewConnection(context.Background(), dbPath)
	require.NoError(t, err)
	defer conn2.Close()

	var version int
	conn2.DB().QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version)
	assert.Equal(t, 1, version)
}

func TestNullableHelpers(t *testing.T) {
	// nullableTime with nil
	nt := nullableTime(nil)
	assert.False(t, nt.Valid)

	// nullableTime with value
	now := time.Now().UTC().Truncate(time.Second)
	nt = nullableTime(&now)
	assert.True(t, nt.Valid)
	assert.Contains(t, nt.String, now.Format(time.RFC3339)[:10]) // Date part matches

	// nullableTime with zero time
	zero := time.Time{}
	nt = nullableTime(&zero)
	assert.False(t, nt.Valid)

	// nullableString with empty
	ns := nullableString("")
	assert.False(t, ns.Valid)

	// nullableString with value
	ns = nullableString("hello")
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)
}
