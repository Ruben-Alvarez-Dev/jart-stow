package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchRootRepo_CRUD(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewWatchRootRepo(conn.DB())

	// Save
	root := &domain.WatchRoot{
		Path:    "/Users/test/Code",
		Enabled: true,
	}
	saved, err := repo.Save(ctx, root)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.True(t, saved.Enabled)

	// FindByID
	found, err := repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, "/Users/test/Code", found.Path)

	// FindByPath
	found, err = repo.FindByPath(ctx, "/Users/test/Code")
	require.NoError(t, err)
	assert.Equal(t, saved.ID, found.ID)

	// FindAll
	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// SetEnabled
	err = repo.SetEnabled(ctx, saved.ID, false)
	require.NoError(t, err)
	found, err = repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.False(t, found.Enabled)

	// Delete
	err = repo.Delete(ctx, saved.ID)
	require.NoError(t, err)
	_, err = repo.FindByID(ctx, saved.ID)
	assert.ErrorIs(t, err, domain.ErrWatchRootNotFound)
}

func TestWatchRootRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewWatchRootRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrWatchRootNotFound)

	_, err = repo.FindByPath(ctx, "/nonexistent")
	assert.ErrorIs(t, err, domain.ErrWatchRootNotFound)

	err = repo.SetEnabled(ctx, 99999, true)
	assert.ErrorIs(t, err, domain.ErrWatchRootNotFound)

	err = repo.Delete(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrWatchRootNotFound)
}
