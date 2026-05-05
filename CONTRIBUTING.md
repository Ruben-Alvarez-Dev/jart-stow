# Contributing to Jart-Stow

## Development Principles

- **No mock data.** Every piece of data must come from real system operations.
- **SOLID + DRY.** Every module has a single responsibility. No duplication.
- **Hexagonal architecture.** Domain logic isolated from infrastructure.
- **Conventional commits.** `feat(scope): description` format strictly followed.
- **English only.** Code, comments, commits, docs — all in English.

## Getting Started

```bash
git clone https://github.com/Ruben-Alvarez-Dev/jart-stow.git
cd jart-stow

# Go daemon + TUI + CLI
go mod download
go build -o jart-stow ./cmd/jart-stow

# Python API
cd api
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

## Commit Guidelines

- **Granular:** one logical change per commit.
- **Organic:** 2-4 sentence body explaining what and why.
- **Push after commit:** `git push origin <branch>`.

```
feat(daemon): implement multi-root FSEvents watcher

The daemon now watches all configured watch_roots from the
database instead of a single hardcoded path. Each root is
independently enabled/disabled via CLI or API.
```

## Pull Requests

1. Branch from `main`: `feat/description`, `fix/description`, `chore/description`
2. Keep branches alive ≤ 2 days
3. All CI checks must pass
4. Squash merge to `main`
