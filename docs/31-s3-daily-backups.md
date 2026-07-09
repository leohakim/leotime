# S3 Daily Backups

leotime can back up the SQLite database to a private S3 bucket once per day. The same delivery includes manual backup, restore from S3, a CLI, and a Settings UI panel.

Supports **AWS S3** and **S3-compatible** providers (MinIO, Backblaze B2, Wasabi, Hetzner Object Storage, etc.) through a configurable endpoint.

## Architecture

```text
main.go
  ├── HTTP server
  │     └── /api/v1/backups/*  (settings, test, run, list, restore)
  └── scheduler (background goroutine)
        ├── email scan tick      → still-running timer outbox
        ├── outbox tick          → SMTP/log delivery
        └── backup tick (1m)     → daily S3 backup when due

CLI (same binary):
  leotime backup run [--force]
  leotime backup list
  leotime backup restore --object-key <key> | --latest [--force]
```

Backup flow:

1. Create a consistent SQLite snapshot through the SQLite backup API (safe with WAL mode).
2. Build a `manifest.json` with SHA-256 hashes for the database and every file under `LEOTIME_DOCUMENT_ROOT`.
3. Package `manifest.json`, `leotime.db`, and `documents/...` into a gzip tar archive (`.tar.gz`).
4. Upload to `{prefix}leotime-{UTC-timestamp}.tar.gz`.
5. Delete objects older than the configured retention window.
6. Persist last run status in `backup_settings`.

Legacy `.db.gz` objects (database only) remain listable and restorable for older backups.

Restore flow:

1. Download the selected object from S3.
2. If the object is `.tar.gz`, extract it to a temp directory and validate `manifest.json` hashes.
3. If the object is legacy `.db.gz`, decompress to a temporary SQLite file.
4. Validate the database opens, has expected tables, and meets the minimum migration version.
5. Create a local safety snapshot of the current database before replacing it.
6. Replace live data through the SQLite backup API (online restore with write lock).
7. For `.tar.gz` archives, replace `LEOTIME_DOCUMENT_ROOT` with the archived `documents/` tree after validation.
8. Record restore status in `backup_settings`.

## Defaults

| Setting | Default |
| --- | --- |
| Schedule hour | **01:00** in the owner's profile timezone (`app_settings.timezone`) |
| Retention | **365 days** |
| Object prefix | `leotime/backups/` |
| Path-style URLs | `false` (enable for many MinIO setups) |

The scheduler runs a lightweight check every minute. A backup executes at most once per local calendar day after the configured hour.

## Configuration

### Environment variables

Add to `.env` (see `.env.example`):

```text
# Required to store S3 credentials from the UI/API (32-byte key)
LEOTIME_SECRETS_KEY=

# Official invoice PDFs (included in tar backups)
LEOTIME_DOCUMENT_ROOT=/data/documents

# Optional; backup scheduler respects DB `enabled` flag regardless
LEOTIME_BACKUP_SCHEDULER_ENABLED=true
```

Generate a secrets key:

```bash
openssl rand -base64 32
```

Without `LEOTIME_SECRETS_KEY`, the API rejects saving S3 credentials with HTTP `503` and a clear error message. Existing encrypted credentials cannot be decrypted if the key is lost.

### UI (Settings → Copias de seguridad / Backups)

The owner configures:

- Enable automatic daily backup
- S3 endpoint (empty = AWS default for the region)
- Region
- Bucket
- Key prefix
- Access key ID
- Secret access key (password field; leave blank to keep existing)
- Path-style addressing toggle
- Schedule hour (0–23, default `1`)
- Retention days (1–3650, default `365`)

Actions:

- **Save** — persist settings (credentials encrypted at rest)
- **Test connection** — uses the current form values (no save required); uploads `.../leotime-connection-test.txt`, then deletes it
- **Run now** — immediate backup
- **Restore** — pick a backup from the list and confirm (see Restore safety)

Status shown:

- Last run time, success/failure, error message, object key
- Last restore time and result

### Provider examples

**AWS S3**

```text
endpoint:   (empty)
region:     eu-central-1
bucket:     my-private-bucket
path-style: false
```

**MinIO**

```text
endpoint:   http://minio:9000
region:     us-east-1
bucket:     leotime-backups
prefix:     leotime/backups/
access key: leotime_backups
secret key: <password from mc admin user add>
path-style: true (optional — auto-enabled for non-AWS endpoints)
```

Important for Docker/VPS:

1. The **endpoint must be reachable from the leotime container**, not only from your laptop. Use the Docker service name (`http://minio:9000`) or the internal VPS IP, not a hostname that resolves only outside the stack.
2. **Access key** = MinIO username (`leotime_backups`). **Secret key** = the password you passed to `mc admin user add`.
3. Path-style is enabled automatically when the endpoint is not `amazonaws.com`. You can still check the box explicitly.
4. **Test connection** sends the current form values; you do not need to save first. If it fails, the API now returns the underlying S3 error (for example `Access Denied`, `connection refused`, or `404`).
5. Verify from the leotime container:

