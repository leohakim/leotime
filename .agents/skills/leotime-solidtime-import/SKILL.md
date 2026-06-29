---
name: leotime-solidtime-import
description: Use when working on Solidtime ZIP export compatibility, import parsing, import validation, idempotency, external mappings, or import tests for leotime.
---

# leotime Solidtime Import

Follow this workflow whenever the task touches Solidtime exports.

## Inputs

- A Solidtime ZIP export path.
- Synthetic fixtures under the repo when writing tests.
- Current import docs in `docs/09-solidtime-import.md` when present.

## Workflow

1. Inspect ZIP metadata without extracting personal data into the repo.
2. Validate expected files and CSV headers.
3. Map Solidtime UUIDs through `external_mappings`.
4. Import in dependency order: organization, members, clients, projects, tasks, tags, time entries.
5. Preserve timestamps when possible.
6. Treat blank billable rates as null/default values.
7. Parse tags as JSON text from the CSV `tags` column.
8. Allow overlapping time entries and mark warnings instead of rejecting them.
9. Add or update synthetic fixtures for tests.
10. Run import parser and idempotency tests.

## Output Expectations

- State whether the import is dry-run or write mode.
- Report created, updated, skipped, warning, and error counts.
- Never include personal export contents in final output.

