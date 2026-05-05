package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJunkItemRepo_CRUD(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkItemRepo(conn.DB())

	// Save
	item := &domain.JunkItem{
		CategoryID:     1,
		Path:           "docker://ubuntu:18.04",
		Description:    "Docker image ubuntu:18.04 (unused)",
		SizeBytes:      127000000,
		VerifiedByUser: domain.VerificationPending,
	}
	saved, err := repo.Save(ctx, item)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.True(t, saved.IsPendingReview())

	// FindByID
	found, err := repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, "docker://ubuntu:18.04", found.Path)

	// FindByCategory
	byCat, err := repo.FindByCategory(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, byCat, 1)

	// FindPending
	pending, err := repo.FindPending(ctx)
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	// SetVerification - approve
	err = repo.SetVerification(ctx, saved.ID, domain.VerificationApproved)
	require.NoError(t, err)
	found, err = repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.True(t, found.IsApproved())

	// SetVerification - skip
	err = repo.SetVerification(ctx, saved.ID, domain.VerificationSkipped)
	require.NoError(t, err)
	found, err = repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.False(t, found.IsApproved())
	assert.False(t, found.IsPendingReview())

	// MarkCleaned
	err = repo.MarkCleaned(ctx, saved.ID)
	require.NoError(t, err)
	found, err = repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.True(t, found.IsCleaned())
}

func TestJunkItemRepo_BatchVerification(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkItemRepo(conn.DB())

	// Create 3 items
	var ids []int64
	for i := 0; i < 3; i++ {
		saved, err := repo.Save(ctx, &domain.JunkItem{
			CategoryID:     1,
			Path:           "path/" + string(rune('a'+i)),
			Description:    "item",
			SizeBytes:      100,
			VerifiedByUser: domain.VerificationPending,
		})
		require.NoError(t, err)
		ids = append(ids, saved.ID)
	}

	// Batch approve
	count, err := repo.BatchSetVerification(ctx, ids, domain.VerificationApproved)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify all approved
	for _, id := range ids {
		found, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.True(t, found.IsApproved())
	}
}

func TestJunkItemRepo_BatchEmpty(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkItemRepo(conn.DB())

	count, err := repo.BatchSetVerification(ctx, nil, domain.VerificationApproved)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJunkItemRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewJunkItemRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrJunkItemNotFound)

	err = repo.SetVerification(ctx, 99999, domain.VerificationApproved)
	assert.ErrorIs(t, err, domain.ErrJunkItemNotFound)

	err = repo.MarkCleaned(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrJunkItemNotFound)
}