```bash
docker compose exec leotime /app/leotime backup run --force
```

Or test with mc using the same credentials:

```bash
docker exec -it <minio-container> mc alias set test http://127.0.0.1:9000 leotime_backups '<password>'
docker exec -it <minio-container> mc cp /etc/hosts test/leotime-backups/leotime-connection-test.txt
```

## Email notifications

Backup and restore can send optional emails through the existing outbox (`timer_still_running`, `password_reset`, etc.).

Configure them in **Profile → Email notifications**:

| Setting | Default | When it sends |
|---------|---------|---------------|
| Backup success | off | After a manual or scheduled backup finishes with `success` |
| Backup failure | on | After a backup fails (upload, snapshot, retention, etc.) |
| Restore success | off | After a restore finishes with `success` |
| Restore failure | on | After a restore fails |

Emails use the profile locale (`es` / `en`), go to the profile email address, and respect `LEOTIME_MAIL_MODE` (`log` in dev, `smtp` in production).

Outbox kinds: `backup_success`, `backup_failure`, `restore_success`, `restore_failure`.

**Backblaze B2 (S3-compatible)**

```text
endpoint:   https://s3.<region>.backblazeb2.com
region:     <region>
bucket:     <bucket-name>
path-style: false
```

## HTTP API

All routes require a valid session cookie.

```text
GET    /api/v1/backups/settings
PUT    /api/v1/backups/settings
POST   /api/v1/backups/test
POST   /api/v1/backups/run
GET    /api/v1/backups/objects
POST   /api/v1/backups/restore
GET    /api/v1/backups/status
```

### GET /api/v1/backups/settings

```json
{
  "enabled": true,
  "endpoint": "https://s3.eu-central-1.amazonaws.com",
  "region": "eu-central-1",
  "bucket": "my-private-bucket",
  "prefix": "leotime/backups/",
  "accessKeyId": "AKIA...",
  "secretAccessKeyConfigured": true,
  "usePathStyle": false,
  "scheduleHour": 1,
  "retentionDays": 365,
  "lastRunAt": "2026-07-07T23:00:05Z",
  "lastStatus": "success",
  "lastError": "",
  "lastObjectKey": "leotime/backups/leotime-2026-07-07T230005Z.db.gz",
  "lastRestoreAt": null,
  "lastRestoreStatus": "never",
  "lastRestoreError": "",
  "lastRestoreObjectKey": ""
}
```

Secrets are never returned in full. `secretAccessKeyConfigured` indicates whether a stored secret exists.

### PUT /api/v1/backups/settings

```json
{
  "enabled": true,
  "endpoint": "",
  "region": "eu-central-1",
  "bucket": "my-private-bucket",
  "prefix": "leotime/backups/",
  "accessKeyId": "AKIA...",
  "secretAccessKey": "only-when-changing",
  "usePathStyle": false,
  "scheduleHour": 1,
  "retentionDays": 365
}
```

Validation:

- When `enabled=true`: `bucket`, `accessKeyId`, and a configured or provided `secretAccessKey` are required.
- `scheduleHour` must be 0–23.
- `retentionDays` must be 1–3650.

If `secretAccessKey` is omitted or empty and a secret is already stored, the existing secret is kept.

### POST /api/v1/backups/test

Uses saved settings, or optional body override for unsaved drafts (same shape as PUT settings). Uploads a small test object, verifies success, deletes the test object.

Optional body example:

```json
{
  "enabled": true,
  "endpoint": "http://minio:9000",
  "region": "us-east-1",
  "bucket": "leotime-backups",
  "prefix": "leotime/backups/",
  "accessKeyId": "leotime_backups",
  "secretAccessKey": "your-password",
  "usePathStyle": true,
  "scheduleHour": 1,
  "retentionDays": 365
}
```

On failure the API returns HTTP `502` with the underlying S3 error in `error`.

Response:

```json
{
  "ok": true,
  "message": "connection_ok"
}
```

### POST /api/v1/backups/run

Runs a backup immediately. Returns when the backup finishes or fails (timeout 10 minutes).

Response:

```json
{
  "status": "success",
  "objectKey": "leotime/backups/leotime-2026-07-07T230005Z.db.gz",
  "sizeBytes": 1048576,
  "startedAt": "2026-07-07T23:00:00Z",
  "finishedAt": "2026-07-07T23:00:05Z"
}
```

### GET /api/v1/backups/objects

Lists backup objects under the configured prefix, newest first. Used by the restore picker in the UI.

