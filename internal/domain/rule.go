package domain

import "time"

// RuleAction defines the action taken when a rule matches.
type RuleAction string

const (
	RuleActionWarn    RuleAction = "warn"
	RuleActionAlert   RuleAction = "alert"
	RuleActionExclude RuleAction = "exclude"
	RuleActionClean   RuleAction = "clean"
)

// Rule represents a user-defined hygiene rule for project directories.
// When ProjectID is nil, the rule applies globally to all projects.
type Rule struct {
	ID            int64      `json:"id"`
	ProjectID     *int64     `json:"project_id,omitempty"`
	Pattern       string     `json:"pattern"`
	MaxSizeBytes  int64      `json:"max_size_bytes"`
	Action        RuleAction `json:"action"`
	Priority      int        `json:"priority"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// IsGlobal returns true if the rule applies to all projects.
func (r Rule) IsGlobal() bool {
	return r.ProjectID == nil
}
