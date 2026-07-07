# S3 Daily Backups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add configurable S3 daily backups with 01:00 default schedule, 365-day retention, in-app restore, CLI commands, scheduler integration, and a Settings UI panel.

**Architecture:** Extend the existing in-process scheduler with a backup tick; implement `internal/backup` for snapshot/gzip/S3 upload/retention/restore; encrypt credentials with `LEOTIME_SECRETS_KEY`; expose authenticated HTTP routes and `leotime backup` CLI subcommands; add a Backups section in Settings.

**Tech Stack:** Go 1.26, AWS SDK v2, modernc.org/sqlite, React/Vite, existing chi router and scheduler.

**Spec:** `docs/31-s3-daily-backups.md`, ADR `docs/adr/0003-s3-backup-encryption-and-restore.md`

---

## File map

| File | Responsibility |
| --- | --- |
| `apps/api/internal/db/migrations/000007_backup_settings.sql` | Schema |
| `apps/api/internal/backup/crypto/secrets.go` | AES-256-GCM encrypt/decrypt |
| `apps/api/internal/backup/snapshot/snapshot.go` | SQLite backup API → file |
| `apps/api/internal/backup/s3/client.go` | S3 upload/list/delete |
| `apps/api/internal/backup/runner.go` | Run backup + retention |
| `apps/api/internal/backup/restore.go` | Download, validate, safety snapshot, apply |
| `apps/api/internal/backup/schedule.go` | Due check (timezone, once/day) |
| `apps/api/internal/store/backup.go` | Settings CRUD + status |
| `apps/api/internal/httpapi/backups.go` | HTTP handlers |
| `apps/api/internal/scheduler/scheduler.go` | Add backup tick |
| `apps/api/cmd/leotime/main.go` | Wire scheduler + `backup` subcommand |
| `apps/api/internal/config/config.go` | `LEOTIME_SECRETS_KEY`, backup flags |
| `apps/web/src/lib/backupSettingsUi.tsx` | Settings UI panel |
| `apps/web/src/lib/api.ts` | Client types and fetchers |
| `docs/31-s3-daily-backups.md` | Product/API reference (done) |

---

### Task 1: Migration and config env vars

**Files:**
- Create: `apps/api/internal/db/migrations/000007_backup_settings.sql`
- Modify: `.env.example`
- Modify: `apps/api/internal/config/config.go`, `config_test.go`

- [ ] **Step 1: Add migration**

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

- [ ] **Step 2: Extend config**

Add to `Config`:

```go
SecretsKey              string
BackupSchedulerEnabled  bool
```

Env:

```text
LEOTIME_SECRETS_KEY=
LEOTIME_BACKUP_SCHEDULER_ENABLED=true
```

- [ ] **Step 3: Run migration test**

```bash
cd apps/api && go test ./internal/db/... -count=1
```

Expected: PASS

---

### Task 2: Secret encryption

**Files:**
- Create: `apps/api/internal/backup/crypto/secrets.go`
- Create: `apps/api/internal/backup/crypto/secrets_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestEncryptDecryptRoundTrip(t *testing.T) {
    key := make([]byte, 32)
    _, _ = rand.Read(key)
    enc, err := Encrypt([]byte("secret"), key)
    if err != nil { t.Fatal(err) }
    plain, err := Decrypt(enc, key)
    if err != nil { t.Fatal(err) }
    if string(plain) != "secret" { t.Fatalf("got %q", plain) }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd apps/api && go test ./internal/backup/crypto/... -count=1 -v
```

- [ ] **Step 3: Implement AES-256-GCM**

Parse `LEOTIME_SECRETS_KEY` as base64 or hex (32 bytes). Return clear error if missing/invalid when encryption is needed.

- [ ] **Step 4: Run test — expect PASS**

---

### Task 3: Store layer

**Files:**
- Create: `apps/api/internal/store/backup.go`
- Create: `apps/api/internal/store/backup_test.go`

- [ ] **Step 1: Define types**

