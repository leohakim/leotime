# H-MIG-06 — Upgrade migration confidence

**Date:** 2026-07-11  
**Backlog:** `docs/35-curated-hardening-backlog.md` (H-MIG-06)

## Problem

Migration `000003_tags_archive.sql` rebuilds the `tags` table inside a transaction.
No automated test started from a version-2 database with `time_entry_tags` links.

## Approach

Add `migrate_test.go` coverage that:

1. Applies migrations `000001` and `000002` to a temp SQLite file.
2. Seeds a user, two tags, one time entry, and two `time_entry_tags` rows.
3. Records `schema_migrations` versions 1 and 2 only.
4. Runs `Migrate()` through `000011`.
5. Asserts preserved tag links, `PRAGMA foreign_key_check`, the partial unique
   index on active tags, and recorded migration versions.

The test exposed that `PRAGMA foreign_keys=OFF` inside migration `000003` did not
apply within the runner transaction, so `000003` now backs up and restores
`time_entry_tags` explicitly.

## Gates

`make test-api`, `make pre-commit`
