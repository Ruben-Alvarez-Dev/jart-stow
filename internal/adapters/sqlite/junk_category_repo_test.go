package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJunkCategoryRepo_Defaults(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkCategoryRepo(conn.DB())

	// Defaults should already be inserted from the initial migration
	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 10)

	// Verify key categories exist
	names := make(map[string]bool)
	for _, c := range all {
		names[c.Name] = true
	}
	assert.True(t, names["unused_docker_images"])
	assert.True(t, names["brew_cache"])

	// brew_cache should have verify_required=false
	for _, c := range all {
		if c.Name == "brew_cache" {
			assert.False(t, c.VerifyRequired)
		}
	}

	// All should start disabled
	for _, c := range all {
		assert.False(t, c.Enabled)
	}
}

func TestJunkCategoryRepo_SetEnabled(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkCategoryRepo(conn.DB())

	all, _ := repo.FindAll(ctx)
	firstID := all[0].ID

	err := repo.SetEnabled(ctx, firstID, true)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, firstID)
	require.NoError(t, err)
	assert.True(t, found.Enabled)

	err = repo.SetEnabled(ctx, firstID, false)
	require.NoError(t, err)

	found, err = repo.FindByID(ctx, firstID)
	require.NoError(t, err)
	assert.False(t, found.Enabled)
}

func TestJunkCategoryRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkCategoryRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrJunkCategoryNotFound)

	err = repo.SetEnabled(ctx, 99999, true)
	assert.ErrorIs(t, err, domain.ErrJunkCategoryNotFound)
}

func TestJunkCategoryRepo_InsertDefaultsIdempotent(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkCategoryRepo(conn.DB())

	// Insert defaults again (should not duplicate due to INSERT OR IGNORE)
	err := repo.InsertDefaults(ctx)
	require.NoError(t, err)

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 10) // Still 10, not 20
}
