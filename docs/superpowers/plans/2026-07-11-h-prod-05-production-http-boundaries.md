# H-PROD-05 — Production configuration and HTTP boundary safety

**Date:** 2026-07-11  
**Backlog:** `docs/35-curated-hardening-backlog.md` (H-PROD-05)

## Problem

Invalid env values silently default; log mail prints password-reset tokens; rate
limits trust `X-Forwarded-For` without configuration; metrics accepts query
tokens; backup errors can leak internals; security headers are missing.

## Approach

1. `config.Load` / `FromLookup` return errors for invalid bool/int/duration env values.
2. Production `Validate` requires secure cookies, explicit public base URL, and
   `LEOTIME_MAIL_LOG_ENABLED=true` when `LEOTIME_MAIL_MODE=log`.
3. Log mail always redacts sensitive body content; optional `LEOTIME_MAIL_LOG_BODY`
   logs a redacted body in non-production.
4. `LEOTIME_TRUST_FORWARDED_HEADERS` gates `RealIP` middleware and rate-limit IP keys.
5. Metrics accepts Bearer tokens only with constant-time comparison.
6. Backup handlers log errors with request ID and return generic messages.
7. Add security headers middleware.

## Gates

`make test-api`, `make pre-commit`, `make docker-build`, `make smoke`, `make deploy-check`
