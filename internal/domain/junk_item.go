package domain

import "time"

// VerificationStatus indicates the user's decision on a junk item.
type VerificationStatus int

const (
	VerificationSkipped  VerificationStatus = -1
	VerificationPending  VerificationStatus = 0
	VerificationApproved VerificationStatus = 1
)

// JunkItem represents a single piece of system junk discovered by a scanner.
type JunkItem struct {
	ID              int64              `json:"id"`
	CategoryID      int64              `json:"category_id"`
	VolumeID        *int64             `json:"volume_id,omitempty"`
	Path            string             `json:"path"`
	Description     string             `json:"description"`
	SizeBytes       int64              `json:"size_bytes"`
	LastAccessed    *time.Time         `json:"last_accessed,omitempty"`
	ScanID          *int64             `json:"scan_id,omitempty"`
	VerifiedByUser  VerificationStatus `json:"verified_by_user"`
	CleanedAt       *time.Time         `json:"cleaned_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

// IsPendingReview returns true if the item has not been reviewed yet.
func (j JunkItem) IsPendingReview() bool {
	return j.VerifiedByUser == VerificationPending
}

// IsApproved returns true if the user approved this item for cleanup.
func (j JunkItem) IsApproved() bool {
	return j.VerifiedByUser == VerificationApproved

}

// IsCleaned returns true if the item has already been cleaned.
func (j JunkItem) IsCleaned() bool {
	return j.CleanedAt != nil
}
