# ADR-008: Conventional Commits with Squash Merge

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

Jart-Stow needs a commit strategy that enables automated changelog generation, semantic versioning, and clean Git history.

## Decision

All commits follow the Conventional Commits specification (`feat(scope): description`). Branches are squashed into `main` via pull requests. Each commit message includes a 2-4 sentence body.

## Consequences

**Positive:**
- Changelog can be auto-generated from commit history.
- Semantic version bumps can be determined programmatically.
- Squash merge keeps `main` history linear and clean.
- Consistent commit format across all contributions.

**Negative:**
- Requires discipline in commit message formatting.
- Pre-commit hooks add friction to the development workflow.

## Alternatives Considered

### Merge commits
Rejected. Merge commits create non-linear history that complicates changelog generation.

### Rebase and merge
Rejected. While it preserves individual commits, reviewing 10+ granular commits in a PR is harder than reviewing a single squash commit.
