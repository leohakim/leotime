# ADR 0003: S3 Daily Backups, Encrypted Credentials, and In-App Restore

## Status

Accepted (planned — not yet implemented).

## Context

leotime stores all product data in a single SQLite file with WAL mode. VPS deployment is Docker-first and single-container. The deploy guide previously recommended external tools (restic, rclone) for offsite copies.

The owner needs:

- Configurable S3 credentials from the UI (AWS or S3-compatible endpoint).
- Automatic daily backup to a private bucket on the public internet.
- Manual backup, restore, and status from the same delivery.
- Both in-process scheduler and CLI entry points.

SQLite WAL mode makes raw file copies unsafe while the app is running. Backups must use the SQLite backup API.

## Decision

### Backup engine

- Package `internal/backup` with snapshot, gzip, S3 upload, retention, and restore.
- Snapshot via SQLite `backup` API into a temp file, then gzip.
- Upload with AWS SDK for Go v2 (`configurable endpoint`, static credentials, optional path-style).
- Object key format: `{prefix}leotime-{UTC-timestamp}.db.gz`.
- Retention default: **365 days**; delete older objects after a successful upload.

### Credentials

- Store S3 settings in `backup_settings` (one row, single owner).
- Encrypt `secret_access_key` with AES-256-GCM; key material from `LEOTIME_SECRETS_KEY` (32 bytes, env).
- Never return decrypted secrets in API responses.

### Scheduling

- Extend the existing in-process scheduler (`internal/scheduler`) with a backup tick (every 1 minute).
- Run when `enabled=true`, current local hour ≥ `schedule_hour` (owner timezone), and no successful run yet today.
- Default schedule hour: **01:00** local time.
- Mutex prevents overlapping backup and restore operations.

### Restore

- Same delivery includes restore from S3 (not deferred).
- Download → decompress → validate → local safety snapshot → SQLite backup API into live DB.
- Exposed via `POST /api/v1/backups/restore`, Settings UI, and `leotime backup restore`.
- Requires explicit confirmation (`confirm: true` or CLI `--force`).

### CLI

Subcommands on the same binary:

```text
leotime backup run [--force]
leotime backup list
leotime backup restore --object-key <key> | --latest [--force]
```

CLI and scheduler share the same runner code path.

## Consequences

Good:

- No extra containers; fits the single-process model from ADR 0002.
- Owner can configure, test, back up, and restore without SSH.
- Consistent snapshots without stopping the container.
- CLI supports automation and disaster recovery.

Tradeoffs:

- Restore while serving HTTP holds a write lock briefly; concurrent requests may wait.
- Losing `LEOTIME_SECRETS_KEY` makes stored credentials unreadable (S3 access must be reconfigured).
- Large databases increase backup duration and S3 storage cost (365-day retention).
- S3 list/restore in UI depends on bucket list permissions.

## Alternatives considered

| Alternative | Why not chosen |
| --- | --- |
| Env-only S3 config | Owner asked for UI configuration |
| Raw `leotime.db` copy | Unsafe with WAL |
| External cron + rclone only | Higher VPS operator friction; no in-app status/restore |
| Restore in a later slice | Owner requested same delivery |
| Separate backup worker container | Conflicts with single-container goal |

## Follow-up

- Optional email alert when daily backup fails (reuse email outbox).
- Optional server-side encryption (SSE-S3/SSE-KMS) flags per provider.
