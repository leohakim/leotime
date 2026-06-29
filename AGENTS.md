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

## Commit Style

Use small commits with clear messages:

- `chore:` for repo/tooling/docs setup.
- `feat:` for product capabilities.
- `fix:` for bug fixes.
- `test:` for test-only changes.
- `docs:` for documentation-only changes.

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

