# Release Checklist

Use this checklist for every Jart-Stow release.

## Pre-Release

- [ ] All CI checks passing on `main`
- [ ] `go test -race ./...` passes locally
- [ ] `cd api && pytest` passes locally
- [ ] `golangci-lint run` passes with zero issues
- [ ] `ruff check api/` passes with zero issues
- [ ] Documentation builds: `mkdocs build`
- [ ] `CHANGELOG.md` updated with this release's changes
- [ ] Version bumped in `internal/version/version.go`
- [ ] Version bumped in `api/app/core/config.py`
- [ ] No open issues tagged for this milestone

## Release

- [ ] Tag pushed: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`
- [ ] GitHub Actions release workflow completes
- [ ] Binary attached to GitHub Release
- [ ] Checksums published
- [ ] Documentation deployed to GitHub Pages

## Post-Release

- [ ] `brew bump-formula-pr` for Homebrew tap (if applicable)
- [ ] Verify `brew install jart-stow` works
- [ ] Announce in relevant channels
