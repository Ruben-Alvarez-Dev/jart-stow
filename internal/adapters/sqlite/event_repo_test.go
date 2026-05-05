package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventRepo_LogAndFind(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewEventRepo(conn.DB())

	// Log events
	err := repo.Log(ctx, &domain.DaemonEvent{
		EventType: domain.EventTypeDaemonStarted,
	})
	require.NoError(t, err)

	pid := int64(1)
	err = repo.Log(ctx, &domain.DaemonEvent{
		EventType:  domain.EventTypeProjectDetected,
		ProjectID:  &pid,
		FolderPath: "/Code/new-project",
		Details:    `{"source":"fsevents"}`,
	})
	require.NoError(t, err)

	err = repo.Log(ctx, &domain.DaemonEvent{
		EventType:  domain.EventTypeError,
		FolderPath: "/Code/fail",
		Details:    `{"error":"permission denied"}`,
	})
	require.NoError(t, err)

	// FindRecent
	recent, err := repo.FindRecent(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, recent, 3)

	// FindByType
	errors, err := repo.FindByType(ctx, domain.EventTypeError, 10)
	require.NoError(t, err)
	assert.Len(t, errors, 1)
	assert.Equal(t, "/Code/fail", errors[0].FolderPath)

	// CountToday
	count, err := repo.CountToday(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestEventRepo_FindRecentWithLimit(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewEventRepo(conn.DB())

	for i := 0; i < 10; i++ {
		repo.Log(ctx, &domain.DaemonEvent{
			EventType: domain.EventTypeDaemonStarted,
		})
	}

	recent, err := repo.FindRecent(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, recent, 5)
}

func TestEventRepo_FindRecentDefaultLimit(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewEventRepo(conn.DB())

	for i := 0; i < 5; i++ {
		repo.Log(ctx, &domain.DaemonEvent{
			EventType: domain.EventTypeDaemonStarted,
		})
	}

	// limit=0 should default to 50
	recent, err := repo.FindRecent(ctx, 0)
	require.NoError(t, err)
	assert.Len(t, recent, 5)
}
