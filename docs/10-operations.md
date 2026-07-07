# Operations

`leotime` is Docker-first. The Makefile is the main operator interface for local development, smoke tests, deploy checks, performance tests, and observability.

## Daily Commands

```bash
make help
make setup
make dev
make test
make up
make smoke
make logs
make down
make resources
```

## Docker Stack

Start the app:

```bash
make up
```

Stop it:

```bash
make down
```

Tail logs:

```bash
make logs
```

## Smoke Tests

`make smoke` checks:

- `/api/health`
- `/api/v1/session`
- `/metrics`
- `/`

Use another URL:

```bash
make smoke BASE_URL=https://leotime.example.com
```

## Metrics

The API exposes Prometheus metrics at:

```text
/metrics
```

Start Prometheus and Grafana:

```bash
make metrics
```

Then open:

```text
Prometheus: http://127.0.0.1:9090
Grafana:    http://127.0.0.1:3001
```

Default local Grafana credentials:

```text
admin
admin
```

Change those before exposing the observability profile outside a local network.

## Resource Measurement

Compare leotime against Solidtime with the same workload shape, not just idle containers.

### Quick measurement

With Docker running:

```bash
make up
make resources
```

Under load:

```bash
make resources WITH_LOAD=1 K6_VUS=10 K6_DURATION=30s
```

Longer idle window:

```bash
make resources SAMPLE_SECONDS=300
```

The script samples `docker stats`, prints average/peak CPU and RAM for the `leotime` container, and reads `/metrics` for Go process memory.

### What leotime runs today

Default Compose starts **one** service:

| Service | Role |
| --- | --- |
| `leotime` | Go API, embedded SQLite, built static web, in-process scheduler, email outbox |

The scheduler sends still-running timer notifications. See `docs/29-email-notifications.md`.

Prometheus and Grafana are optional (`make metrics`) and should not be counted in product footprint unless you deploy them.

### Solidtime reference footprint

Example snapshot from a VPS running official Solidtime:

| Container | RAM |
| --- | ---: |
| queue | ~171 MiB |
| scheduler | ~37 MiB |
| app | ~558 MiB |
| database | ~51 MiB |
| **Total** | **~817 MiB** |

That stack also runs background workers for queues, schedules, and mail. leotime covers still-running timer mail inside the same container; password reset and other mail types are not implemented yet.

### Fair comparison notes

Measure leotime after importing a representative Solidtime ZIP and using the app normally (timer, timesheet, reports). Compare:

1. **Total RAM across containers** (Solidtime) vs **single `leotime` container**.
2. **Idle vs active** usage. Timer polling, report exports, and the email scheduler change CPU slightly.
3. **Remaining mail features**:
   - password reset and other transactional templates
   - profile UI for still-running threshold (database columns exist today)

Re-measure with `make resources` after enabling the scheduler and SMTP in production-like settings.

### Measured baseline (2026-07-06)

First local measurement on the production Docker image (`make up`), empty bootstrap database (`leotime.db` ≈ 0.17 MiB), single owner, no Solidtime import loaded yet.

Commands:

```bash
make resources SAMPLE_SECONDS=120
SAMPLE_SECONDS=300 WITH_LOAD=1 make resources
```

| Scenario | Sample window | Avg RAM | Peak RAM | Avg CPU | Peak CPU | PIDs |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Idle | 120s / 5s (20 samples) | 21.9 MiB | 22.6 MiB | 0.00% | 0.00% | 13 |
| k6 load (10 VUs, 30s) | 300s / 5s (50 samples) | 22.4 MiB | 24.8 MiB | 0.14% | 1.84% | 13 |

Prometheus snapshot after idle run:

| Metric | Value |
| --- | ---: |
| `process_resident_memory_bytes` | 28.4 MiB |
| `go_memstats_heap_inuse_bytes` | 2.7 MiB |
| `go_goroutines` | 10 |

Comparison against the Solidtime VPS snapshot documented above:

| Stack | Containers | Peak RAM |
| --- | ---: | ---: |
| Solidtime (queue + scheduler + app + database) | 4 | ~817 MiB |
| leotime (this baseline) | 1 | ~25 MiB |

That is roughly **33× less peak RAM** in this empty-stack scenario. It predates the in-process email scheduler; expect a small RAM/CPU increase with background ticks enabled.

Re-run after importing a representative Solidtime ZIP and during normal daily use (timer running, timesheet edits, report export) before treating these numbers as deployment guidance.

### Email scheduler metrics

When the background scheduler is enabled, scrape:

```text
leotime_still_running_timers_detected_total
leotime_email_outbox_sent_total
leotime_email_outbox_retried_total
leotime_email_outbox_dead_total
leotime_scheduler_scan_errors_total
leotime_scheduler_outbox_errors_total
```

```bash
curl -s http://127.0.0.1:8080/metrics | rg '^leotime_'
```

See `docs/29-email-notifications.md` for configuration and troubleshooting.

### Prometheus metrics useful for memory

```text
process_resident_memory_bytes
go_memstats_heap_inuse_bytes
go_goroutines
```

Scrape locally:

```bash
curl -s http://127.0.0.1:8080/metrics | rg 'process_resident_memory_bytes|go_memstats_heap_inuse_bytes|go_goroutines'
```

## Stress Tests

Stress tests run through Docker with k6, so no host k6 install is required.

```bash
make stress
```

Customize load:

```bash
make stress K6_VUS=25 K6_DURATION=2m
```

The default thresholds fail when:

- HTTP failure rate is 1% or higher.
- p95 latency is 500 ms or higher.

## Benchmarks

Run Go benchmarks:

```bash
make bench
```

Import benchmarks should live near the import package and run through this same target.

## Deploy Check

Before deploying:

```bash
make deploy-check
```

This runs:

- Backend tests.
- Frontend tests.
- Frontend build.
- Docker image build.

## Migrations

Migrations are embedded in the Go binary and run at startup. The Makefile includes a dedicated migration target:

```bash
make migrate
```

That target uses the same application boot path and exits after migrations.

## Console UX

Make targets intentionally use clear status lines and icons. These are for human-facing workflows. Machine-readable commands should still return proper exit codes and avoid relying on colored output.

