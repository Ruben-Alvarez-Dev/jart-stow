# Jart-Stow Development Workflow Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Git Strategy

### Branch Model: Trunk-Based Development

```
main ──────────────────────────────────────────────
       \               \                \
        feat/xxx        fix/yyy          chore/zzz
```

- **`main`** is the single source of truth. Always deployable.
- **Feature branches** are short-lived (`feat/*`, `fix/*`, `chore/*`, `docs/*`).
- **No `develop` branch.** Merge directly to `main` via squash or rebase.
- Branches live **maximum 2 days** before merging.

---

## 2. Conventional Commits

Every commit follows the [Conventional Commits](https://www.conventionalcommits.org/) specification strictly.

### Format

```
<type>(<scope>): <description>

[optional body: 2-4 sentences explaining what and why]

[optional footer: BREAKING CHANGE, Closes #issue]
```

### Types

| Type | Use when |
|---|---|
| `feat` | New feature or significant functionality |
| `fix` | Bug fix |
| `refactor` | Code restructuring without behavior change |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `docs` | Documentation changes |
| `chore` | Maintenance, dependencies, tooling |
| `ci` | CI/CD pipeline changes |
| `style` | Formatting, linting (no logic change) |

### Scopes

| Scope | Applies to |
|---|---|
| `daemon` | Go daemon (FSEvents, launchd, scanning) |
| `tui` | Bubble Tea terminal interface |
| `cli` | Cobra CLI commands |
| `api` | FastAPI REST API |
| `engine` | Domain logic, services, ports |
| `adapters` | SQLite, tmutil, CCC, FSEvents adapters |
| `db` | Database schema, migrations |
| `docs` | MkDocs documentation |
| `specs` | Design specifications |
| `build` | Build system, Makefile, scripts |
| `ci` | GitHub Actions workflows |

### Examples

```
feat(daemon): implement FSEvents watcher for /Code directory

The daemon now watches the designated workspace root using macOS
native FSEvents API via the fsnotify Go package. New project
directories are detected with a 2-second debounce to avoid
false positives during filesystem operations.

Closes #12
```

```
fix(tui): correct scanner gauge overflow on large projects

When scanning projects with more than 999 folders, the progress
gauge would overflow due to an integer division bug. Fixed by
using float64 arithmetic and clamping to 0-100 range.
```

```
refactor(engine): extract ScanService from monolithic scanner

The scan logic was embedded in the daemon package directly.
Extracted into a dedicated ScanService following hexagonal
architecture, making it testable and reusable by both daemon
and CLI.
```

---

## 3. Commit Automation

### Rules

- Commits are **granular**: one logical change per commit.
- Commits are **organic**: written by the developer, not auto-generated.
- Each commit message is **2-4 sentences** in the body explaining **what** changed and **why**.
- After each commit: `git push origin <branch>`.

### Pre-Commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    hooks:
      - id: golangci-lint
  - repo: https://github.com/astral-sh/ruff-pre-commit
    hooks:
      - id: ruff        # Python linter
      - id: ruff-format # Python formatter
  - repo: https://github.com/pre-commit/pre-commit-hooks
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
```

---

## 4. GitHub Repository Setup

### Repository

```
github.com/Ruben-Alvarez-Dev/jart-stow
```

### Branch Protection Rules (main)

| Rule | Value |
|---|---|
| Require pull request before merging | ✅ |
| Require approvals | 0 (solo dev) |
| Require status checks to pass | ✅ (CI) |
| Require conversation resolution | ✅ |
| Require linear history | ✅ (squash merging) |

### Merge Strategy

**Squash and merge** for all PRs. Squash commit message follows conventional commits format.

---

## 5. CI/CD Pipeline

### CI: `ci.yml` (on every push and PR)

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  go-lint:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v6

  go-test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out

  python-test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install -r api/requirements.txt
      - run: cd api && pytest --cov=app --cov-report=xml

  build:
    runs-on: macos-latest
    needs: [go-test]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go build -o jart-stow ./cmd/jart-stow
      - uses: actions/upload-artifact@v4
        with:
          name: jart-stow-macos
          path: jart-stow
```

### Release: `release.yml` (on tag push)

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: macos-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go build -o jart-stow ./cmd/jart-stow
      - run: |
          tar -czf jart-stow-darwin-arm64.tar.gz jart-stow
          shasum -a 256 jart-stow-darwin-arm64.tar.gz > checksums.txt
      - uses: softprops/action-gh-release@v1
        with:
          files: |
            jart-stow-darwin-arm64.tar.gz
            checksums.txt
          generate_release_notes: true
```

---

## 6. Versioning

[Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

| Bump | When |
|---|---|
| MAJOR | Breaking API changes, database schema incompatible changes |
| MINOR | New features, new endpoints, new daemon capabilities |
| PATCH | Bug fixes, performance improvements, docs |

### Changelog

`CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
# Changelog

## [1.0.0] - 2026-06-01

### Added
- FSEvents daemon for automatic /Code monitoring
- Bubble Tea TUI with 6 screens
- FastAPI 3.0 REST API with OpenAPI docs
- Time Machine and CCC backup exclusion
- Hygiene rule system per project
- SQLite persistence with WAL mode
- launchd integration for auto-start

### Changed
- Renamed from EXCLUSION-SCRIPT to jart-stow
- Migrated to hexagonal architecture
```

---

## 7. Documentation Pipeline

```
Source (docs/*.md) → MkDocs build → GitHub Pages
```

### mkdocs.yml

```yaml
site_name: Jart-Stow
site_description: macOS development hygiene & backup exclusion manager
site_author: Ruben Alvarez
repo_url: https://github.com/Ruben-Alvarez-Dev/jart-stow
theme:
  name: material
  palette:
    scheme: slate
    primary: cyan
    accent: pink
  features:
    - navigation.tracking
    - navigation.tabs
    - search.suggest
    - content.code.copy

nav:
  - Home: index.md
  - Architecture:
      - Overview: architecture/overview.md
      - Decisions: architecture/decisions/
  - API Reference: api/reference.md
  - TUI Guide: tui/user-guide.md
  - Daemon: daemon/lifecycle.md
  - Development:
      - Contributing: development/contributing.md
      - Setup: development/setup.md

markdown_extensions:
  - pymdownx.highlight
  - pymdownx.superfences
  - admonition
  - footnotes
```

### Deployment

```yaml
# .github/workflows/docs.yml
name: Deploy Docs
on:
  push:
    branches: [main]
    paths: ['docs/**', 'mkdocs.yml']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install mkdocs-material
      - run: mkdocs gh-deploy --force
```

---

## 8. ADR Template

Architecture Decision Records live in `docs/architecture/decisions/`:

```markdown
# ADR-NNN: Title

**Status:** Proposed | Accepted | Deprecated | Superseded
**Date:** YYYY-MM-DD
**Deciders:** Ruben Alvarez
**Supersedes:** ADR-NNN (if applicable)

## Context

What is the issue that motivates this decision?

## Decision

What is the change we are proposing?

## Consequences

What becomes easier or harder because of this decision?

## Alternatives Considered

What other options were evaluated and why were they rejected?
```

---

## 9. Issue & PR Templates

### Issue Template (`.github/ISSUE_TEMPLATE/bug_report.md`)

```markdown
---
name: Bug Report
about: Report a defect in Jart-Stow
title: 'fix(scope): '
labels: bug
---

## Description

## Steps to Reproduce

1.
2.
3.

## Expected Behavior

## Actual Behavior

## Environment

- macOS version:
- jart-stow version:
- Daemon running? yes/no
```

### PR Template (`.github/pull_request_template.md`)

```markdown
## Description

<!-- 2-4 sentences describing what this PR does and why -->

## Type of Change

- [ ] feat: New feature
- [ ] fix: Bug fix
- [ ] refactor: Code restructuring
- [ ] docs: Documentation
- [ ] chore: Maintenance

## Checklist

- [ ] Tests pass locally (`go test ./...` / `pytest`)
- [ ] Linting passes (`golangci-lint run` / `ruff check`)
- [ ] Documentation updated (if applicable)
- [ ] CHANGELOG entry added (if user-facing)
- [ ] No mock or fake data introduced
```

---

## 10. Development Environment Setup

```bash
# Clone
git clone https://github.com/Ruben-Alvarez-Dev/jart-stow.git
cd jart-stow

# Go dependencies
go mod download

# Python API dependencies
cd api
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# Pre-commit hooks
pre-commit install

# Build
go build -o jart-stow ./cmd/jart-stow

# Run tests
go test -race ./...
cd api && pytest

# Start API
./jart-stow api start

# Install daemon
./jart-stow daemon install
```

---

## 11. Code Quality Gates

| Gate | Tool | Threshold |
|---|---|---|
| Go lint | golangci-lint | 0 issues |
| Go format | gofmt | Must pass |
| Go vet | go vet | 0 issues |
| Python lint | ruff | 0 issues |
| Python format | ruff format | Must pass |
| Test coverage (Go) | go test -cover | ≥ 80% |
| Test coverage (Python) | pytest-cov | ≥ 80% |