```go
type BackupSettings struct {
    Enabled                   bool   `json:"enabled"`
    Endpoint                  string `json:"endpoint"`
    Region                    string `json:"region"`
    Bucket                    string `json:"bucket"`
    Prefix                    string `json:"prefix"`
    AccessKeyID               string `json:"accessKeyId"`
    SecretAccessKeyConfigured bool   `json:"secretAccessKeyConfigured"`
    UsePathStyle              bool   `json:"usePathStyle"`
    ScheduleHour              int    `json:"scheduleHour"`
    RetentionDays             int    `json:"retentionDays"`
    LastRunAt                 *string `json:"lastRunAt"`
    LastStatus                string `json:"lastStatus"`
    LastError                 string `json:"lastError"`
    LastObjectKey             string `json:"lastObjectKey"`
    LastRestoreAt             *string `json:"lastRestoreAt"`
    LastRestoreStatus         string `json:"lastRestoreStatus"`
    LastRestoreError          string `json:"lastRestoreError"`
    LastRestoreObjectKey      string `json:"lastRestoreObjectKey"`
}
```

- [ ] **Step 2: Tests**

- Get empty settings for user → defaults (`scheduleHour=1`, `retentionDays=365`)
- Upsert with secret → `SecretAccessKeyConfigured=true`, GET never exposes secret
- Upsert without secret field → keeps previous encrypted secret

- [ ] **Step 3: Implement** `GetBackupSettings`, `UpsertBackupSettings`, `UpdateBackupRunStatus`, `UpdateBackupRestoreStatus`

- [ ] **Step 4: Run tests**

```bash
cd apps/api && go test ./internal/store/... -run Backup -count=1
```

---

### Task 4: SQLite snapshot

**Files:**
- Create: `apps/api/internal/backup/snapshot/snapshot.go`
- Create: `apps/api/internal/backup/snapshot/snapshot_test.go`

- [ ] **Step 1: Test** temp DB with one row → snapshot file → open copy → row present

- [ ] **Step 2: Implement** `SnapshotToFile(ctx, dbPath, destPath) error` using SQLite backup API

- [ ] **Step 3: Add** `GzipFile(src, dest) error`

---

### Task 5: S3 client

**Files:**
- Create: `apps/api/internal/backup/s3/client.go`
- Create: `apps/api/internal/backup/s3/client_test.go`
- Modify: `apps/api/go.mod`

- [ ] **Step 1: Add dependency**

```bash
cd apps/api && go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/credentials github.com/aws/aws-sdk-go-v2/service/s3
```

- [ ] **Step 2: Implement interface**

```go
type Client interface {
    Put(ctx context.Context, key string, body io.Reader, contentType string) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]Object, error)
}
```

Support custom `BaseEndpoint`, `UsePathStyle`, static credentials.

- [ ] **Step 3: Test with httptest mock S3 API**

---

### Task 6: Backup runner + retention

**Files:**
- Create: `apps/api/internal/backup/runner.go`
- Create: `apps/api/internal/backup/runner_test.go`
- Create: `apps/api/internal/backup/schedule.go`
- Modify: `apps/api/internal/metrics/metrics.go`

- [ ] **Step 1: Implement** `RunBackup(ctx, deps) (*RunResult, error)`

Steps: load settings → snapshot → gzip → upload → prune objects older than `retentionDays` → update status.

Object key: `leotime-{20060102T150405Z}.db.gz` under prefix.

- [ ] **Step 2: Implement** `IsBackupDue(settings, timezone, now) bool`

True when: enabled, local hour >= scheduleHour (default 1), and last successful run is not today (local date).

- [ ] **Step 3: Global mutex** `var jobMu sync.Mutex` shared with restore

- [ ] **Step 4: Metrics**

```go
BackupLastSuccessTimestamp prometheus.GaugeFunc
BackupFailuresTotal      prometheus.Counter
BackupDurationSeconds    prometheus.Histogram
```

- [ ] **Step 5: Tests** with mock S3 and temp DB

---

### Task 7: Restore

**Files:**
- Create: `apps/api/internal/backup/restore.go`
- Create: `apps/api/internal/backup/restore_test.go`

- [ ] **Step 1: Implement** `RestoreBackup(ctx, deps, objectKey) (*RestoreResult, error)`

1. Acquire `jobMu`
2. Download + gunzip to temp file
3. Validate SQLite (tables: `users`, `time_entries`, `clients`)
4. Safety snapshot: gzip current DB to `/data/leotime-pre-restore-{ts}.db.gz`
5. SQLite backup API from temp file into live DB connection
6. Update restore status fields

- [ ] **Step 2: Test** backup a tiny DB, restore into empty second DB, verify row counts

- [ ] **Step 3: Metrics** `BackupRestoreSuccessTotal`, `BackupRestoreFailuresTotal`

---

### Task 8: Scheduler integration

**Files:**
- Modify: `apps/api/internal/scheduler/scheduler.go`
- Modify: `apps/api/cmd/leotime/main.go`

