# ADR 0002: In-Process Scheduler and SQLite Email Outbox

## Status

Accepted.

## Context

Solidtime self-hosting runs separate scheduler and queue worker containers for timed jobs and outbound email. leotime targets a single Docker container, low RAM, and one owner first.

We still need:

- Periodic scans for long-running timers.
- Durable outbound email with retries after SMTP failures.
- Parity with Solidtime's `still_active_email_sent_at` behavior.

## Decision

Run background work inside the main `leotime` process:

- A scheduler goroutine with two tickers (scan + outbox processing).
- A SQLite `email_outbox` table for durable, deduplicated mail jobs.
- `net/smtp` or log mode for delivery.
- Exponential backoff retries until success or `dead` status.

Mark `still_active_email_sent_at` only after successful delivery, not when enqueueing.

## Consequences

Good:

- No extra containers or Redis/Postgres queue.
- Retries survive process restarts.
- Footprint stays near the single-container baseline.

Tradeoffs:

- Email sending shares the same process and SQLite writer as HTTP.
- No horizontal scaling of workers without redesign.
- Password reset and other mail types still need to reuse this outbox.

## Alternatives Considered

Separate scheduler container (Solidtime-style):

- Familiar ops model, but higher RAM and SQLite multi-process complexity.

In-memory retries only:

- Simpler code, but loses mail jobs on restart.

External transactional API (Resend, Postmark):

- Useful later; SMTP keeps parity with Solidtime docs and generic VPS hosting.
