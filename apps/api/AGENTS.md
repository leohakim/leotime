# API Agent Instructions

## Scope

This directory owns the Go backend, SQLite migrations, CLI commands, metrics, import services, HTTP API, and backend tests.

## Go Rules

- Run `gofmt` after editing Go files.
- Keep handlers thin and put persistence/business logic in `internal/store` or a dedicated domain package.
- Use integration tests with temporary SQLite files for database behavior.
- Keep migrations forward-only once committed.
- Keep CLI commands deterministic and scriptable.

## Import Rules

- Solidtime import code must support dry-run mode.
- Never require the real attached export in tests.
- Validate expected CSV headers and fail clearly for unknown or missing required files.
- Keep external ID mapping idempotent through `external_mappings`.

## Checks

```bash
go test ./...
go test -bench=. ./...
```

