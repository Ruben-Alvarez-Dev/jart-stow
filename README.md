# Jart-Stow

**macOS development hygiene & backup exclusion manager.**

Jart-Stow performs two distinct but complementary functions:

1. **Backup Exclusion** — monitors configurable workspace roots (local and external drives) via a background daemon, automatically scans projects for development artifacts, and applies exclusions to Time Machine and Carbon Copy Cloner.

2. **System Hygiene** — detects and presents system junk for review (unused Docker resources, stale APFS snapshots, caches, temporary files) and enables granular, user-verified cleanup across all connected volumes.

---

## Quick Start

```bash
brew install jart-stow

jart-stow daemon install
jart-stow scan
jart-stow status
```

---

## Documentation

Full documentation at [ruben-alvarez-dev.github.io/jart-stow](https://ruben-alvarez-dev.github.io/jart-stow)

---

## License

MIT © Ruben Alvarez
