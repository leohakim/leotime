# Solidtime Import Compatibility

`leotime` supports Solidtime ZIP imports through the web app and the CLI.

## Web UI

Open **Import / Export** in the sidebar (`#/import-export`).

1. Choose a Solidtime `version: 1.0` ZIP export.
2. Keep **Validate only (dry-run)** checked to preview counts without writing.
3. Uncheck dry-run and upload again to import into the signed-in account.

The page also exports time data as CSV or JSON for the selected date range.

## HTTP API

```http
POST /api/v1/imports/solidtime?dryRun=true
Content-Type: multipart/form-data

file=<solidtime-export.zip>
```

The response body includes a JSON `summary` with created, updated, skipped, warning, and error counts. Authentication is required.

## CLI

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

The importer validates expected headers and rejects missing required files
before database writes. Each upload is capped at **32 MiB compressed**. Inside
the archive leotime accepts exactly `meta.json` and the nine documented CSV
files, rejects duplicate/unknown/absolute/traversal-like members, and enforces:

- at most **16** ZIP file entries,
- **1 MiB** for `meta.json`,
- **32 MiB** per CSV,
- **128 MiB** total uncompressed.

`import_runs.source_path` stores only the uploaded ZIP basename, never a full
local path.

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
- `still_active_email_sent_at` is imported when present so migrated timers that already received Solidtime mail are not notified again in leotime.
- Empty client/project/task references are allowed on time entries.
- Unknown non-empty references fail validation.
- Overlapping time entries are allowed and recorded as warnings.

## Data Safety

Real Solidtime export ZIP files can contain personal client names, project names, task descriptions, email addresses, and work history. Do not commit real exports to the repository.

Tests use synthetic ZIP fixtures generated in memory.

## Current Limitations

- Solidtime organization invitations and project members are validated for file shape but not imported into first-class tables.
- The importer targets the current single-owner model.
- Failed import runs are not persisted yet because imports run inside one transaction.
- CLI imports currently receive a full local path in the import-run metadata;
  H-IMP-03 changes this to a sanitized basename.
