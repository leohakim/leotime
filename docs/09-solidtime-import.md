# Solidtime Import Compatibility

`leotime` supports a first import path for Solidtime ZIP exports. The import is CLI-first so compatibility can be validated with tests before adding a browser upload flow.

## Command

Dry-run:

```bash
make import-solidtime-dry ZIP=/path/to/solidtime-export.zip USER_EMAIL=admin@example.com
```

Write import:

```bash
make import-solidtime ZIP=/path/to/solidtime-export.zip USER_EMAIL=admin@example.com
```

Direct binary usage:

```bash
cd apps/api
go run ./cmd/leotime import solidtime --file /path/to/export.zip --user-email admin@example.com --dry-run
```

The command prints a JSON summary with created, updated, skipped, warning, and error counts.

## Supported Export Shape

The first importer supports Solidtime export `version: 1.0` with these files:

- `meta.json`
- `organizations.csv`
- `organization_invitations.csv`
- `members.csv`
- `clients.csv`
- `projects.csv`
- `project_members.csv`
- `tasks.csv`
- `tags.csv`
- `time_entries.csv`

The importer validates expected headers. Unknown or missing required files fail the import before database writes.

## Mapping

| Solidtime | leotime |
| --- | --- |
| organization | current importing user ownership and default currency |
| member | current importing user |
| client | `clients` |
| project | `projects` |
| task | `tasks` |
| tag | `tags` |
| time entry | `time_entries` |
| time entry tags | `time_entry_tags` |

Solidtime UUIDs are stored in `external_mappings` using:

```text
provider = solidtime
external_type
external_id
internal_type
internal_id
```

This makes imports idempotent: running the same ZIP again updates existing mapped rows instead of duplicating them.

## Field Rules

- Timestamps are parsed as UTC RFC3339/RFC3339Nano.
- Booleans are parsed from `true`, `false`, `1`, and `0`.
- Empty billable rates become null/default values.
- Decimal billable rates are converted to minor units by multiplying by 100.
- Tags in `time_entries.csv` are parsed as JSON text.
- Empty client/project/task references are allowed on time entries.
- Unknown non-empty references fail validation.
- Overlapping time entries are allowed and recorded as warnings.

## Data Safety

Real Solidtime export ZIP files can contain personal client names, project names, task descriptions, email addresses, and work history. Do not commit real exports to the repository.

Tests use synthetic ZIP fixtures generated in memory.

## Current Limitations

- Import is CLI/service only; no browser upload UI yet.
- Solidtime organization invitations and project members are validated for file shape but not imported into first-class tables.
- The importer targets the current single-owner model.
- Failed import runs are not persisted yet because imports run inside one transaction.

