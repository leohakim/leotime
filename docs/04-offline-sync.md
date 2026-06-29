# Offline And Sync Strategy

The MVP should support offline work without pretending to be a full local-first database on day one.

## MVP Offline Behavior

When the browser is offline, the frontend should allow:

- Creating clients.
- Creating projects.
- Creating tasks.
- Creating tags.
- Creating time entries.
- Starting and stopping a timer.
- Editing unsynced entries.

These changes are stored in IndexedDB as local records plus queued mutations.

## Sync Queue

Each offline mutation should have:

- Local ID.
- Operation type.
- Payload.
- Created timestamp.
- Retry count.
- Last error.

When the browser is online again, the app sends queued mutations to the API in order.

## Conflict Policy

The first conflict policy should be boring and visible:

- If a server record did not change, apply the client mutation.
- If both changed, keep the server record and create a conflict item for the user to review.
- Time entries are append-friendly, so conflicts should be rare.

## Why Not Full Local-First In MVP

Full local-first sync means conflict resolution, schema migrations on client databases, device identity, background retry behavior, and good debugging tools. That is valuable, but it is too much for the first usable version.

The MVP approach gives the daily benefit first: the timer and manual entry forms keep working without connection.

## Backlog Options

Later we can evaluate:

- ElectricSQL.
- Replicache-style mutation logs.
- A custom sync endpoint around SQLite server state.
- Tauri with a local SQLite database for desktop-first usage.

