# VCS connection and context test buttons

**Status:** approved  
**Date:** 2026-07-27

## Goal

Let the owner verify Gitea auth and that linked repositories can supply normalized daily-summary context from Settings.

## API

- `POST /api/v1/vcs/connections/{id}/test` — lightweight authenticated ping.
- `POST /api/v1/vcs/connections/{id}/preview-context?date=YYYY-MM-DD` — collect context for all repos on the connection (date defaults to today UTC).
- `POST /api/v1/vcs/repositories/{id}/preview-context?date=YYYY-MM-DD` — collect for one repo.

Responses never include tokens or raw provider payloads. Errors map to unauthorized / rate_limited / unavailable / invalid.

## UI

- Connection rows: Test connection + Test context (all repos).
- Repository rows: Test context.
- Toast + compact result summary under the relevant section.

## Out of scope

- Date picker UI (query param only).
- Testing unsaved draft credentials.
