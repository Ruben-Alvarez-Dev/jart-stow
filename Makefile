# Jart-Stow Build System

.PHONY: build run test lint clean docs tui tui-dev

# --- Go ---

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOLINT=golangci-lint
BINARY=jart-stow

build:
	$(GOBUILD) -o $(BINARY) ./cmd/$(BINARY)

run: build
	./$(BINARY)

test:
	$(GOTEST) -race -coverprofile=coverage.out ./...

lint:
	$(GOLINT) run ./...

clean:
	rm -f $(BINARY)
	rm -f coverage.out

# --- TUI ---

tui: build
	./$(BINARY) tui

tui-dev: build
	./$(BINARY) tui 2>tui-debug.log

# --- Docs ---

docs-serve:
	mkdocs serve

docs: docs-build

docs-build:
	mkdocs build

docs-deploy:
	mkdocs gh-deploy --force

# --- CI helpers ---

ci: lint test build
