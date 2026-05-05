package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure ProjectRepo implements ports.ProjectRepository.
var _ ports.ProjectRepository = (*ProjectRepo)(nil)

// ProjectRepo implements ports.ProjectRepository using SQLite.
type ProjectRepo struct {
	db *sql.DB
}

// NewProjectRepo creates a new ProjectRepo.
func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) FindAll(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, path, name, root_path, last_scanned, status, created_at, updated_at
		 FROM projects ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) FindByID(ctx context.Context, id int64) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, path, name, root_path, last_scanned, status, created_at, updated_at
		 FROM projects WHERE id = ?`, id)
	return scanProject(row)
}

func (r *ProjectRepo) FindByPath(ctx context.Context, path string) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, path, name, root_path, last_scanned, status, created_at, updated_at
		 FROM projects WHERE path = ?`, path)
	return scanProject(row)
}

func (r *ProjectRepo) FindByRootPath(ctx context.Context, rootPath string) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, path, name, root_path, last_scanned, status, created_at, updated_at
		 FROM projects WHERE root_path = ? ORDER BY name ASC`, rootPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) Upsert(ctx context.Context, project *domain.Project) (*domain.Project, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	project.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (path, name, root_path, last_scanned, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET
		   name = excluded.name,
		   root_path = excluded.root_path,
		   last_scanned = excluded.last_scanned,
		   status = excluded.status,
		   updated_at = excluded.updated_at`,
		project.Path, project.Name, project.RootPath,
		nullableTime(project.LastScanned), string(project.Status),
		now, now,
	)
	if err != nil {
		return nil, err
	}

	// Re-read to get the ID
	return r.FindByPath(ctx, project.Path)
}

func (r *ProjectRepo) UpdateStatus(ctx context.Context, id int64, status domain.ProjectStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, id)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

// scanner is an interface for scanning rows (both *sql.Row and *sql.Rows implement it).
type scanner interface {
	Scan(dest ...any) error
}

func scanProject(s scanner) (*domain.Project, error) {
	var (
		p           domain.Project
		lastScanned sql.NullString
		createdAt   string
		updatedAt   string
		status      string
	)

	if err := s.Scan(&p.ID, &p.Path, &p.Name, &p.RootPath,
		&lastScanned, &status, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}

	p.Status = domain.ProjectStatus(status)
	if lastScanned.Valid {
		t, err := time.Parse(time.RFC3339, lastScanned.String)
		if err == nil {
			p.LastScanned = &t
		}
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &p, nil
}
