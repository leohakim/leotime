# H-BACKUP-04 — Restore database and documents safely together

**Date:** 2026-07-11  
**Backlog:** `docs/35-curated-hardening-backlog.md` (H-BACKUP-04)

## Problem

`Restore` replaces the live SQLite database, then deletes `LEOTIME_DOCUMENT_ROOT` before copying archived PDFs. A filesystem failure during document replacement can leave restored invoice metadata pointing at missing files.

## Approach

1. Keep existing pre-live validation: `ExtractArchive` + `ValidateManifest`, then `snapshot.ValidateDatabase`.
2. For `.tar.gz` restores, copy archived `documents/` to a **sibling staging tree** (`{documentRoot}.restore-staging`), validate hashes with `ValidateDocumentManifest`, and only then touch live data.
3. Before swapping live data, copy the current document tree to `{documentRoot}.restore-backup`.
4. Restore the database from the validated archive snapshot.
5. Promote staging → live with `Rename` (atomic at directory level on same volume).
6. On promotion failure, roll back the database from the in-temp safety snapshot and restore documents from the backup tree.
7. Legacy `.db.gz` restores skip document staging entirely.
8. Maintenance mode stays active until a paired restore succeeds; failed restores leave maintenance on until process restart or explicit test cleanup.

## Files

- `apps/api/internal/backup/manifest.go` — `ValidateDocumentManifest`
- `apps/api/internal/backup/archive.go` — staging, backup, promote, rollback helpers
- `apps/api/internal/backup/service.go` — orchestration + test hook `promoteDocuments`
- `apps/api/internal/backup/service_test.go` — acceptance tests
- `docs/31-s3-daily-backups.md`, `docs/06-deploy-vps.md`, `docs/adr/0003-s3-backup-encryption-and-restore.md`
- `docs/35-curated-hardening-backlog.md` — mark Done, next H-PROD-05

## Acceptance tests

1. Failed document promotion leaves original DB rows and document hashes readable.
2. Successful archive restore matches manifest document hashes.
3. Legacy `.db.gz` restore does not replace documents.
4. Maintenance mode remains active after failed paired restore.

## Gates

`make test-api`, `make pre-commit`, `make smoke`, `make deploy-check`
