# ADR-007: MkDocs + Material for MkDocs for Documentation

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

Jart-Stow requires professional, version-controlled documentation published alongside the codebase. The documentation must be open source, support Markdown, and generate a searchable static site deployable to GitHub Pages.

## Decision

Use MkDocs with the Material for MkDocs theme. Documentation lives in `docs/` as Markdown files. Deploy via `mkdocs gh-deploy` to GitHub Pages.

## Consequences

**Positive:**
- MIT licensed, fully open source.
- Material theme is used by FastAPI, Pydantic, Kubernetes, and Mozilla.
- Markdown-native — developers write docs without learning a new syntax.
- Automatic dark mode, search, code highlighting, and navigation.
- CI/CD deployment via GitHub Actions.

**Negative:**
- Requires Python (already a dependency for the API).
- Custom extensions may require Material Insiders (paid), but the free tier is sufficient.

## Alternatives Considered

### Docusaurus
Rejected. React-based, overengineered for a documentation site. Adds a Node.js dependency to a Go + Python project.

### VitePress
Rejected. Vue-based, same dependency concern.

### Docsify
Rejected. Less polished output and weaker search capabilities than Material for MkDocs.
