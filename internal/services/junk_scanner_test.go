package services

import (
	"context"
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
)

// stubScanner implements JunkScanner for testing.
type stubScanner struct {
	name  string
	items []domain.JunkItem
}

func (s *stubScanner) Name() string { return s.name }
func (s *stubScanner) Scan(_ context.Context, _ domain.JunkCategory) ([]domain.JunkItem, error) {
	return s.items, nil
}

func TestNewJunkScanService(t *testing.T) {
	svc := NewJunkScanService()
	assert.NotNil(t, svc)
}

func TestRegisteredScanners(t *testing.T) {
	svc := NewJunkScanService(
		&stubScanner{name: "docker"},
		&stubScanner{name: "apfs"},
	)
	names := svc.RegisteredScanners()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "docker")
	assert.Contains(t, names, "apfs")
}

func TestScanCategory_Found(t *testing.T) {
	expected := []domain.JunkItem{
		{ID: 1, Path: "/tmp/junk1", CategoryID: 1, SizeBytes: 1024},
	}
	svc := NewJunkScanService(
		&stubScanner{name: "docker", items: expected},
	)

	category := domain.JunkCategory{Name: "docker", Scanner: "docker"}
	items, err := svc.ScanCategory(context.Background(), category)
	assert.NoError(t, err)
	assert.Equal(t, expected, items)
}

func TestScanCategory_NotFound(t *testing.T) {
	svc := NewJunkScanService(
		&stubScanner{name: "docker"},
	)

	category := domain.JunkCategory{Name: "apfs", Scanner: "apfs"}
	_, err := svc.ScanCategory(context.Background(), category)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrJunkCategoryNotFound, err)
}

func TestScanCategory_MultipleScanners(t *testing.T) {
	svc := NewJunkScanService(
		&stubScanner{name: "docker", items: []domain.JunkItem{{ID: 1}}},
		&stubScanner{name: "apfs", items: []domain.JunkItem{{ID: 2}, {ID: 3}}},
	)

	cat1 := domain.JunkCategory{Name: "docker"}
	items1, err := svc.ScanCategory(context.Background(), cat1)
	assert.NoError(t, err)
	assert.Len(t, items1, 1)

	cat2 := domain.JunkCategory{Name: "apfs"}
	items2, err := svc.ScanCategory(context.Background(), cat2)
	assert.NoError(t, err)
	assert.Len(t, items2, 2)
}

func TestScanCategory_MatchedByScannerType(t *testing.T) {
	expected := []domain.JunkItem{{ID: 42, Path: "/tmp/test"}}
	svc := NewJunkScanService(
		&stubScanner{name: "filesystem", items: expected},
	)

	// Category Name differs from scanner name, but Scanner field matches
	category := domain.JunkCategory{
		Name:    "temp_files",
		Scanner: "filesystem",
	}
	items, err := svc.ScanCategory(context.Background(), category)
	assert.NoError(t, err)
	assert.Equal(t, expected, items)
}
