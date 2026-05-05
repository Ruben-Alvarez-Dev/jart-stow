package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanJobRepo_Lifecycle(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewScanJobRepo(conn.DB())

	// Create
	job, err := repo.Create(ctx, &domain.ScanJob{
		ProjectID: 1,
		Status:    domain.ScanStatusRunning,
	})
	require.NoError(t, err)
	assert.NotZero(t, job.ID)
	assert.Equal(t, domain.ScanStatusRunning, job.Status)

	// MarkCompleted
	err = repo.MarkCompleted(ctx, job.ID, 12, 8400000000)
	require.NoError(t, err)
	completed, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ScanStatusCompleted, completed.Status)
	assert.Equal(t, 12, completed.FoldersFound)
	assert.Equal(t, int64(8400000000), completed.TotalSizeBytes)
	assert.True(t, completed.IsFinished())

	// Create another and MarkFailed
	job2, _ := repo.Create(ctx, &domain.ScanJob{
		ProjectID: 2,
		Status:    domain.ScanStatusRunning,
	})
	err = repo.MarkFailed(ctx, job2.ID)
	require.NoError(t, err)
	failed, err := repo.FindByID(ctx, job2.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ScanStatusFailed, failed.Status)
	assert.True(t, failed.IsFinished())
}

func TestScanJobRepo_FindByProjectID(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewScanJobRepo(conn.DB())

	repo.Create(ctx, &domain.ScanJob{ProjectID: 10, Status: domain.ScanStatusRunning})
	repo.Create(ctx, &domain.ScanJob{ProjectID: 10, Status: domain.ScanStatusCompleted})
	repo.Create(ctx, &domain.ScanJob{ProjectID: 20, Status: domain.ScanStatusRunning})

	jobs, err := repo.FindByProjectID(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)

	jobs, err = repo.FindByProjectID(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, jobs, 0)
}

func TestScanJobRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewScanJobRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrScanJobNotFound)
}
