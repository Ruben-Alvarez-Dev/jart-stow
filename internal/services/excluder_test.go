package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProjectRepo implements ports.ProjectRepository for testing.
type stubProjectRepo struct {
	project *domain.Project
}

func (r *stubProjectRepo) FindAll(_ context.Context) ([]domain.Project, error) { return nil, nil }
func (r *stubProjectRepo) FindByPath(_ context.Context, _ string) (*domain.Project, error) {
	if r.project != nil {
		return r.project, nil
	}
	return nil, domain.ErrProjectNotFound
}
func (r *stubProjectRepo) FindByID(_ context.Context, _ int64) (*domain.Project, error) {
	return nil, nil
}
func (r *stubProjectRepo) FindByRootPath(_ context.Context, _ string) ([]domain.Project, error) {
	return nil, nil
}
func (r *stubProjectRepo) Upsert(_ context.Context, p *domain.Project) (*domain.Project, error) {
	return p, nil
}
func (r *stubProjectRepo) UpdateStatus(_ context.Context, _ int64, _ domain.ProjectStatus) error {
	return nil
}
func (r *stubProjectRepo) Delete(_ context.Context, _ int64) error { return nil }

// stubExclusionRepo implements ports.ExclusionRepository for testing.
type stubExclusionRepo struct {
	exclusions map[string]*domain.Exclusion
}

func (r *stubExclusionRepo) FindAll(_ context.Context) ([]domain.Exclusion, error) { return nil, nil }
func (r *stubExclusionRepo) FindByID(_ context.Context, _ int64) (*domain.Exclusion, error) {
	return nil, nil
}
func (r *stubExclusionRepo) FindActive(_ context.Context) ([]domain.Exclusion, error) {
	return nil, nil
}
func (r *stubExclusionRepo) FindByPath(_ context.Context, path string) (*domain.Exclusion, error) {
	if r.exclusions == nil {
		return nil, domain.ErrExclusionNotFound
	}
	ex, ok := r.exclusions[path]
	if !ok {
		return nil, domain.ErrExclusionNotFound
	}
	return ex, nil
}
func (r *stubExclusionRepo) FindByProjectID(_ context.Context, _ int64) ([]domain.Exclusion, error) {
	return nil, nil
}
func (r *stubExclusionRepo) Save(_ context.Context, ex *domain.Exclusion) (*domain.Exclusion, error) {
	return ex, nil
}
func (r *stubExclusionRepo) SaveBulk(_ context.Context, _ []domain.Exclusion) error { return nil }
func (r *stubExclusionRepo) MarkRemoved(_ context.Context, _ int64) error           { return nil }
func (r *stubExclusionRepo) Delete(_ context.Context, _ int64) error                { return nil }

// stubBackupProvider implements ports.BackupProvider for testing.
type stubBackupProvider struct {
	name      string
	available bool
	excluded  map[string]bool
}

func (p *stubBackupProvider) Name() string                                { return p.name }
func (p *stubBackupProvider) IsAvailable(_ context.Context) (bool, error) { return p.available, nil }
func (p *stubBackupProvider) Exclude(_ context.Context, path string) error {
	if p.excluded == nil {
		p.excluded = make(map[string]bool)
	}
	p.excluded[path] = true
	return nil
}
func (p *stubBackupProvider) Remove(_ context.Context, _ string) error { return nil }
func (p *stubBackupProvider) IsExcluded(_ context.Context, path string) (bool, error) {
	return p.excluded[path], nil
}
func (p *stubBackupProvider) ListExcluded(_ context.Context) ([]string, error) { return nil, nil }

func TestNewExcludeService(t *testing.T) {
	svc := NewExcludeService(&stubProjectRepo{}, &stubExclusionRepo{}, nil)
	assert.NotNil(t, svc)
}

func TestExcludeProject(t *testing.T) {
	// Create a temp project with a node_modules directory
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "node_modules", "test.js"), []byte("test"), 0o644))

	scanService := NewScanService(0, nil)
	projectRepo := &stubProjectRepo{
		project: &domain.Project{ID: 1, Name: "myproject", Path: projectDir},
	}
	exclusionRepo := &stubExclusionRepo{exclusions: map[string]*domain.Exclusion{}}
	backup := &stubBackupProvider{name: "time_machine", available: true}

	svc := NewExcludeService(projectRepo, exclusionRepo, scanService, backup)

	count, size, err := svc.ExcludeProject(context.Background(), projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Greater(t, size, int64(0))
}

