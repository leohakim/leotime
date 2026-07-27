# VCS repository project link (single project)

**Status:** approved  
**Date:** 2026-07-27

## Goal

Allow each VCS repository link to optionally point at one leotime project belonging to its effective client. Multiple repositories may share the same project (e.g. `ENACT/backend` and `ENACT/frontend` → project ENACT).

## Behavior

- Settings repository form adds a Project select filtered by effective client (explicit repo client, else connection default client).
- Empty project = client-scoped link only.
- Changing client or connection clears a project that no longer belongs to the new effective client.
- List rows show project name when set.
- No schema change; API already accepts `projectId`.

## Out of scope

- Many-to-many repo↔project links.
