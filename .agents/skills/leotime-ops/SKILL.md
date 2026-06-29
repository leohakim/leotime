---
name: leotime-ops
description: Use for Docker-first development, VPS deployment, Makefile commands, metrics, performance testing, backups, and operational scripts in leotime.
---

# leotime Operations

Operational work should keep local development and VPS deployment simple.

## Expectations

- Prefer Docker Compose for full-stack workflows.
- Keep Make targets discoverable through `make help`.
- Keep commands scriptable with environment variables.
- Use friendly console output for human workflows.
- Keep destructive operations explicit.

## Required Operational Surfaces

- `make setup`
- `make dev`
- `make up`
- `make down`
- `make logs`
- `make test`
- `make smoke`
- `make bench`
- `make stress`
- `make metrics`
- `make deploy-check`

## Observability

Prometheus metrics should be available at `/metrics`.
Grafana and Prometheus should run behind an optional Compose profile, not as default required services.

