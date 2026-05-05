package sqlite

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleRepo_CRUD(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewRuleRepo(conn.DB())

	// Save global rule
	rule := &domain.Rule{
		ProjectID:    nil,
		Pattern:      "node_modules",
		MaxSizeBytes: 524288000,
		Action:       domain.RuleActionExclude,
		Priority:     10,
		Enabled:      true,
	}
	saved, err := repo.Save(ctx, rule)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.True(t, saved.IsGlobal())

	// Save project-specific rule
	pid := int64(1)
	projRule := &domain.Rule{
		ProjectID:    &pid,
		Pattern:      ".venv",
		MaxSizeBytes: 1073741824,
		Action:       domain.RuleActionWarn,
		Priority:     5,
		Enabled:      true,
	}
	projSaved, err := repo.Save(ctx, projRule)
	require.NoError(t, err)
	assert.False(t, projSaved.IsGlobal())

	// FindAll
	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// FindByID
	found, err := repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, "node_modules", found.Pattern)

	// FindByProjectID
	byProject, err := repo.FindByProjectID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, byProject, 1)
	assert.Equal(t, ".venv", byProject[0].Pattern)

	// FindGlobalRules
	globals, err := repo.FindGlobalRules(ctx)
	require.NoError(t, err)
	assert.Len(t, globals, 1)
	assert.Equal(t, "node_modules", globals[0].Pattern)

	// Update
	saved.Pattern = "node_modules_updated"
	saved.MaxSizeBytes = 999
	updated, err := repo.Update(ctx, saved)
	require.NoError(t, err)
	assert.Equal(t, "node_modules_updated", updated.Pattern)
	assert.Equal(t, int64(999), updated.MaxSizeBytes)

	// Delete
	err = repo.Delete(ctx, saved.ID)
	require.NoError(t, err)
	_, err = repo.FindByID(ctx, saved.ID)
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)
}

func TestRuleRepo_NotFound(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewRuleRepo(conn.DB())

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)

	err = repo.Delete(ctx, 99999)
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)

	_, err = repo.Update(ctx, &domain.Rule{ID: 99999, Pattern: "x", MaxSizeBytes: 1, Action: domain.RuleActionWarn, Enabled: true})
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)
}

func TestRuleRepo_Disabled(t *testing.T) {
	conn := setupTestDB(t)
	ctx := freshContext()
	repo := NewRuleRepo(conn.DB())

	rule := &domain.Rule{
		Pattern: "disabled-rule", MaxSizeBytes: 100,
		Action: domain.RuleActionAlert, Priority: 1, Enabled: false,
	}
	saved, err := repo.Save(ctx, rule)
	require.NoError(t, err)
	assert.False(t, saved.Enabled)
}
