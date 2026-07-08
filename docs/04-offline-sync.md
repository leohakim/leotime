# Offline And Sync Strategy

The MVP supports offline work without pretending to be a full local-first database.

**MVP status:** Implemented with documented limitations in [Known gaps and audit](34-known-gaps-and-audit.md) (offline FK remap, logout hygiene, queue stall-on-error, update/delete not queued).

## MVP Offline Behavior

When the browser is offline (or the network probe fails), the frontend allows **creating**:

- Clients, projects, tasks, tags
- Time entries and timers (start/stop/update while queued)

These changes are stored in IndexedDB as queued mutations plus local→server ID mappings.

## Sync Queue

Each offline mutation has:

- Local ID
- Operation type
- Payload
- Created timestamp
- Retry count
- Last error

When online, `flushOfflineQueue` sends mutations **in order**. Foreign keys are remapped from local IDs to server IDs before API calls (clients, projects, tasks, tags, time entries, timers).

## Logout Hygiene

On logout, the app clears React Query cache, queued mutations, and ID mappings so the next login does not leak or replay another user's pending work.

## Conflict Policy

The first conflict policy is boring and visible:

- If a server record did not change, apply the client mutation.
- If both changed, keep the server record and surface a conflict for review (future).
- Time entries are append-friendly; conflicts should be rare.

## Current Limitations (not full local-first)

| Limitation | Detail |
| --- | --- |
| Update/archive/delete offline | CRUD panels call the API directly; only creates (and timer/entry updates) use the queue |
| Queue stall | First failed mutation blocks later items until resolved |
| Manual entry directory | Uses week-scoped fetch; not a global offline index |
| Network detection | `navigator.onLine` + `TypeError`; HTTP 503 may not queue |

See [Known gaps and audit](34-known-gaps-and-audit.md) for fixes and priority.

## Why Not Full Local-First In MVP

Full local-first sync needs conflict resolution, client schema migrations, device identity, and debugging tools. The MVP delivers daily value: timer and manual entry keep working without connection.

## Backlog Options

Later we can evaluate:

- ElectricSQL or PowerSync
- Per-user offline DB namespacing
- Queue skip/retry UI
- Offline update/delete operations
