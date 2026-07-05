# leotime Agent Instructions

## Project Mission

Build `leotime` as a lightweight, self-hosted time tracker and invoicing tool for one owner first. Keep it easy to deploy, easy to back up, and easy to understand for contributors coming from Python/Django.

## Working Rules

- Do not implement real client CRUD until the scaffold, repo AI guidance, operations tooling, and Solidtime import foundation are committed.
- Do not commit real Solidtime export ZIP files or personal production data.
- Keep Docker-first deployment working.
- Keep documentation updated whenever behavior, commands, schema, import mapping, or deployment expectations change.
- Prefer explicit Go and SQL over hidden framework magic.
- Prefer focused React components and stable layouts over decorative UI.
- Do not revert user changes. Work with the existing tree.

## Required Checks

Run the smallest relevant checks while working, and run the full gate before final delivery:

```bash
make test
make build-web
make smoke
```

For import work, also run:

```bash
make bench
make import-solidtime-dry ZIP=<path-to-export.zip>
```

Install git hooks once per clone with `make setup-hooks` (included in `make setup`). Every commit runs `make pre-commit`: gofmt, go vet, backend tests, frontend tests, and web build.

### Before Finishing Any Change

Before telling the user that work is done, always run:

```bash
make pre-commit
```

This is the same gate the git hook runs. It verifies:

- Go formatting with `gofmt`
- Go static analysis with `go vet`
- Backend tests with `go test ./...`
- Frontend unit tests with Vitest
- Frontend production build with TypeScript and Vite

If `make pre-commit` fails, fix the reported issues and rerun until it passes. Do not hand off work with a failing gate. Common fixes:

- `gofmt -w <file>` for Go formatting
- address `go vet` findings in the reported package
- fix failing tests or TypeScript/build errors before retrying

Only skip this step when the change is explicitly read-only (questions, reviews with no edits). After larger delivery checks, also run `make smoke` and `make deploy-check` when behavior or deployment expectations changed.

## Commit Style

Use small commits with clear messages following [Conventional Commits](https://www.conventionalcommits.org/):

- `chore:` for repo/tooling/docs setup.
- `feat:` for product capabilities.
- `fix:` for bug fixes.
- `test:` for test-only changes.
- `docs:` for documentation-only changes.

After every change set, propose an explicit commit message to the user (subject + short body when useful). Describe what capability, fix, or documentation was introduced. Do not run `git commit` unless the user asks.

Example:

```text
feat: add tasks CRUD with API, UI panel, and integration tests

Expose authenticated /api/v1/tasks endpoints, a TaskPanel in the dashboard,
store and HTTP lifecycle tests, and Spanish API docs in docs/16-tasks-api.md.
```

## Architecture Expectations

- Backend code lives under `apps/api`.
- Frontend code lives under `apps/web`.
- Product and operational decisions live under `docs`.
- Repo-scoped Codex skills live under `.agents/skills`.
- Project-scoped subagents live under `.codex/agents`.

## Security And Data

- Treat attached exports as personal data.
- Use synthetic fixtures in tests.
- Do not log passwords, session tokens, or full personal exports.
- Auth cookies must remain HTTP-only.
- Importers must validate ZIP contents before writing to the database.

