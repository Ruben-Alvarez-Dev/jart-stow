package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure ExclusionRepo implements ports.ExclusionRepository.
var _ ports.ExclusionRepository = (*ExclusionRepo)(nil)

// ExclusionRepo implements ports.ExclusionRepository using SQLite.
type ExclusionRepo struct {
	db *sql.DB
}

// NewExclusionRepo creates a new ExclusionRepo.
func NewExclusionRepo(db *sql.DB) *ExclusionRepo {
	return &ExclusionRepo{db: db}
}

func (r *ExclusionRepo) FindAll(ctx context.Context) ([]domain.Exclusion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, folder_path, pattern_matched, backup_system,
		        size_bytes, applied_at, removed_at, created_at
		 FROM exclusions ORDER BY applied_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExclusions(rows)
}

func (r *ExclusionRepo) FindByID(ctx context.Context, id int64) (*domain.Exclusion, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, folder_path, pattern_matched, backup_system,
		        size_bytes, applied_at, removed_at, created_at
		 FROM exclusions WHERE id = ?`, id)
	return scanExclusion(row)
}

func (r *ExclusionRepo) FindByProjectID(ctx context.Context, projectID int64) ([]domain.Exclusion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, folder_path, pattern_matched, backup_system,
		        size_bytes, applied_at, removed_at, created_at
		 FROM exclusions WHERE project_id = ? ORDER BY applied_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExclusions(rows)
}

func (r *ExclusionRepo) FindActive(ctx context.Context) ([]domain.Exclusion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, folder_path, pattern_matched, backup_system,
		        size_bytes, applied_at, removed_at, created_at
		 FROM exclusions WHERE removed_at IS NULL ORDER BY applied_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExclusions(rows)
}

func (r *ExclusionRepo) FindByPath(ctx context.Context, folderPath string) (*domain.Exclusion, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, folder_path, pattern_matched, backup_system,
		        size_bytes, applied_at, removed_at, created_at
		 FROM exclusions WHERE folder_path = ? AND removed_at IS NULL`, folderPath)
	return scanExclusion(row)
}

func (r *ExclusionRepo) Save(ctx context.Context, exclusion *domain.Exclusion) (*domain.Exclusion, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO exclusions (project_id, folder_path, pattern_matched, backup_system,
		                         size_bytes, applied_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		exclusion.ProjectID, exclusion.FolderPath, exclusion.PatternMatched,
		string(exclusion.BackupSystem), exclusion.SizeBytes, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func (r *ExclusionRepo) MarkRemoved(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE exclusions SET removed_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrExclusionNotFound
	}
	return nil
}

func scanExclusion(s scanner) (*domain.Exclusion, error) {
	var (
		e         domain.Exclusion
		appliedAt string
		removedAt sql.NullString
		createdAt string
		backupSys string
	)

	if err := s.Scan(&e.ID, &e.ProjectID, &e.FolderPath, &e.PatternMatched,
		&backupSys, &e.SizeBytes, &appliedAt, &removedAt, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrExclusionNotFound
		}
		return nil, err
	}

	e.BackupSystem = domain.BackupSystem(backupSys)
	e.AppliedAt, _ = time.Parse(time.RFC3339, appliedAt)
	if removedAt.Valid {
		t, _ := time.Parse(time.RFC3339, removedAt.String)
		e.RemovedAt = &t
	}
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &e, nil
}

func scanExclusions(rows *sql.Rows) ([]domain.Exclusion, error) {
	var exclusions []domain.Exclusion
	for rows.Next() {
		e, err := scanExclusion(rows)
		if err != nil {
			return nil, err
		}
		exclusions = append(exclusions, *e)
	}
	return exclusions, rows.Err()
}
