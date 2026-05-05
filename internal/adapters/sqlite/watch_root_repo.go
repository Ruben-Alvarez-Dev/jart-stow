package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure WatchRootRepo implements ports.WatchRootRepository.
var _ ports.WatchRootRepository = (*WatchRootRepo)(nil)

// WatchRootRepo implements ports.WatchRootRepository using SQLite.
type WatchRootRepo struct {
	db *sql.DB
}

// NewWatchRootRepo creates a new WatchRootRepo.
func NewWatchRootRepo(db *sql.DB) *WatchRootRepo {
	return &WatchRootRepo{db: db}
}

func (r *WatchRootRepo) FindAll(ctx context.Context) ([]domain.WatchRoot, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, path, volume_uuid, enabled, created_at
		 FROM watch_roots ORDER BY path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWatchRoots(rows)
}

func (r *WatchRootRepo) FindByID(ctx context.Context, id int64) (*domain.WatchRoot, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, path, volume_uuid, enabled, created_at
		 FROM watch_roots WHERE id = ?`, id)
	return scanWatchRoot(row)
}

func (r *WatchRootRepo) FindByPath(ctx context.Context, path string) (*domain.WatchRoot, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, path, volume_uuid, enabled, created_at
		 FROM watch_roots WHERE path = ?`, path)
	return scanWatchRoot(row)
}

func (r *WatchRootRepo) Save(ctx context.Context, root *domain.WatchRoot) (*domain.WatchRoot, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO watch_roots (path, volume_uuid, enabled, created_at)
		 VALUES (?, ?, ?, ?)`,
		root.Path, nullableString(root.VolumeUUID), boolToInt(root.Enabled), now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func (r *WatchRootRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE watch_roots SET enabled = ? WHERE id = ?`, boolToInt(enabled), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrWatchRootNotFound
	}
	return nil
}

func (r *WatchRootRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM watch_roots WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrWatchRootNotFound
	}
	return nil
}

func scanWatchRoot(s scanner) (*domain.WatchRoot, error) {
	var (
		r          domain.WatchRoot
		volumeUUID sql.NullString
		createdAt  string
		enabled    int
	)

	if err := s.Scan(&r.ID, &r.Path, &volumeUUID, &enabled, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrWatchRootNotFound
		}
		return nil, err
	}

	if volumeUUID.Valid {
		r.VolumeUUID = volumeUUID.String
	}
	r.Enabled = enabled != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &r, nil
}

func scanWatchRoots(rows *sql.Rows) ([]domain.WatchRoot, error) {
	var roots []domain.WatchRoot
	for rows.Next() {
		r, err := scanWatchRoot(rows)
		if err != nil {
			return nil, err
		}
		roots = append(roots, *r)
	}
	return roots, rows.Err()
}
