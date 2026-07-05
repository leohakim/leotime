---
name: leotime-quality-gates
description: Use before finishing leotime work to run or plan backend, frontend, E2E, Docker, smoke, benchmark, and stress verification.
---

# leotime Quality Gates

Use the smallest relevant gate during development and the full gate before delivery.

## Pre-Commit Gate (Required Before Handoff)

Before finishing any code change, run:

```bash
make pre-commit
```

This matches the repository git hook. It runs `gofmt`, `go vet`, backend tests, frontend tests, and the web production build.

If it fails, fix the reported issues and rerun until it passes. Do not mark work complete while this gate is red.

## Fast Gate

```bash
make test-api
make test-web
```

Use this for quick iteration while editing; still run `make pre-commit` before final delivery.

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

