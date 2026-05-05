package sqlite
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure CleanupJobRepo implements ports.CleanupJobRepository.
var _ ports.CleanupJobRepository = (*CleanupJobRepo)(nil)

// CleanupJobRepo implements ports.CleanupJobRepository using SQLite.
type CleanupJobRepo struct {
	db *sql.DB
}

// NewCleanupJobRepo creates a new CleanupJobRepo.
func NewCleanupJobRepo(db *sql.DB) *CleanupJobRepo {
	return &CleanupJobRepo{db: db}
}

func (r *CleanupJobRepo) Save(ctx context.Context, job *domain.CleanupJob) (*domain.CleanupJob, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO cleanup_jobs (category_id, items_count, total_size_bytes, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?)`,
		job.CategoryID, job.ItemsCount, job.TotalSizeBytes,
		job.StartedAt.Format(time.RFC3339), job.FinishedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func (r *CleanupJobRepo) FindByID(ctx context.Context, id int64) (*domain.CleanupJob, error) {
	var (
		j          domain.CleanupJob
		categoryID sql.NullInt64
		startedAt  string
		finishedAt string
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, items_count, total_size_bytes, started_at, finished_at
		 FROM cleanup_jobs WHERE id = ?`, id,
	).Scan(&j.ID, &categoryID, &j.ItemsCount, &j.TotalSizeBytes, &startedAt, &finishedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrScanJobNotFound
		}
		return nil, err
	}

	if categoryID.Valid {
		j.CategoryID = &categoryID.Int64
	}
	j.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	j.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)

	return &j, nil
}
