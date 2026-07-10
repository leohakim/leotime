# Known Gaps and Audit

Audit date: **2026-07-08** (post-MVP). This document records verified limitations, bugs, and hardening work—not MVP blockers, but items to track before trusting leotime with irreplaceable production data at scale.

Severity legend:

- **Critical** — data loss, corruption, or cross-user leakage risk
- **High** — security or major functional gap
- **Medium** — incorrect behavior under edge cases
- **Low** — polish, dead code, or doc/UX mismatches

Items marked **Fixed** were addressed in the same documentation pass that produced this file.

---

## Critical

### C1. Live DB restore while API keeps serving writes — **Fixed**

**Location:** `apps/api/internal/backup/service.go` (`Restore`), `apps/api/internal/maintenance`, `apps/api/internal/httpapi/router.go`

**Issue:** Restore hot-swaps the SQLite file while HTTP handlers and the scheduler continue. Concurrent writes can corrupt the database or produce a partial restore.

**Fix:** Enter maintenance mode for the duration of restore; middleware returns `503 maintenance_mode` for `/api/*` (except health); scheduler skips work while maintenance is active. Successful restore returns `requiresRestart: true` and the UI reloads the page.

**CLI note:** Prefer restore with the HTTP server stopped, or use the in-app restore flow.

### C2. Offline sync did not remap local IDs for projects/tasks — **Fixed**

**Location:** `apps/web/src/lib/offline/sync.ts`

**Issue:** Creating a client offline then a project referencing `local_cli_*` failed on sync because only time entries/timers remapped foreign keys.

**Fix:** `remapProjectInput` / `remapTaskInput` applied before API create.

### C3. Logout left cached data and offline queue — **Fixed**

**Location:** `apps/web/src/features/shell/DashboardShell.tsx`, `apps/web/src/lib/offline/db.ts`

**Issue:** Logout only invalidated the session query. React Query cache and IndexedDB mutations could leak data or sync into another account on shared browsers.

**Fix:** `queryClient.clear()`, `clearQueuedMutations()`, `clearIdMappings()` on successful logout.

### C4. Session fetch failure showed login screen — **Fixed**

**Location:** `apps/web/src/App.tsx`

**Issue:** Network or 500 errors on `/api/v1/session` were indistinguishable from “not logged in”.

**Fix:** Dedicated error screen with retry when `sessionQuery.isError`.

---

## High

### H1. Password change did not invalidate other sessions — **Fixed**

**Location:** `apps/api/internal/store/profile.go` (`ChangePassword`)

**Issue:** Stolen session cookies remained valid after the user changed password (password reset did clear sessions).

**Fix:** `DELETE FROM sessions WHERE user_id = ?` after password hash update. User must log in again after password change.

### H2. Static file handler path traversal risk — **Fixed**

**Location:** `apps/api/internal/httpapi/security.go` (`safeStaticFilePath`)

**Fix:** Resolve absolute paths and reject any file outside the static root via `filepath.Rel`.

### H3. `/metrics` unauthenticated — **Fixed**

**Location:** `apps/api/internal/httpapi/router.go` (`metrics`)

**Fix:** In `LEOTIME_ENV=production`, `/metrics` returns 404 unless `LEOTIME_METRICS_TOKEN` is set; when set, require Bearer token or `?token=` query param.

### H4. Default bootstrap credentials — **Fixed**

**Location:** `apps/api/internal/config/config.go` (`Validate`)

**Fix:** When `LEOTIME_ENV=production`, startup fails unless `LEOTIME_BOOTSTRAP_PASSWORD` is explicitly set and not `change-me-now`.

### H5. Structured `ApiError` only on `apiJSON` paths — **Fixed**

**Location:** `apps/web/src/lib/api.ts`

**Fix:** Shared `ensureOk` / `apiGet` / `apiDelete` / `apiPost` helpers using `parseApiErrorPayload`; GET, DELETE, auth, and most mutations migrated.

### H6. `taskProjectRequired` not enforced in UI — **Fixed**

**Location:** Profile setting vs `TaskPanel`, timer picker, manual entry

**Fix:** Shell loads profile flag; forms validate with `validateProjectRequired` and map server field errors.

### H7. Manual time entry list uses week-scoped query — **Fixed**

**Location:** `DashboardShell.tsx` + `timeEntryUi.tsx`

**Issue:** Manual entry “directory” shows current week only, sliced to 12 rows.

**Fix:** Dedicated 90-day query, honest count label, and paginated “Load more” (25 rows per page).

### H8. Offline queue stops on first failure — **Fixed**

**Location:** `apps/web/src/lib/offline/sync.ts` (`flushOfflineQueue`)

**Fix:** Continue flushing independent ops after a failed mutation; unit test in `offline.test.ts`.

### H9. Inline timesheet save used wrong cache key — **Fixed**

**Location:** `apps/web/src/lib/timeEntryUi.tsx`

**Issue:** `setQueryData(['time-entries'], …)` vs live key `['time-entries', view, period]`.

**Fix:** Use `patchTimeEntriesCache`.

### H10. Offline scope is create-heavy only — **Fixed**

**Location:** CRUD panels call API directly for update/archive/delete

