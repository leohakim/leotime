# H-IMP-03: Solidtime ZIP Boundary and Import Privacy

**Status:** Done (2026-07-11)  
**Backlog:** [35-curated-hardening-backlog.md](../../35-curated-hardening-backlog.md#h-imp-03--solidtime-zip-boundary-and-import-privacy)

## Problem

Compressed uploads are capped at 32 MiB, but the parser read every ZIP member fully,
accepted extra members, and import runs stored full local paths.

## Outcomes

- Allow exactly `meta.json` and nine documented CSV files; reject unknown, duplicate,
  absolute, and traversal-like names before parsing.
- Enforce at most 16 ZIP entries, 1 MiB for `meta.json`, 32 MiB per CSV, and
  128 MiB total uncompressed.
- Store only a sanitized basename in `import_runs.source_path`.
- Preserve dry-run and idempotent external mappings.

## Gates

```bash
make test-api
make bench
make import-solidtime-dry ZIP=<synthetic>
make pre-commit
```
