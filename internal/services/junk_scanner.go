package services

import (
	"context"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
)

// JunkScanner defines the interface for scanning a specific category of system junk.
// Each scanner implementation handles one category (Docker, APFS, filesystem, cache, etc.).
type JunkScanner interface {
	// Scan discovers junk items for the given category and returns them.
	Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error)

	// Name returns the scanner identifier matching the category's scanner field.
	Name() string
}

// JunkScanService orchestrates scanning across multiple junk categories.
type JunkScanService struct {
	scanners map[string]JunkScanner
}

// NewJunkScanService creates a new JunkScanService with the given scanners.
func NewJunkScanService(scanners ...JunkScanner) *JunkScanService {
	m := make(map[string]JunkScanner, len(scanners))
	for _, s := range scanners {
		m[s.Name()] = s
	}
	return &JunkScanService{scanners: m}
}

// ScanCategory runs a junk scan for a specific category if its scanner is registered.
func (s *JunkScanService) ScanCategory(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	scanner, ok := s.scanners[category.Name]
	if !ok {
		// Try matching by scanner type
		scanner, ok = s.scanners[string(category.Scanner)]
	}
	if !ok {
		return nil, domain.ErrJunkCategoryNotFound
	}
	return scanner.Scan(ctx, category)
}

// RegisteredScanners returns the names of all registered scanners.
func (s *JunkScanService) RegisteredScanners() []string {
	names := make([]string, 0, len(s.scanners))
	for name := range s.scanners {
		names = append(names, name)
	}
	return names
}
