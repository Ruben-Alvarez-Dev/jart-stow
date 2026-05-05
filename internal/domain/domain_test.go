package domain

import (
	"testing"
	"time"
)

func TestExclusion_IsActive(t *testing.T) {
	active := Exclusion{RemovedAt: nil}
	if !active.IsActive() {
		t.Error("exclusion without RemovedAt should be active")
	}

	now := time.Now()
	removed := Exclusion{RemovedAt: &now}
	if removed.IsActive() {
		t.Error("exclusion with RemovedAt should not be active")
	}
}

func TestRule_IsGlobal(t *testing.T) {
	global := Rule{ProjectID: nil}
	if !global.IsGlobal() {
		t.Error("rule with nil ProjectID should be global")
	}

	pid := int64(1)
	local := Rule{ProjectID: &pid}
	if local.IsGlobal() {
		t.Error("rule with ProjectID should not be global")
	}
}

func TestScanJob_IsFinished(t *testing.T) {
	tests := []struct {
		name     string
		status   ScanStatus
		expected bool
	}{
		{"running is not finished", ScanStatusRunning, false},
		{"completed is finished", ScanStatusCompleted, true},
		{"failed is finished", ScanStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := ScanJob{Status: tt.status}
			if got := job.IsFinished(); got != tt.expected {
				t.Errorf("IsFinished() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestJunkItem_Verification(t *testing.T) {
	t.Run("pending review", func(t *testing.T) {
		item := JunkItem{VerifiedByUser: VerificationPending}
		if !item.IsPendingReview() {
			t.Error("VerificationPending should be pending review")
		}
		if item.IsApproved() {
			t.Error("VerificationPending should not be approved")
		}
	})

	t.Run("approved", func(t *testing.T) {
		item := JunkItem{VerifiedByUser: VerificationApproved}
		if item.IsPendingReview() {
			t.Error("VerificationApproved should not be pending review")
		}
		if !item.IsApproved() {
			t.Error("VerificationApproved should be approved")
		}
	})

	t.Run("skipped", func(t *testing.T) {
		item := JunkItem{VerifiedByUser: VerificationSkipped}
		if item.IsPendingReview() {
			t.Error("VerificationSkipped should not be pending review")
		}
		if item.IsApproved() {
			t.Error("VerificationSkipped should not be approved")
		}
	})
}

func TestJunkItem_IsCleaned(t *testing.T) {
	notCleaned := JunkItem{CleanedAt: nil}
	if notCleaned.IsCleaned() {
		t.Error("item without CleanedAt should not be cleaned")
	}

	now := time.Now()
	cleaned := JunkItem{CleanedAt: &now}
	if !cleaned.IsCleaned() {
		t.Error("item with CleanedAt should be cleaned")
	}
}

func TestDefaultCategories(t *testing.T) {
	categories := DefaultCategories()

	if len(categories) != 10 {
		t.Errorf("expected 10 default categories, got %d", len(categories))
	}

	expectedNames := map[string]bool{
		"unused_docker_images":     true,
		"unused_docker_containers": true,
		"unused_docker_volumes":    true,
		"docker_build_cache":       true,
		"unused_apfs_snapshots":    true,
		"system_caches":            true,
		"user_caches":              true,
		"tmp_files":                true,
		"xcode_derived_data":       true,
		"brew_cache":               true,
	}

	for _, c := range categories {
		if !expectedNames[c.Name] {
			t.Errorf("unexpected category name: %s", c.Name)
		}
		if c.CreatedAt.IsZero() {
			t.Errorf("category %s has zero CreatedAt", c.Name)
		}
	}

	// Verify brew_cache has verify_required = false
	for _, c := range categories {
		if c.Name == "brew_cache" && c.VerifyRequired {
			t.Error("brew_cache should have verify_required = false")
		}
		if c.Name != "brew_cache" && !c.VerifyRequired {
			t.Errorf("%s should have verify_required = true", c.Name)
		}
	}

	// Verify all start disabled
	for _, c := range categories {
		if c.Enabled {
			t.Errorf("category %s should start disabled", c.Name)
		}
	}
}