```json
{
  "objects": [
    {
      "key": "leotime/backups/leotime-2026-07-07T230005Z.db.gz",
      "sizeBytes": 1048576,
      "lastModified": "2026-07-07T23:00:05Z"
    }
  ]
}
```

### POST /api/v1/backups/restore

```json
{
  "objectKey": "leotime/backups/leotime-2026-07-07T230005Z.db.gz",
  "confirm": true
}
```

Or restore the newest object:

```json
{
  "latest": true,
  "confirm": true
}
```

Response:

```json
{
  "status": "success",
  "objectKey": "leotime/backups/leotime-2026-07-07T230005Z.db.gz",
  "safetySnapshotPath": "/data/leotime-pre-restore-2026-07-07T231500Z.db.gz",
  "startedAt": "2026-07-07T23:15:00Z",
  "finishedAt": "2026-07-07T23:15:12Z"
}
```

`confirm: true` is required. Without it the API returns `400`.

During restore the API briefly holds a global backup mutex. Concurrent backup/restore requests receive `409 Conflict`.

### GET /api/v1/backups/status

Lightweight poll endpoint with last run and last restore fields only.

## CLI

Same binary as the HTTP server:

```bash
# Run backup now (uses saved settings; exits non-zero on failure)
leotime backup run

# Skip "already ran today" guard
leotime backup run --force

# List remote backup objects
leotime backup list

# Restore a specific object
leotime backup restore --object-key leotime/backups/leotime-2026-07-07T230005Z.db.gz

# Restore newest backup
leotime backup restore --latest

# Restore without interactive confirmation (scripts / recovery)
leotime backup restore --latest --force
```

Inside Docker:

```bash
docker compose exec leotime /app/leotime backup run
docker compose exec leotime /app/leotime backup list
docker compose exec leotime /app/leotime backup restore --latest --force
```

Use CLI restore for disaster recovery when the UI is unavailable. The HTTP server should be running for online restore; for cold recovery, stop the container, replace `/data/leotime.db` manually from a downloaded `.db.gz`, then start again (documented in VPS deploy guide).

## Restore safety

Restore is destructive to the current database contents.

Before replacing data, the app:

1. Creates a local safety snapshot at `/data/leotime-pre-restore-{timestamp}.db.gz` (same directory as the live DB).
2. Validates the downloaded backup opens as SQLite with core tables present.
3. Applies the restore through the SQLite backup API.

The UI requires:

- Selecting a backup from the list
- Checking a confirmation box ("I understand this replaces current data")
- Clicking **Restore**

Recommended operator practice:

- Test restore on a staging copy before relying on production backups.
- Keep `LEOTIME_SECRETS_KEY` backed up separately from S3 credentials.
- Monitor backup metrics and last-run status weekly.

## Database schema

Migration `000007_backup_settings.sql`:

```sql
CREATE TABLE backup_settings (
  user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  enabled INTEGER NOT NULL DEFAULT 0,
  endpoint TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  bucket TEXT NOT NULL DEFAULT '',
  prefix TEXT NOT NULL DEFAULT 'leotime/backups/',
  access_key_id TEXT NOT NULL DEFAULT '',
  secret_access_key_enc TEXT NOT NULL DEFAULT '',
  use_path_style INTEGER NOT NULL DEFAULT 0,
  schedule_hour INTEGER NOT NULL DEFAULT 1,
  retention_days INTEGER NOT NULL DEFAULT 365,
  last_run_at TEXT,
  last_status TEXT NOT NULL DEFAULT 'never',
  last_error TEXT NOT NULL DEFAULT '',
  last_object_key TEXT NOT NULL DEFAULT '',
  last_restore_at TEXT,
  last_restore_status TEXT NOT NULL DEFAULT 'never',
  last_restore_error TEXT NOT NULL DEFAULT '',
  last_restore_object_key TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
```

Single-owner assumption: one row for the bootstrap user.

## Prometheus metrics

```text
leotime_backup_last_success_timestamp
leotime_backup_failures_total
leotime_backup_duration_seconds
leotime_backup_restore_success_total
leotime_backup_restore_failures_total
```

## Security

- S3 credentials are encrypted at rest with AES-256-GCM using `LEOTIME_SECRETS_KEY`.
- Secret access keys are never logged and never returned by GET endpoints.
- Backup and restore routes require authentication (same session as the rest of the API).
- Use a private bucket with least-privilege IAM: `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject`, `s3:ListBucket` on the prefix only.

## Related docs

- VPS deployment and manual cold restore: `docs/06-deploy-vps.md`
- Operations commands: `docs/10-operations.md`
- Scheduler integration: `docs/adr/0003-s3-backup-encryption-and-restore.md`
- Implementation plan: `docs/superpowers/plans/2026-07-07-s3-daily-backups.md`
