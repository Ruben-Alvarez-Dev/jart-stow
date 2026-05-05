package sqlite
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure JunkCategoryRepo implements ports.JunkCategoryRepository.
var _ ports.JunkCategoryRepository = (*JunkCategoryRepo)(nil)

// JunkCategoryRepo implements ports.JunkCategoryRepository using SQLite.
type JunkCategoryRepo struct {
	db *sql.DB
}

// NewJunkCategoryRepo creates a new JunkCategoryRepo.
func NewJunkCategoryRepo(db *sql.DB) *JunkCategoryRepo {
	return &JunkCategoryRepo{db: db}
}

func (r *JunkCategoryRepo) FindAll(ctx context.Context) ([]domain.JunkCategory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, scanner, verify_required, enabled, created_at
		 FROM junk_categories ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.JunkCategory
	for rows.Next() {
		c, err := scanJunkCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *c)
	}
	return categories, rows.Err()
}

func (r *JunkCategoryRepo) FindByID(ctx context.Context, id int64) (*domain.JunkCategory, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, scanner, verify_required, enabled, created_at
		 FROM junk_categories WHERE id = ?`, id)
	return scanJunkCategory(row)
}

func (r *JunkCategoryRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE junk_categories SET enabled = ? WHERE id = ?`, boolToInt(enabled), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrJunkCategoryNotFound
	}
	return nil
}

func (r *JunkCategoryRepo) InsertDefaults(ctx context.Context) error {
	categories := domain.DefaultCategories()
	for _, c := range categories {
		_, err := r.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO junk_categories (name, scanner, verify_required, enabled)
			 VALUES (?, ?, ?, ?)`,
			c.Name, string(c.Scanner), boolToInt(c.VerifyRequired), boolToInt(c.Enabled),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func scanJunkCategory(s scanner) (*domain.JunkCategory, error) {
	var (
		c          domain.JunkCategory
		scanner    string
		createdAt  string
		verifyReq  int
		enabled    int
	)

	if err := s.Scan(&c.ID, &c.Name, &scanner, &verifyReq, &enabled, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrJunkCategoryNotFound
		}
		return nil, err
	}

	c.Scanner = domain.ScannerName(scanner)
	c.VerifyRequired = verifyReq != 0
	c.Enabled = enabled != 0
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &c, nil
}
