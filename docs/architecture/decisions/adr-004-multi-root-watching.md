# ADR-004: Multi-Root Configurable Watching

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

The initial design hardcoded `/Users/<username>/Code` as the single watched directory. The user identified that external drives may also contain development projects that should be included in backup exclusions.

## Decision

Replace the hardcoded root with a `watch_roots` database table. Users can add, enable, disable, and remove watch roots via CLI, TUI, or API. Each root is independently watched by the FSEvents daemon.

## Consequences

**Positive:**
- External SSDs, network mounts, and secondary volumes are supported.
- Roots can be disabled without deletion when drives are disconnected.
- Volume UUID tracking prevents stale paths on reconnected drives.

**Negative:**
- Multiple FSEvents streams increase resource usage slightly.
- Disconnected external volumes require graceful error handling.

## Alternatives Considered

### Single root with symlinks
Rejected. Symlinks are fragile, can break silently, and don't work across volumes reliably.

### Separate daemon per root
Rejected as overengineered. A single daemon can manage multiple FSEvents streams efficiently.
