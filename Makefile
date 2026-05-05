# Jart-Stow Build System

.PHONY: build run test lint clean docs

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

# --- Python API ---

api-install:
	cd api && python3 -m venv .venv && source .venv/bin/activate && pip install -r requirements.txt

api-run:
	cd api && uvicorn app.main:app --host 0.0.0.0 --port 8420 --reload

api-test:
	cd api && pytest --cov=app --cov-report=xml

# --- Docs ---

docs-serve:
	mkdocs serve

docs-build:
	mkdocs build

docs-deploy:
	mkdocs gh-deploy --force

# --- CI helpers ---

ci: lint test api-test build
