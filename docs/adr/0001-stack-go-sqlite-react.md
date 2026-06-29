# ADR 0001: Go + SQLite + React

## Status

Accepted.

## Context

The product should be:

- Lightweight.
- Easy to deploy on a VPS.
- Good for one owner first.
- Friendly to contributors coming from Python/Django.
- Test-heavy.
- Capable of offline work.
- Able to generate reports and invoices.

## Decision

Use:

- Go for the backend.
- SQLite for persistence.
- React + Vite + TypeScript for the frontend.
- Docker Compose as the primary deployment path.

## Consequences

Good:

- Simple production runtime.
- Fast tests.
- Easy deployment.
- SQLite backup/restore is understandable.
- React ecosystem helps with calendars, tables, forms, and tests.
- Go is approachable for developers coming from Python.

Tradeoffs:

- Go has less compile-time strictness than Rust.
- SQLite needs careful backup handling in WAL mode.
- Offline sync still needs deliberate conflict handling.
- React adds more frontend dependency surface than Svelte.

## Alternatives Considered

Rust + Axum:

- Stronger correctness model and very high performance.
- Slower initial development and steeper learning curve.

Bun + Hono:

- Very fast TypeScript iteration.
- Less conservative for a long-lived VPS service.

Next.js:

- Productive fullstack framework.
- More runtime weight and deployment complexity than needed here.

