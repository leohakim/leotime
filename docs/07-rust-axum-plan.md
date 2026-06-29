# Rust + Axum Alternative Plan

The chosen MVP stack is Go + SQLite + React. This document keeps a detailed Rust plan so the same product can be built with Rust later or compared honestly.

## Rust Stack

- Runtime: Tokio.
- HTTP framework: Axum.
- Middleware: Tower and tower-http.
- Database: SQLite.
- SQL access: SQLx.
- Migrations: SQLx migrations.
- Auth: argon2 crate for password hashing.
- Sessions: signed HTTP-only cookies plus session table.
- Frontend: same React/Vite app.
- Packaging: Docker multi-stage build.

## Equivalent Layout

```text
apps/api-rust
├── Cargo.toml
├── migrations
└── src
    ├── main.rs
    ├── config.rs
    ├── db.rs
    ├── auth.rs
    ├── http
    │   ├── mod.rs
    │   ├── routes.rs
    │   └── handlers.rs
    └── store
        ├── mod.rs
        ├── users.rs
        └── time_entries.rs
```

## Advantages Over Go

- Stronger type system.
- Very explicit error handling.
- Excellent performance and memory profile.
- Compile-time SQL checking is possible with SQLx.
- Good fit for a future Tauri desktop app.

## Costs Compared With Go

- Slower development for business CRUD.
- More concepts for new contributors.
- Longer compile times.
- More friction around lifetimes, async traits, and error typing.
- Harder for a Django developer to approach casually.

## Rust Implementation Plan

1. Create `apps/api-rust` with Axum, Tokio, SQLx, tracing, serde, thiserror.
2. Port the existing SQLite migrations.
3. Implement config from environment.
4. Implement password hashing with Argon2.
5. Implement session cookies.
6. Implement `/api/health`.
7. Implement `/api/v1/session`, login, logout.
8. Implement clients/projects/tasks/tags/time entries.
9. Implement report queries.
10. Implement invoices and PDF generation.
11. Reuse the same frontend.
12. Add Rust unit and integration tests.
13. Add a Rust Docker target.

## When To Switch To Rust

Switch only if one of these becomes true:

- The Go backend is too resource-heavy, which is unlikely.
- The project becomes a Rust learning project.
- The desktop/Tauri version becomes the primary product.
- The stronger type system becomes more valuable than faster feature work.

Until then, Go is the pragmatic path.

