# Go Conventions

## Style

- `gofmt` on every save (enforced by pre-commit).
- `golangci-lint` with default configuration.
- Maximum line length: 120 characters (soft guideline).

## Naming

- **Packages:** lowercase, single word, no underscores. `ports`, `services`, `adapters`.
- **Exported functions:** PascalCase. `NewScanService`, `FindArtifacts`.
- **Unexported functions:** camelCase. `debounceWait`, `parseEvent`.
- **Interfaces:** suffix `-er`. `BackupProvider`, `FileSystemWatcher`.
- **Constants:** PascalCase. `DefaultMaxDepth`, `CategoryDocker`.
- **Errors:** prefix `Err`. `ErrProjectNotFound`, `ErrExclusionAlreadyExists`.

## Project Structure

```
internal/
├── domain/      # No imports from other internal packages
├── ports/       # Only imports domain
├── services/    # Imports domain + ports
├── cli/         # Imports services
├── tui/         # Imports services
└── adapters/    # Imports ports + domain
```

Layer dependency rule: outer layers import inner layers. Never the reverse.

## Error Handling

- Always check errors. Never use `_` for error returns.
- Wrap errors with context: `fmt.Errorf("scanning %s: %w", path, err)`.
- Domain errors are defined in `internal/domain/errors.go`.

## Testing

- Test files alongside source: `scanner.go` → `scanner_test.go`.
- Use table-driven tests for multiple inputs.
- Use `testify` for assertions.
- Integration tests use a temporary SQLite database.
- Never mock the database in unit tests — use a real in-memory SQLite.

## Dependency Injection

Services receive their dependencies via constructor injection:

```go
func NewScanService(
    projectRepo ports.ProjectRepository,
    exclusionRepo ports.ExclusionRepository,
    backup ports.BackupProvider,
) *ScanService {
    return &ScanService{
        projectRepo:   projectRepo,
        exclusionRepo: exclusionRepo,
        backup:        backup,
    }
}
```

No global state. No package-level variables that hold mutable state.