- [ ] **Step 1: Inject** `*backup.Runner` (or factory) into `Scheduler`

- [ ] **Step 2: Add ticker** every `1m` when `BackupSchedulerEnabled`

On tick: load settings for bootstrap user → if `IsBackupDue` → `RunBackup`

- [ ] **Step 3: Log** success/failure; increment metrics

- [ ] **Step 4: Wire in main.go** alongside existing email scheduler

---

### Task 9: CLI

**Files:**
- Modify: `apps/api/cmd/leotime/main.go`
- Create: `apps/api/cmd/leotime/backup_cmd.go`

- [ ] **Step 1: Dispatch** `leotime backup` subcommand

```bash
leotime backup run [--force]
leotime backup list
leotime backup restore --object-key KEY | --latest [--force]
```

- [ ] **Step 2: `run --force`** skips "already ran today" guard

- [ ] **Step 3: `restore --force`** skips interactive confirm (for scripts)

- [ ] **Step 4: Exit codes** 0 success, 1 failure; JSON summary to stdout for `run` and `restore`

- [ ] **Step 5: Test** with `go test` on flag parsing helpers

---

### Task 10: HTTP API

**Files:**
- Create: `apps/api/internal/httpapi/backups.go`
- Create: `apps/api/internal/httpapi/backups_test.go`
- Modify: `apps/api/internal/httpapi/router.go`

- [ ] **Step 1: Register routes**

```go
r.Get("/backups/settings", ...)
r.Put("/backups/settings", ...)
r.Post("/backups/test", ...)
r.Post("/backups/run", ...)
r.Get("/backups/objects", ...)
r.Post("/backups/restore", ...)
r.Get("/backups/status", ...)
```

- [ ] **Step 2: Handlers**

- PUT validates scheduleHour 0–23, retentionDays 1–3650
- POST restore requires `confirm: true` or returns 400
- POST run returns 409 if mutex held
- Missing secrets key → 503 on PUT with credentials

- [ ] **Step 3: Router tests** with temp SQLite + mock S3

```bash
cd apps/api && go test ./internal/httpapi/... -run Backup -count=1
```

---

### Task 11: Frontend UI

**Files:**
- Create: `apps/web/src/lib/backupSettingsUi.tsx`
- Modify: `apps/web/src/lib/api.ts`
- Modify: `apps/web/src/lib/profileSettingsUi.tsx` or `App.tsx`
- Modify: i18n message files

- [ ] **Step 1: API client** types matching `docs/31-s3-daily-backups.md`

- [ ] **Step 2: Panel "Copias de seguridad" / "Backups"** in Settings

Fields: enabled, endpoint, region, bucket, prefix, access key, secret key, path-style, schedule hour (default 1), retention days (default 365).

Buttons: Save, Test connection, Run now.

- [ ] **Step 3: Restore section**

- Fetch `GET /backups/objects` → table (date, size, key)
- Select row + confirmation checkbox + Restore button
- Show last restore status

- [ ] **Step 4: States** loading, error (including 503 missing secrets key), success toasts

- [ ] **Step 5: Vitest** render form defaults (`scheduleHour=1`, `retentionDays=365`)

```bash
npm --workspace @leotime/web test -- --run backupSettings
```

---

### Task 12: Makefile and smoke (optional targets)

**Files:**
- Modify: `Makefile` (if helpful)

- [ ] **Step 1: Document in help** — backup CLI runs via `docker compose exec leotime /app/leotime backup ...`

No new Make target required unless adding `make backup-run` wrapper.

---

### Task 13: Final verification

- [ ] **Step 1:**

```bash
make pre-commit
```

- [ ] **Step 2: Manual test** with MinIO or real bucket

1. Set `LEOTIME_SECRETS_KEY`
2. Configure in UI → Test → Run now → object in bucket
3. List objects in UI → Restore with confirm → data intact
4. `leotime backup run --force` from CLI
5. Wait/simulate next day logic with `--force` and changed clock in unit test

---

## Commit sequence (suggested)

1. `docs: add S3 daily backups spec, ADR, and implementation plan`
2. `feat: add backup settings migration and encrypted store`
3. `feat: add SQLite snapshot and S3 backup runner`
4. `feat: add backup restore, scheduler tick, and CLI`
5. `feat: add backups HTTP API and Settings UI panel`

---

## Out of scope for this plan

- Email alert on backup failure
- SSE-KMS / bucket encryption flags
- Multi-user backup settings
