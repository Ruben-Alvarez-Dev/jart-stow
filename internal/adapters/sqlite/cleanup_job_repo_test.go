package sqlite

import (
	"testing"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupJobRepo_SaveAndFind(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewCleanupJobRepo(conn.DB())

	now := time.Now().UTC()
	categoryID := int64(1)

	job := &domain.CleanupJob{
		CategoryID:     &categoryID,
		ItemsCount:     5,
		TotalSizeBytes: 2400000000,
		StartedAt:      now,
		FinishedAt:     now.Add(time.Minute),
	}
	saved, err := repo.Save(ctx, job)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.Equal(t, 5, saved.ItemsCount)
	assert.Equal(t, int64(2400000000), saved.TotalSizeBytes)

	found, err := repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, saved.ItemsCount, found.ItemsCount)
	assert.NotNil(t, found.CategoryID)
	assert.Equal(t, int64(1), *found.CategoryID)
}

func TestCleanupJobRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewCleanupJobRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrScanJobNotFound) // Uses same error convention
}
