package sqlite

import (
	"testing"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectRepo_CRUD(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewProjectRepo(conn.DB())

	// Create
	project := &domain.Project{
		Path:     "/Users/test/Code/my-project",
		Name:     "my-project",
		RootPath: "/Users/test/Code",
		Status:   domain.ProjectStatusActive,
	}
	created, err := repo.Upsert(ctx, project)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "my-project", created.Name)
	assert.NotZero(t, created.CreatedAt)

	// FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Path, found.Path)

	// FindByPath
	found, err = repo.FindByPath(ctx, "/Users/test/Code/my-project")
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)

	// FindAll
	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// FindByRootPath
	byRoot, err := repo.FindByRootPath(ctx, "/Users/test/Code")
	require.NoError(t, err)
	assert.Len(t, byRoot, 1)

	// UpdateStatus
	err = repo.UpdateStatus(ctx, created.ID, domain.ProjectStatusIgnored)
	require.NoError(t, err)
	found, err = repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ProjectStatusIgnored, found.Status)

	// Upsert (update existing)
	project.Path = "/Users/test/Code/my-project"
	project.Name = "my-project-renamed"
	project.Status = domain.ProjectStatusActive
	updated, err := repo.Upsert(ctx, project)
	require.NoError(t, err)
	assert.Equal(t, "my-project-renamed", updated.Name)
	assert.Equal(t, created.ID, updated.ID) // Same ID, updated

	// Delete
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)
	_, err = repo.FindByID(ctx, created.ID)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)

	// Delete non-existent
	err = repo.Delete(ctx, 99999)
	require.NoError(t, err) // SQLite DELETE without WHERE doesn't error on zero rows
}

func TestProjectRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewProjectRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)

	_, err = repo.FindByPath(ctx, "/nonexistent")
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestProjectRepo_LastScanned(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewProjectRepo(conn.DB())

	now := time.Now().UTC().Truncate(time.Second)
	project := &domain.Project{
		Path:        "/Users/test/Code/scanned-project",
		Name:        "scanned-project",
		RootPath:    "/Users/test/Code",
		Status:      domain.ProjectStatusActive,
		LastScanned: &now,
	}
	created, err := repo.Upsert(ctx, project)
	require.NoError(t, err)
	require.NotNil(t, created.LastScanned)
	assert.WithinDuration(t, now, *created.LastScanned, time.Second)
}
