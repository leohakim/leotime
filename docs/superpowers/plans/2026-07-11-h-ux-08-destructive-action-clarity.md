# H-UX-08 — Destructive-action clarity and focused maintenance

**Date:** 2026-07-11  
**Backlog:** `docs/35-curated-hardening-backlog.md` (H-UX-08)

## Approach

1. Add `destructiveUi.ts` with a small `confirmDestructiveAction` helper and reuse it
   from archive, permanent-delete, and invoice-cancel flows.
2. Add i18n keys that distinguish archive (reversible) from permanent delete and
   tighten backup-restore confirmation copy.
3. Detect `maintenance_mode` API errors in `QueryErrorBanner` and the session boot
   screen; show focused copy and a reload action instead of generic retry.
4. Add Vitest coverage for the helper and maintenance detection.

## Gates

`make test-web`, `make pre-commit`, `make smoke`
