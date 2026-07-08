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

### C1. Live DB restore while API keeps serving writes

**Location:** `apps/api/internal/backup/service.go` (`Restore`, `copyDatabaseInto`)

**Issue:** Restore hot-swaps the SQLite file while HTTP handlers and the scheduler continue. Concurrent writes can corrupt the database or produce a partial restore.

**Mitigation today:** Run restore only during maintenance; prefer CLI on a stopped container.

**Recommended fix:** Maintenance mode (reject writes), or restore-then-restart workflow documented and enforced in UI/CLI.

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

### H2. Static file handler path traversal risk

**Location:** `apps/api/internal/httpapi/router.go` (`notFound`)

**Issue:** `filepath.Join(staticDir, cleanPath)` without verifying the result stays under `StaticDir`.

**Fix:** Resolve absolute path and reject if outside static root.

### H3. `/metrics` unauthenticated

**Location:** `apps/api/internal/httpapi/router.go`

**Issue:** Prometheus metrics exposed without auth (backup counters, outbox stats).

**Fix:** Bind to internal network, reverse-proxy auth, or separate listener.

### H4. Default bootstrap credentials

**Location:** `apps/api/internal/config/config.go`

**Issue:** `admin@example.com` / `change-me-now` on empty database.

**Fix:** Fail startup in production without explicit `LEOTIME_BOOTSTRAP_PASSWORD`; document mandatory rotation.

### H5. Structured `ApiError` only on `apiJSON` paths

**Location:** `apps/web/src/lib/api.ts`

**Issue:** GET/DELETE/auth helpers still throw plain `Error('request_failed:…')`.

**Fix:** Shared `apiFetch` wrapper using `parseApiErrorPayload`.

### H6. `taskProjectRequired` not enforced in UI

**Location:** Profile setting vs `TaskPanel`, timer picker, manual entry

**Issue:** Backend rejects tasks without project when setting is on; frontend still treats project as optional.

**Fix:** Load profile flag in shell; validate forms and map server field errors.

### H7. Manual time entry list uses week-scoped query

**Location:** `DashboardShell.tsx` + `timeEntryUi.tsx`

**Issue:** Manual entry “directory” shows current week only, sliced to 12 rows.

**Fix:** Dedicated query (broader range) and honest count label or pagination.

### H8. Offline queue stops on first failure

**Location:** `apps/web/src/lib/offline/sync.ts` (`flushOfflineQueue`)

**Issue:** One bad mutation blocks all later sync.

**Fix:** Continue independent ops where safe; skip/retry UI.

### H9. Inline timesheet save used wrong cache key — **Fixed**

**Location:** `apps/web/src/lib/timeEntryUi.tsx`

**Issue:** `setQueryData(['time-entries'], …)` vs live key `['time-entries', view, period]`.

**Fix:** Use `patchTimeEntriesCache`.

### H10. Offline scope is create-heavy only

**Location:** CRUD panels call API directly for update/archive/delete

**Issue:** Edits/deletes fail offline without queue fallback.

**Fix:** Extend queue or document “offline = create timers/entries only” prominently in UI.

---

## Medium

| ID | Title | Location | Notes |
| --- | --- | --- | --- |
| M1 | Restore `latest` picks unsorted S3 list order | `backup/service.go` | Sort by `LastModified` before picking |
| M2 | Restore validation minimal | `backup/snapshot/snapshot.go` | Check migrations + `integrity_check` |
| M3 | Prune failure marks backup run failed after upload | `backup/service.go` | Treat prune as best-effort |
| M4 | Invoice status transitions unconstrained | `store/invoice.go` | Add transition matrix |
| M5 | Archived tags attachable to entries | `store/time_entry.go` | Reject archived tag IDs |
| M6 | `StartTimer` ignores client `startedAt` | `store/timer.go` | Honor or remove from API |
| M7 | Backup errors leak S3 internals to client | `httpapi/backups.go` | Generic client message |
| M8 | Backup resolve errors lack `fields` | `backup/config_resolve.go` | Use `validationError` |
| M9 | `rates` table unused | migration 000001 | Implement or deprecate |
| M10 | Outbox double-send if MarkSent fails | `outbox/processor.go` | Idempotency / ordering |
| M11 | No HTTP tests for backup routes | `router_test.go` | Add coverage |
| M12 | No login/forgot-password rate limits | `httpapi/router.go` | Throttle per IP/email |
| M13 | Session DB errors returned as 401 | `router.go` `currentUser` | Distinguish 503 |
| M14 | JSON body size unlimited (except import) | handlers | `MaxBytesReader` helper |
| M15 | Report date params unvalidated | `httpapi/reports.go` | Return 400 on bad range |
| M16 | Dashboard restart timer bypasses offline | `dashboardUi.tsx` | Use offline mutations |
| M17 | Profile forms ignore `ApiError.fields` | `profileSettingsUi.tsx` | Map field errors |
| M18 | Report export before preview | `reportUi.tsx` | Disable until preview OK |
| M19 | Import invalidates wrong query key — **Fixed** | `importExportUi.tsx` | Use `dashboard-stats` |
| M20 | `fetchOverview` unused; nav “Overview” is reports | shell + `api.ts` | Wire or rename nav |
| M21 | Multiple open timers; UI controls first only | `DashboardShell.tsx` | Product decision |
| M22 | Shell queries lack error states | CRUD panels | Show error pill |
| M23 | Locale/theme dual localStorage vs profile | App + profile | Single source of truth |
| M24 | Backup restore does not refresh app state | `backupSettingsUi.tsx` | Full reload after restore |
| M25 | Invoice draft with local client IDs | `invoiceUi.tsx` | Filter `isLocalId` |

---

## Low

| ID | Title | Notes |
| --- | --- | --- |
| L1 | Expired sessions/tokens never purged | Scheduler cleanup job |
| L2 | `ErrInvalidTimerInput` unused | Remove or use |
| L3 | Backup field `scheduleHourUtc` vs JSON `scheduleHour` | Align names |
| L4 | `writeJSON` ignores encode errors | Log failures |
| L5 | Restore response exposes filesystem path | Omit from API |
| L6 | Shared reports nav placeholder | Hide or implement |
| L7 | Invoice edit UI missing (PATCH exists) | Future slice |
| L8 | Auth form pre-filled dev credentials | Empty in production builds |
| L9 | Import summary hardcoded English | i18n keys |
| L10 | Decorative timesheet “select all” checkbox | Remove or implement |
| L11 | `isNetworkFailure` only catches `TypeError` | Treat 502/503 as offline |

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
