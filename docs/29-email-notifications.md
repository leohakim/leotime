# Email Notifications

leotime sends outbound email from the same Go process as the HTTP API. There is no separate queue, scheduler, or mail worker container.

The first supported notification matches Solidtime behavior: **one email when a timer stays open longer than a configurable threshold** (default 8 hours).

## Architecture

```text
main.go
  ├── HTTP server
  └── scheduler (background goroutine)
        ├── scan tick  → detect long-running timers → email_outbox (pending)
        └── outbox tick → SMTP/log send + retries → mark sent / dead
```

Compared with official Solidtime self-hosting:

| Solidtime | leotime |
| --- | --- |
| 4 containers (app, queue, scheduler, database) | 1 container |
| Laravel scheduler + database queue worker | In-process tickers + SQLite outbox |
| Marks `still_active_email_sent_at` when mail is queued | Marks it after successful delivery |

## Behavior

1. Every `LEOTIME_SCHEDULER_SCAN_INTERVAL` (default `10m`), the app lists open timers where:
   - `source = 'timer'` and `ended_at IS NULL`
   - elapsed time ≥ threshold hours (`app_settings.timer_still_running_hours`, default `8`)
   - `still_active_email_sent_at` is empty
   - notifications are enabled (`app_settings.timer_still_running_enabled`, default on)
   - no pending/sent outbox row exists for that timer
2. For each match, a row is inserted into `email_outbox` with kind `timer_still_running`.
3. Every `LEOTIME_OUTBOX_PROCESS_INTERVAL` (default `30s`), pending outbox rows due for send are processed.
4. On successful SMTP/log delivery:
   - outbox status becomes `sent`
   - `time_entries.still_active_email_sent_at` is set
5. On failure, transient errors are retried with exponential backoff until `LEOTIME_MAIL_MAX_ATTEMPTS`.

Only **one notification per open timer** is sent, even if the timer runs for days.

Email subject/body are localized from `users.locale` (`es` or `en`).

## Configuration

Copy from `.env.example`:

```bash
cp .env.example .env
```

### Scheduler

| Variable | Default | Description |
| --- | --- | --- |
| `LEOTIME_SCHEDULER_ENABLED` | `true` | Enable background scan + outbox loops |
| `LEOTIME_SCHEDULER_SCAN_INTERVAL` | `10m` | How often to detect long-running timers |
| `LEOTIME_OUTBOX_PROCESS_INTERVAL` | `30s` | How often to send/retry outbox mail |

Set `LEOTIME_SCHEDULER_ENABLED=false` for local debugging when you do not want background ticks.

### Mail transport

| Variable | Default | Description |
| --- | --- | --- |
| `LEOTIME_MAIL_MODE` | `log` | `log` (stdout) or `smtp` |
| `LEOTIME_MAIL_FROM` | `no-reply@localhost` | From address |
| `LEOTIME_MAIL_FROM_NAME` | `leotime` | From display name |
| `LEOTIME_PUBLIC_BASE_URL` | `http://127.0.0.1:8080` | Link included in the email body |
| `LEOTIME_SMTP_HOST` | — | Required when `MAIL_MODE=smtp` |
| `LEOTIME_SMTP_PORT` | `587` | SMTP port (`587` STARTTLS or `465` implicit TLS) |
| `LEOTIME_SMTP_USERNAME` | — | Optional SMTP auth |
| `LEOTIME_SMTP_PASSWORD` | — | Optional SMTP auth |
| `LEOTIME_SMTP_TLS` | `true` | Use TLS where applicable |

Local development:

```bash
LEOTIME_MAIL_MODE=log
```

Production example:

```bash
LEOTIME_MAIL_MODE=smtp
LEOTIME_MAIL_FROM=no-reply@your-domain.com
LEOTIME_MAIL_FROM_NAME=leotime
LEOTIME_PUBLIC_BASE_URL=https://leotime.example.com
LEOTIME_SMTP_HOST=smtp.example.com
LEOTIME_SMTP_PORT=587
LEOTIME_SMTP_USERNAME=...
LEOTIME_SMTP_PASSWORD=...
LEOTIME_COOKIE_SECURE=true
```

### Retry policy

| Variable | Default | Description |
| --- | --- | --- |
| `LEOTIME_MAIL_MAX_ATTEMPTS` | `5` | Max send attempts before `dead` |
| `LEOTIME_MAIL_RETRY_BASE` | `1m` | Base backoff delay |
| `LEOTIME_MAIL_RETRY_MAX` | `6h` | Cap between retries |

Approximate retry spacing: 1m → 2m → 4m → 8m → 16m (plus jitter).

### Per-user threshold (profile settings)

Configure in **Settings** (`/settings`):

| Field | Default | Meaning |
| --- | --- | --- |
| `timerStillRunningEnabled` | on | Send still-running emails |
| `timerStillRunningHours` | `8` | Hours before the first alert (1–24) |

These map to `app_settings.timer_still_running_enabled` and `app_settings.timer_still_running_hours`.

SQL fallback:

```sql
UPDATE app_settings
SET timer_still_running_hours = 6, timer_still_running_enabled = 1
WHERE user_id = (SELECT id FROM users LIMIT 1);
```

## Observability

Prometheus metrics on `/metrics`:

```text
leotime_still_running_timers_detected_total
leotime_email_outbox_sent_total
leotime_email_outbox_retried_total
leotime_email_outbox_dead_total
leotime_scheduler_scan_errors_total
leotime_scheduler_outbox_errors_total
```

Example:

```bash
curl -s http://127.0.0.1:8080/metrics | rg '^leotime_'
```

With `LEOTIME_MAIL_MODE=log`, successful sends appear in container logs:

```bash
make logs
```

Look for lines such as `scheduler enqueued N still-running timer notification(s)` and `mail log mode: from=...`.

## Manual verification

1. Start the stack: `make up`
2. Log in and start a timer.
3. Backdate the timer start in SQLite (test only):

```bash
docker compose exec leotime sh -lc "sqlite3 /data/leotime.db \"UPDATE time_entries SET started_at = datetime('now', '-9 hours') WHERE ended_at IS NULL AND source = 'timer';\""
```

4. Wait up to one scan interval (`10m` by default) or temporarily set `LEOTIME_SCHEDULER_SCAN_INTERVAL=30s` in Compose and restart.
5. Check logs or `/metrics` for enqueue/send counters.

## Solidtime import note

Solidtime exports include `still_active_email_sent_at` on time entries. The importer reads that column but does not persist it yet; after import work lands, migrated timers that already received Solidtime mail should not be notified again.

## Code map

| Area | Location |
| --- | --- |
| Scheduler loops | `apps/api/internal/scheduler/scheduler.go` |
| Timer detection + enqueue | `apps/api/internal/notify/still_running.go` |
| Email templates | `apps/api/internal/notify/templates.go` |
| Outbox + retries | `apps/api/internal/outbox/` |
| Mail senders | `apps/api/internal/mail/` |
| Store queries | `apps/api/internal/store/still_running.go` |
| Wiring | `apps/api/cmd/leotime/main.go` |
| Migration | `apps/api/internal/db/migrations/000005_email_notifications.sql` |
| Metrics | `apps/api/internal/metrics/metrics.go` |

## Related docs

- VPS SMTP deployment: `docs/06-deploy-vps.md`
- Operations and footprint: `docs/10-operations.md`
- Timer API: `docs/19-timers-api.md`
- Data model: `docs/03-data-model.md`
- ADR: `docs/adr/0002-in-process-scheduler-outbox.md`
