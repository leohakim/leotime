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

