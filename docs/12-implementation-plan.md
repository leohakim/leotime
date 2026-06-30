# Implementation Plan

This is the living implementation plan for `leotime`. It turns the product vision into small, reviewable delivery slices.

## Current Rule

Every product slice should include:

- Backend store/service code.
- Authenticated HTTP API.
- Frontend API client.
- UI with validation and empty/loading/error states.
- Unit or integration tests for risky behavior.
- Documentation update.
- A focused commit.

## Completed Foundation

- Monorepo scaffold with Go API, React/Vite web app, SQLite, Docker, and docs.
- Repo-scoped Codex guidance, skills, and agents.
- Docker-first operational tooling through `Makefile`.
- Prometheus metrics endpoint and observability profile.
- Solidtime ZIP import foundation with parser, importer, compatibility tables, and tests.
- Client CRUD with backend API, frontend workbench, validation, tests, and docs.
- Project CRUD with optional client assignment, colors, optional rate override, tests, and docs.

## MVP Delivery Slices

1. **Tasks CRUD**
   - Depends on projects.
   - Supports optional project assignment according to user setting.
   - Includes billable default and archive behavior.

2. **Tags CRUD**
   - Independent taxonomy used by time entries and reports.
   - Must keep unique names per user.

3. **Manual Time Entries**
   - Depends on clients, projects, tasks, and tags.
   - Supports one-minute precision, billable flag, overlap warnings, and duration calculation.

4. **Timer Workflow**
   - Start, stop, and resume daily tracking.
   - Allows overlapping activities and flags overlaps for reporting.

5. **Timesheet View**
   - Weekly editable view grouped by day, client, project, task, and tags.

6. **Calendar View**
   - Calendar-style inspection and editing of time entries.

7. **Reports And Exports**
   - CSV and JSON exports.
   - Hide start/end timestamps when the report is configured to show only totals.

8. **Invoices**
   - Draft invoices from billable time.
   - Multi-currency, frozen client fields, line items, tax/withholding fields, and PDF/export.

9. **Offline Queue MVP**
   - Offline creation/edit queue for the core daily workflow.
   - Syncs when connectivity returns.

## Quality Gates By Slice

Small backend-only slice:

```bash
make test-api
```

Frontend slice:

```bash
make test-web
npm --workspace @leotime/web run build
```

Full product slice:

```bash
make test
make test-e2e
make docker-build
make smoke
```

When port `8080` is already occupied by a local process without static assets, run smoke against a temporary app instance with `BASE_URL`.

## Current Next Task

The next implementation slice is **Tasks CRUD**.
