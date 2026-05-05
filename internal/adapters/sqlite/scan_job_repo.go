package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure ScanJobRepo implements ports.ScanJobRepository.
var _ ports.ScanJobRepository = (*ScanJobRepo)(nil)

// ScanJobRepo implements ports.ScanJobRepository using SQLite.
type ScanJobRepo struct {
	db *sql.DB
}

// NewScanJobRepo creates a new ScanJobRepo.
func NewScanJobRepo(db *sql.DB) *ScanJobRepo {
	return &ScanJobRepo{db: db}
}

func (r *ScanJobRepo) Create(ctx context.Context, job *domain.ScanJob) (*domain.ScanJob, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO scan_jobs (project_id, status, folders_found, total_size_bytes, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		job.ProjectID, string(job.Status), job.FoldersFound, job.TotalSizeBytes, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func (r *ScanJobRepo) MarkCompleted(ctx context.Context, id int64, foldersFound int, totalSize int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE scan_jobs SET status = 'completed', folders_found = ?,
		 total_size_bytes = ?, finished_at = ? WHERE id = ?`,
		foldersFound, totalSize, now, id)
	return err
}

func (r *ScanJobRepo) MarkFailed(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE scan_jobs SET status = 'failed', finished_at = ? WHERE id = ?`,
		now, id)
	return err
}

func (r *ScanJobRepo) FindByID(ctx context.Context, id int64) (*domain.ScanJob, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, status, folders_found, total_size_bytes,
		        started_at, finished_at
		 FROM scan_jobs WHERE id = ?`, id)
	return scanScanJob(row)
}

func (r *ScanJobRepo) FindByProjectID(ctx context.Context, projectID int64) ([]domain.ScanJob, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, status, folders_found, total_size_bytes,
		        started_at, finished_at
		 FROM scan_jobs WHERE project_id = ? ORDER BY started_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.ScanJob
	for rows.Next() {
		j, err := scanScanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *j)
	}
	return jobs, rows.Err()
}

func scanScanJob(s scanner) (*domain.ScanJob, error) {
	var (
		j          domain.ScanJob
		startedAt  string
		finishedAt sql.NullString
		status     string
	)

	if err := s.Scan(&j.ID, &j.ProjectID, &status, &j.FoldersFound,
		&j.TotalSizeBytes, &startedAt, &finishedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrScanJobNotFound
		}
		return nil, err
	}

	j.Status = domain.ScanStatus(status)
	j.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	if finishedAt.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAt.String)
		j.FinishedAt = &t
	}

	return &j, nil
}
