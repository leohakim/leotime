# Offline Queue MVP

The web app queues write operations in IndexedDB when the browser is offline or when a network request fails. When connectivity returns, queued mutations sync to the API in order.

## Supported Offline Writes

| Operation | Queued when offline |
| --- | --- |
| Create client | Yes |
| Create project | Yes |
| Create task | Yes |
| Create tag | Yes |
| Create manual time entry | Yes |
| Update manual time entry | Yes |
| Start timer | Yes |
| Update active timer | Yes |
| Stop timer | Yes |

Reads continue to use React Query cache with `networkMode: offlineFirst`.

## Queue Storage

IndexedDB database: `leotime-offline`

| Store | Purpose |
| --- | --- |
| `mutations` | Ordered mutation queue |
| `idMap` | Maps `local_*` IDs to server IDs after sync |

Each queued mutation stores:

- Operation type.
- Optional local ID for creates.
- Optional entity ID for updates/stops.
- JSON payload.
- Retry count and last error.

## Sync Behavior

1. User performs a write while offline.
2. The app enqueues the mutation and returns an optimistic entity with a `local_*` ID.
3. React Query caches are patched locally so the UI stays usable.
4. When the browser goes online, `OfflineProvider` flushes the queue automatically.
5. Users can also click **Sync now** in the toolbar pill when pending changes exist.

Local IDs are remapped before dependent operations run, for example stopping an offline timer after its start mutation synced.

## UI

- Toolbar pill shows offline mode, pending count, syncing state, or sync failure.
- Optimistic time entries use `source: offline` or `source: timer` depending on workflow.

## Where To Read The Behavior

| Layer | Location |
| --- | --- |
| Queue + sync | `apps/web/src/lib/offline/` |
| Offline-aware mutations | `apps/web/src/lib/offline/mutations.ts` |
| Provider + status UI | `apps/web/src/lib/offline/offlineContext.tsx`, `offlineStatusUi.tsx` |
| App wiring | `apps/web/src/main.tsx`, `apps/web/src/App.tsx` |
| Strategy doc | `docs/04-offline-sync.md` |

## Checks

```bash
make test-web
make build-web
```

Manual check:

1. Load the app online and log in.
2. Open DevTools → Network → Offline.
3. Start a timer or create a manual entry.
4. Confirm the toolbar pill shows pending changes.
5. Go back online and confirm sync completes.

## Out Of Scope For This Slice

- Offline edits to already-synced server records beyond queued PATCH support.
- Conflict review UI.
- Offline invoice/report workflows.
- Background sync service worker.