func TestExcludeProject_AlreadyExcluded(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "myproject")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "node_modules", "test.js"), []byte("test"), 0o644))

	scanService := NewScanService(0, nil)
	projectRepo := &stubProjectRepo{
		project: &domain.Project{ID: 1, Name: "myproject", Path: projectDir},
	}
	artifactsDir := filepath.Join(projectDir, "node_modules")
	exclusionRepo := &stubExclusionRepo{
		exclusions: map[string]*domain.Exclusion{
			artifactsDir: {ID: 1, ProjectID: 1, FolderPath: artifactsDir},
		},
	}
	backup := &stubBackupProvider{name: "time_machine", available: true}

	svc := NewExcludeService(projectRepo, exclusionRepo, scanService, backup)

	// Second scan should skip
	count, _, err := svc.ExcludeProject(context.Background(), projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "already excluded artifact should be skipped")
}

func TestExcludeProject_NoArtifacts(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "emptyproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	scanService := NewScanService(0, nil)
	projectRepo := &stubProjectRepo{
		project: &domain.Project{ID: 1, Name: "emptyproject", Path: projectDir},
	}
	exclusionRepo := &stubExclusionRepo{exclusions: map[string]*domain.Exclusion{}}
	backup := &stubBackupProvider{name: "time_machine", available: true}

	svc := NewExcludeService(projectRepo, exclusionRepo, scanService, backup)

	count, _, err := svc.ExcludeProject(context.Background(), projectDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "no artifacts should be found")
}

func TestDetermineBackupSystem(t *testing.T) {
	ctx := context.Background()

	t.Run("both", func(t *testing.T) {
		tm := &stubBackupProvider{name: "time_machine", excluded: map[string]bool{"/tmp": true}}
		ccc := &stubBackupProvider{name: "carbon_copy_cloner", excluded: map[string]bool{"/tmp": true}}
		result := determineBackupSystem(ctx, []ports.BackupProvider{tm, ccc}, "/tmp")
		assert.Equal(t, domain.BackupSystemBoth, result)
	})

	t.Run("time_machine_only", func(t *testing.T) {
		tm := &stubBackupProvider{name: "time_machine", excluded: map[string]bool{"/tmp": true}}
		ccc := &stubBackupProvider{name: "carbon_copy_cloner", excluded: map[string]bool{}}
		result := determineBackupSystem(ctx, []ports.BackupProvider{tm, ccc}, "/tmp")
		assert.Equal(t, domain.BackupSystemTimeMachine, result)
	})

	t.Run("ccc_only", func(t *testing.T) {
		tm := &stubBackupProvider{name: "time_machine", excluded: map[string]bool{}}
		ccc := &stubBackupProvider{name: "carbon_copy_cloner", excluded: map[string]bool{"/tmp": true}}
		result := determineBackupSystem(ctx, []ports.BackupProvider{tm, ccc}, "/tmp")
		assert.Equal(t, domain.BackupSystemCarbonCopyCloner, result)
	})

	t.Run("none", func(t *testing.T) {
		tm := &stubBackupProvider{name: "time_machine", excluded: map[string]bool{}}
		ccc := &stubBackupProvider{name: "carbon_copy_cloner", excluded: map[string]bool{}}
		result := determineBackupSystem(ctx, []ports.BackupProvider{tm, ccc}, "/tmp")
		assert.Equal(t, domain.BackupSystemBoth, result) // defaults to both when unknown
	})

	t.Run("unknown_backup_error", func(t *testing.T) {
		unknown := &stubBackupProvider{name: "unknown_system", excluded: map[string]bool{"/tmp": true}}
		result := determineBackupSystem(ctx, []ports.BackupProvider{unknown}, "/tmp")
		assert.Equal(t, domain.BackupSystemBoth, result)
	})
}
