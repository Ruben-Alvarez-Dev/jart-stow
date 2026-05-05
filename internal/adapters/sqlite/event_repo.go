package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure EventRepo implements ports.EventRepository.
var _ ports.EventRepository = (*EventRepo)(nil)

// EventRepo implements ports.EventRepository using SQLite.
type EventRepo struct {
	db *sql.DB
}

// NewEventRepo creates a new EventRepo.
func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Log(ctx context.Context, event *domain.DaemonEvent) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO daemon_events (event_type, project_id, folder_path, details)
		 VALUES (?, ?, ?, ?)`,
		string(event.EventType), event.ProjectID, event.FolderPath, event.Details,
	)
	return err
}

func (r *EventRepo) FindRecent(ctx context.Context, limit int) ([]domain.DaemonEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, event_type, project_id, folder_path, details, created_at
		 FROM daemon_events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *EventRepo) FindByType(ctx context.Context, eventType domain.EventType, limit int) ([]domain.DaemonEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, event_type, project_id, folder_path, details, created_at
		 FROM daemon_events WHERE event_type = ? ORDER BY created_at DESC LIMIT ?`,
		string(eventType), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *EventRepo) CountToday(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM daemon_events
		 WHERE created_at >= datetime('now', 'start of day')`,
	).Scan(&count)
	return count, err
}

func scanEvent(s scanner) (*domain.DaemonEvent, error) {
	var (
		e          domain.DaemonEvent
		projectID  sql.NullInt64
		folderPath sql.NullString
		details    sql.NullString
		createdAt  string
		eventType  string
	)

	if err := s.Scan(&e.ID, &eventType, &projectID, &folderPath,
		&details, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}

	e.EventType = domain.EventType(eventType)
	if projectID.Valid {
		e.ProjectID = &projectID.Int64
	}
	if folderPath.Valid {
		e.FolderPath = folderPath.String
	}
	if details.Valid {
		e.Details = details.String
	}
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &e, nil
}

func scanEvents(rows *sql.Rows) ([]domain.DaemonEvent, error) {
	var events []domain.DaemonEvent
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, rows.Err()
}
