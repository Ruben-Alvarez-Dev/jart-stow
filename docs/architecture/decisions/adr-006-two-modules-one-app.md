# ADR-006: Two Modules in a Single Application

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

Jart-Stow has two distinct functions: Backup Exclusion (excluding dev artifacts from TM and CCC) and System Hygiene (detecting and cleaning system junk). We considered spinning them into separate tools.

## Decision

Keep both functions in a single application — `jart-stow` — as two complementary modules sharing the same daemon, database, TUI, and API.

## Consequences

**Positive:**
- Single installation, single daemon, single database.
- Unified TUI navigation (7 screens covering both functions).
- The daemon handles both project watching and periodic junk scanning.
- Consistent user experience across both functions.

**Negative:**
- Increased scope for the MVP.
- CLI command namespace is larger (`jart-stow scan` vs `jart-stow junk scan`).

## Alternatives Considered

### Separate tools: `jart-stow` + `jart-clean`
Rejected. Would require two daemons, two databases, two TUIs, two APIs. Duplication of infrastructure for functions that naturally complement each other.

### Jart-Stow for backup + system-native tools for junk
Rejected. The value of Jart-Stow is presenting both concerns in a unified, keyboard-driven interface rather than scattering them across Docker CLI, tmutil, and manual cache inspection.
