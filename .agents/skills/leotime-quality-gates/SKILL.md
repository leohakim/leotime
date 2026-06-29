---
name: leotime-quality-gates
description: Use before finishing leotime work to run or plan backend, frontend, E2E, Docker, smoke, benchmark, and stress verification.
---

# leotime Quality Gates

Use the smallest relevant gate during development and the full gate before delivery.

## Fast Gate

```bash
make test-api
make test-web
```

## Full Gate

```bash
make test
make build-web
make smoke
make bench
```

## Docker Gate

```bash
make docker-build
make up
make smoke
make metrics
```

## Reporting

Summarize:

- Commands run.
- Pass/fail status.
- Important timings.
- Any skipped checks and why.

