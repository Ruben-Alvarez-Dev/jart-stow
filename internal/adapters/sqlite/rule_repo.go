package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure RuleRepo implements ports.RuleRepository.
var _ ports.RuleRepository = (*RuleRepo)(nil)

// RuleRepo implements ports.RuleRepository using SQLite.
type RuleRepo struct {
	db *sql.DB
}

// NewRuleRepo creates a new RuleRepo.
func NewRuleRepo(db *sql.DB) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) FindAll(ctx context.Context) ([]domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, pattern, max_size_bytes, action,
		        priority, enabled, created_at, updated_at
		 FROM rules ORDER BY priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *RuleRepo) FindByID(ctx context.Context, id int64) (*domain.Rule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, project_id, pattern, max_size_bytes, action,
		        priority, enabled, created_at, updated_at
		 FROM rules WHERE id = ?`, id)
	return scanRule(row)
}

func (r *RuleRepo) FindByProjectID(ctx context.Context, projectID int64) ([]domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, pattern, max_size_bytes, action,
		        priority, enabled, created_at, updated_at
		 FROM rules WHERE project_id = ? ORDER BY priority DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *RuleRepo) FindGlobalRules(ctx context.Context) ([]domain.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, project_id, pattern, max_size_bytes, action,
		        priority, enabled, created_at, updated_at
		 FROM rules WHERE project_id IS NULL ORDER BY priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *RuleRepo) Save(ctx context.Context, rule *domain.Rule) (*domain.Rule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO rules (project_id, pattern, max_size_bytes, action,
		                    priority, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ProjectID, rule.Pattern, rule.MaxSizeBytes, string(rule.Action),
		rule.Priority, boolToInt(rule.Enabled), now, now,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return r.FindByID(ctx, id)
}

func (r *RuleRepo) Update(ctx context.Context, rule *domain.Rule) (*domain.Rule, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`UPDATE rules SET pattern = ?, max_size_bytes = ?, action = ?,
		                  priority = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		rule.Pattern, rule.MaxSizeBytes, string(rule.Action),
		rule.Priority, boolToInt(rule.Enabled), now, rule.ID,
	)
	if err != nil {
		return nil, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, domain.ErrRuleNotFound
	}
	return r.FindByID(ctx, rule.ID)
}

func (r *RuleRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrRuleNotFound
	}
	return nil
}

func scanRule(s scanner) (*domain.Rule, error) {
	var (
		r         domain.Rule
		projectID sql.NullInt64
		createdAt string
		updatedAt string
		action    string
		enabled   int
	)

	if err := s.Scan(&r.ID, &projectID, &r.Pattern, &r.MaxSizeBytes,
		&action, &r.Priority, &enabled, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRuleNotFound
		}
		return nil, err
	}

	if projectID.Valid {
		r.ProjectID = &projectID.Int64
	}
	r.Action = domain.RuleAction(action)
	r.Enabled = enabled != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &r, nil
}

func scanRules(rows *sql.Rows) ([]domain.Rule, error) {
	var rules []domain.Rule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *r)
	}
	return rules, rows.Err()
}

// boolToInt converts a boolean to 0 or 1 for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
