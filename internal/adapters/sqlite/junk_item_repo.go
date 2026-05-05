package sqlite
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure JunkItemRepo implements ports.JunkItemRepository.
var _ ports.JunkItemRepository = (*JunkItemRepo)(nil)

// JunkItemRepo implements ports.JunkItemRepository using SQLite.
type JunkItemRepo struct {
	db *sql.DB
}

// NewJunkItemRepo creates a new JunkItemRepo.
func NewJunkItemRepo(db *sql.DB) *JunkItemRepo {
	return &JunkItemRepo{db: db}
}

func (r *JunkItemRepo) FindByCategory(ctx context.Context, categoryID int64) ([]domain.JunkItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, category_id, volume_id, path, description, size_bytes,
		        last_accessed, scan_id, verified_by_user, cleaned_at, created_at
		 FROM junk_items WHERE category_id = ? ORDER BY size_bytes DESC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJunkItems(rows)
}

func (r *JunkItemRepo) FindPending(ctx context.Context) ([]domain.JunkItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, category_id, volume_id, path, description, size_bytes,
		        last_accessed, scan_id, verified_by_user, cleaned_at, created_at
		 FROM junk_items WHERE verified_by_user = 0 AND cleaned_at IS NULL
		 ORDER BY size_bytes DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJunkItems(rows)
}

func (r *JunkItemRepo) FindByID(ctx context.Context, id int64) (*domain.JunkItem, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, volume_id, path, description, size_bytes,
		        last_accessed, scan_id, verified_by_user, cleaned_at, created_at
		 FROM junk_items WHERE id = ?`, id)
	return scanJunkItem(row)
}

func (r *JunkItemRepo) SetVerification(ctx context.Context, id int64, status domain.VerificationStatus) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE junk_items SET verified_by_user = ? WHERE id = ?`, int(status), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrJunkItemNotFound
	}
	return nil
}

func (r *JunkItemRepo) BatchSetVerification(ctx context.Context, ids []int64, status domain.VerificationStatus) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]any, 0, len(ids)+1)
	args = append(args, int(status))
	for i, id := range ids {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, id)
	}

	query := `UPDATE junk_items SET verified_by_user = ? WHERE id IN (` + placeholders + `)`
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (r *JunkItemRepo) MarkCleaned(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE junk_items SET cleaned_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrJunkItemNotFound
	}
	return nil
}

func (r *JunkItemRepo) Save(ctx context.Context, item *domain.JunkItem) (*domain.JunkItem, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO junk_items (category_id, volume_id, path, description, size_bytes,
		                         last_accessed, scan_id, verified_by_user)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.CategoryID, item.VolumeID, item.Path, item.Description,
		item.SizeBytes, nullableTime(item.LastAccessed), item.ScanID,
		int(item.VerifiedByUser),
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func scanJunkItem(s scanner) (*domain.JunkItem, error) {
	var (
		j            domain.JunkItem
		volumeID     sql.NullInt64
		lastAccessed sql.NullString
		scanID       sql.NullInt64
		cleanedAt    sql.NullString
		createdAt    string
		verified     int
	)

	if err := s.Scan(&j.ID, &j.CategoryID, &volumeID, &j.Path, &j.Description,
		&j.SizeBytes, &lastAccessed, &scanID, &verified, &cleanedAt, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrJunkItemNotFound
		}
		return nil, err
	}

	if volumeID.Valid {
		j.VolumeID = &volumeID.Int64
	}
	if lastAccessed.Valid {
		t, _ := time.Parse(time.RFC3339, lastAccessed.String)
		j.LastAccessed = &t
	}
	if scanID.Valid {
		j.ScanID = &scanID.Int64
	}
	j.VerifiedByUser = domain.VerificationStatus(verified)
	if cleanedAt.Valid {
		t, _ := time.Parse(time.RFC3339, cleanedAt.String)
		j.CleanedAt = &t
	}
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &j, nil
}

func scanJunkItems(rows *sql.Rows) ([]domain.JunkItem, error) {
	var items []domain.JunkItem
	for rows.Next() {
		item, err := scanJunkItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
