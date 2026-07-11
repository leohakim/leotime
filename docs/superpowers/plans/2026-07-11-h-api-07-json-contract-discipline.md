# H-API-07 — JSON contract discipline and startup errors

**Date:** 2026-07-11  
**Backlog:** `docs/35-curated-hardening-backlog.md` (H-API-07)

## Approach

1. Harden `decodeJSONBody` with `DisallowUnknownFields`, empty-body handling, and
   rejection of trailing JSON values (keep 1 MiB limit).
2. Change `NewRouter` to return `(http.Handler, error)` instead of panicking when
   the billing document store cannot initialize.
3. Audit frontend mutation payloads against backend `json` tags; fix only mismatches.
4. Add `json_body_test.go`, router startup test, and document `invalid_json` rules.

Frontend audit (2026-07-11): mutation helpers in `apps/web/src/lib/api.ts` and
form submitters send only documented camelCase fields for clients, projects,
tasks, tags, timers, time entries, profile, backups, invoices, and auth. Empty
`{}` bodies are only sent to endpoints that do not decode JSON (`/issue`, `/run`,
restore/archive actions, timer stop).

## Gates

`make test-api`, `make test-web`, `make pre-commit`, `make smoke`