**Fix:** UI documents limitation via `offlineCreatesOnly` when offline with no pending queue items.

---

## Medium

| ID | Title | Location | Notes |
| --- | --- | --- | --- |
| M1 | Restore `latest` picks unsorted S3 list order — **Fixed** | `backup/service.go` | Sort by `LastModified` before picking |
| M2 | Restore validation minimal — **Fixed** | `backup/snapshot/snapshot.go` | `integrity_check` + migration version |
| M3 | Prune failure marks backup run failed after upload — **Fixed** | `backup/service.go` | Treat prune as best-effort |
| M4 | Invoice status transitions unconstrained — **Fixed** | `store/invoice.go` | Enforce draft→issued and issued→paid matrix |
| M5 | Archived tags attachable to entries — **Fixed** | `store/time_entry.go` | Reject archived tag IDs |
| M6 | `StartTimer` ignores client `startedAt` — **Fixed** | `store/timer.go` | Honor optional RFC3339 `startedAt` on start |
| M7 | Backup errors leak S3 internals to client — **Fixed** | `httpapi/backups.go` | Generic `backup_remote_storage_failed` |
| M8 | Backup resolve errors lack `fields` — **Fixed** | `backup/config_resolve.go` | Use `BackupSettingsValidationError` |
| M9 | `rates` table unused | migration 000001 | Implement or deprecate |
| M10 | Outbox double-send if MarkSent fails — **Fixed** | `outbox/processor.go` | Quarantine after delivery when mark sent cannot complete |
| M11 | No HTTP tests for backup routes — **Fixed** | `httpapi/backups_test.go` | Auth, confirm, secrets key, remote errors |
| M12 | No login/forgot-password rate limits — **Fixed** | `httpapi/ratelimit.go` | 10 login / 15 min per IP; 5 forgot-password / hour per IP+email |
| M13 | Session DB errors returned as 401 — **Fixed** | `router.go` `lookupSessionUser` | Return 503 on lookup failures |
| M14 | JSON body size unlimited (except import) — **Fixed** | `httpapi/json_body.go` | 1 MiB default via `MaxBytesReader` |
| M15 | Report date params unvalidated — **Fixed** | `httpapi/reports.go` | Return 400 on bad range |
| M16 | Dashboard restart timer bypasses offline — **Fixed** | `dashboardUi.tsx` | Use offline `startTimer` + timer cache patch |
| M17 | Profile forms ignore `ApiError.fields` — **Fixed** | `profileSettingsUi.tsx` | Map field errors on profile and password |
| M18 | Report export before preview — **Fixed** | `reportUi.tsx` | Disable until preview OK |
| M19 | Import invalidates wrong query key — **Fixed** | `importExportUi.tsx` | Use `dashboard-stats` |
| M20 | `fetchOverview` unused; nav “Overview” is reports — **Fixed** | shell + `api.ts` | Remove dead client; nav/title use `reporting`; invalidate `dashboard-stats` |
| M21 | Multiple open timers; UI controls first only | `DashboardShell.tsx` | Product decision |
| M22 | Shell queries lack error states — **Fixed** | CRUD panels | `QueryErrorBanner` with retry |
| M23 | Locale/theme dual localStorage vs profile — **Fixed** | App + profile | Hydrate preferences from profile on login |
| M24 | Backup restore does not refresh app state — **Fixed** | `backupSettingsUi.tsx` | Full reload when `requiresRestart` |
| M25 | Invoice draft with local client IDs — **Fixed** | `invoiceUi.tsx` | Filter `isLocalId` |

---

## Low

| ID | Title | Notes |
| --- | --- | --- |
| L1 | Expired sessions/tokens never purged | Scheduler cleanup job |
| L2 | `ErrInvalidTimerInput` unused — **Fixed** | Use for timer `startedAt` validation |
| L3 | Backup field `scheduleHourUtc` vs JSON `scheduleHour` — **Fixed** | Validation field name aligned to `scheduleHour` |
| L4 | `writeJSON` ignores encode errors | Log failures |
| L5 | Restore response exposes filesystem path — **Fixed** | Omit `safetySnapshotPath` from JSON |
| L6 | Shared reports nav placeholder — **Fixed** | Hide nav link until feature exists |
| L7 | Invoice edit UI missing (PATCH exists) | Future slice |
| L8 | Auth form pre-filled dev credentials — **Fixed** | Empty defaults outside dev builds |
| L9 | Import summary hardcoded English — **Fixed** | `importEntitySeen` i18n key |
| L10 | Decorative timesheet “select all” checkbox — **Fixed** | Removed non-functional control |
| L11 | `isNetworkFailure` only catches `TypeError` — **Fixed** | Treat 502/503 as offline |

---

## How to Re-run This Audit

```bash
make pre-commit
make smoke
make test-e2e          # if Playwright installed
cd apps/api && go test ./...
```

Manual checks before production:

1. Change bootstrap password and verify login with old password fails.
2. Configure SMTP; trigger still-running timer mail (or log sender).
3. Configure S3 backup; run `backup run --force`; verify object in bucket.
4. Test offline: create timer offline, reconnect, confirm sync.
5. Test restore on a **copy** of data, not live production DB.

Update this document when items are fixed or re-prioritized.
