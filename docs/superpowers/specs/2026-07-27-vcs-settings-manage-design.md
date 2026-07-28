# VCS Settings Manage (edit/delete) Design

**Status:** approved  
**Date:** 2026-07-27

## Goal

Let the owner manage Gitea connections and repository links from Settings: create, edit, and delete, with UI aligned to existing client/tag directory patterns.

## Approach

Reusable form + list rows with pencil/trash actions (same pattern as Clients). Add `PUT` update endpoints; keep existing `DELETE` (connection delete cascades to repositories via FK).

## Backend

- `PUT /api/v1/vcs/connections/{connectionID}` — update baseUrl, ownerIdentity, defaultClientId; optional token (empty/omitted keeps existing encrypted token); optional enabled.
- `PUT /api/v1/vcs/repositories/{repositoryID}` — update connectionId, owner, name, clientId, projectId; optional enabled.
- Reuse validation/normalization from create paths and host allowlist for baseUrl.
- Responses never include the raw token; only `tokenConfigured`.
- Deleting a connection cascades repositories (`ON DELETE CASCADE`).

## Frontend

- Two subsections: Connections and Repositories.
- Each item is a `client-row` with title/meta and aligned `client-row-actions` (edit + delete).
- Shared form switches between create and edit; Cancel clears selection.
- Delete uses `confirmDestructiveAction`.
- Spanish/English strings in `i18n.ts`.
- Toast on success/failure.

## Out of scope

- Live Gitea connection test button
- GitHub/GitLab providers
- New CSS design system (reuse existing Settings/client-row styles)
