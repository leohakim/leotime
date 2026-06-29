# Testing Strategy

`leotime` should be test-heavy from the beginning. The test suite is part of the product, not a cleanup task.

## Backend Tests

Backend tests should cover:

- Config parsing.
- Password hashing and verification.
- Migration runner.
- Store methods.
- HTTP handlers.
- Auth cookie behavior.
- Error responses.

Run:

```bash
cd apps/api
go test ./...
```

## Database Tests

Database tests use temporary SQLite files instead of mocks when persistence behavior matters.

This catches:

- Migration mistakes.
- Constraint mistakes.
- Foreign-key issues.
- Query behavior that mocks would hide.

## Frontend Unit Tests

Frontend tests should cover:

- Rendering critical screens.
- Layout mode switching.
- Language switching.
- Offline queue behavior.
- Form validation.
- API client error handling.

Run:

```bash
cd apps/web
npm test -- --run
```

## End-To-End Tests

Playwright should cover the real user flows:

- Login.
- Start timer.
- Stop timer.
- Create manual entry.
- Edit weekly timesheet.
- Create invoice draft.
- Export report.

Run:

```bash
cd apps/web
npm run test:e2e
```

## Test Pyramid

Most tests should be fast unit and integration tests. E2E tests should cover confidence-critical flows rather than every edge case.

```text
many unit tests
some integration/API/database tests
few end-to-end tests
```

## CI Expectation

The future GitHub Actions workflow should run:

- `go test ./...`
- `npm ci`
- `npm test -- --run`
- `npm run build`
- Playwright smoke test
- Docker build

