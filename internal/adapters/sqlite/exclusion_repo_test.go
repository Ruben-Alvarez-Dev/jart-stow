package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExclusionRepo_CRUD(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	projectRepo := NewProjectRepo(conn.DB())
	exclusionRepo := NewExclusionRepo(conn.DB())

	// Create a parent project
	proj, err := projectRepo.Upsert(ctx, &domain.Project{
		Path: "/Code/test-proj", Name: "test-proj", RootPath: "/Code",
		Status: domain.ProjectStatusActive,
	})
	require.NoError(t, err)

	// Save exclusion
	exclusion := &domain.Exclusion{
		ProjectID:      proj.ID,
		FolderPath:     "/Code/test-proj/node_modules",
		PatternMatched: "node_modules",
		BackupSystem:   domain.BackupSystemBoth,
		SizeBytes:      2300000000,
	}
	saved, err := exclusionRepo.Save(ctx, exclusion)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.True(t, saved.IsActive())

	// FindByID
	found, err := exclusionRepo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, "node_modules", found.PatternMatched)
	assert.Equal(t, domain.BackupSystemBoth, found.BackupSystem)

	// FindByProjectID
	byProject, err := exclusionRepo.FindByProjectID(ctx, proj.ID)
	require.NoError(t, err)
	assert.Len(t, byProject, 1)

	// FindActive
	active, err := exclusionRepo.FindActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 1)

	// FindByPath
	byPath, err := exclusionRepo.FindByPath(ctx, "/Code/test-proj/node_modules")
	require.NoError(t, err)
	assert.Equal(t, saved.ID, byPath.ID)

	// MarkRemoved
	err = exclusionRepo.MarkRemoved(ctx, saved.ID)
	require.NoError(t, err)

	found, err = exclusionRepo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.RemovedAt)
	assert.False(t, found.IsActive())

	// Active list should now be empty
	active, err = exclusionRepo.FindActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 0)
}

func TestExclusionRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewExclusionRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrExclusionNotFound)

	_, err = repo.FindByPath(ctx, "/nonexistent")
	assert.ErrorIs(t, err, domain.ErrExclusionNotFound)
}

func TestExclusionRepo_MarkRemovedNotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewExclusionRepo(conn.DB())

	err := repo.MarkRemoved(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrExclusionNotFound)
}
