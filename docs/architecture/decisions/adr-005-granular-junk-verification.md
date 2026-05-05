# ADR-005: Granular User Verification for Junk Cleanup

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

The system hygiene module detects system junk — Docker resources, APFS snapshots, caches, temporary files. The user explicitly required that cleanup never happen automatically and that each item type must be individually verified before deletion.

## Decision

All junk categories default to `verify_required = 1`. Items discovered by scanners are written to the `junk_items` table with `verified_by_user = 0` (pending). The user must explicitly approve items (one-by-one or in batch) via the TUI or API before any cleanup action is taken.

## Consequences

**Positive:**
- Zero risk of accidental data loss.
- User retains full control over what is cleaned.
- Audit trail via `cleanup_jobs` and `junk_items.cleaned_at`.

**Negative:**
- User interaction is required before cleanup.
- Large junk sets may require significant review time.

## Alternatives Considered

### Auto-clean with allowlist
Rejected. Even with an allowlist, the risk of cleaning something the user values (e.g., old Docker images used as build references) is unacceptable.

### Auto-clean with undo
Rejected. Docker images and APFS snapshots cannot be easily restored after deletion.
