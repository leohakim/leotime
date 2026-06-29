# Contributing To leotime

This project is intentionally written for clarity. Prefer boring, explicit code over clever abstractions.

## Before Changing Code

Read:

- `README.md`
- `docs/01-product-vision.md`
- `docs/02-architecture-go.md`
- `docs/05-testing-strategy.md`

## Development Rules

- Keep backend behavior covered by Go tests.
- Keep frontend behavior covered by Vitest or Playwright.
- Add migrations for schema changes.
- Document architecture decisions in `docs/adr` when a choice changes the shape of the project.
- Keep Docker deployment working.

## Commands

```bash
make test-api
make test-web
npm --workspace @leotime/web run test:e2e
docker compose build
```

## Code Style

Go:

- Use `gofmt`.
- Keep handlers thin.
- Put persistence logic in `internal/store`.
- Prefer integration tests with temporary SQLite databases when testing database behavior.

Frontend:

- Keep screens usable without explanatory marketing text.
- Prefer explicit components.
- Keep layout modes working on mobile and desktop.
- Use icons for compact controls.

