# Data Model

The schema is designed for one owner first, but it keeps enough structure to support future sync and multi-device work.

## Core Entities

```text
users
sessions
clients
projects
tasks
tags
time_entries
time_entry_tags
rates
invoices
invoice_lines
invoice_series
billing_documents
app_settings
email_outbox
```

## Ownership Rules

- A `client` belongs to a user.
- A `project` usually belongs to a client, but the schema can allow null client IDs later if needed.
- A `task` usually belongs to a project. The current product default should encourage that structure.
- A `time_entry` can reference client, project, task, and many tags.
- A `rate` can be client-level or project-level.
- An invoice belongs to a client and contains frozen invoice lines.

## Time Entries

Time entries should store:

- Start timestamp.
- End timestamp, nullable while a timer is running.
- Duration seconds for finalized reporting.
- Description.
- Billable flag.
- Currency snapshot when relevant.
- Sync metadata.
- `still_active_email_sent_at`, nullable; set after a successful still-running timer email.

Overlaps are allowed. The UI and reports should warn, not reject.

## Email Outbox

`email_outbox` stores durable outbound mail jobs processed by the in-process scheduler:

- One pending/sent row per `(kind, time_entry_id)` for timer notifications.
- Additional kinds such as `password_reset` use `time_entry_id = NULL`.
- Status: `pending`, `sent`, or `dead`.
- Retry metadata: `attempts`, `next_retry_at`, `last_error`.

See `docs/29-email-notifications.md` and `docs/30-password-reset.md`.

## Password Reset Tokens

`password_reset_tokens` stores hashed one-time reset tokens:

- Linked to `users.id`
- Expires after `LEOTIME_PASSWORD_RESET_TTL` (default 1 hour)
- Marked with `used_at` after a successful reset
- Successful reset clears all active sessions for that user

## App Settings

Per-user preferences in `app_settings` include:

- `timer_still_running_enabled` (default on)
- `timer_still_running_hours` (default 8, editable in Profile Settings)

## Invoices

Invoices are simple but should look official:

- Invoice number (draft reference until issue; official number from fiscal series on issue).
- Fiscal series (`series_id`, `fiscal_sequence`).
- Billing period (`period_from`, `period_to`).
- Issue date.
- Due date.
- Currency.
- Seller name, tax ID, and address.
- Client name, tax ID, and address.
- Tax lines such as IVA.
- Optional withholding lines such as IRPF.
- Status: draft, issued, paid, cancelled.
- Frozen `document_snapshot_json` after issue.
- `work_protocol_detail`: `summary`, `standard`, or `detailed`.
- Cancellation metadata (`cancelled_at`, `cancellation_reason`).

### Invoice series

`invoice_series` stores configurable fiscal numbering per user:

- `code`, `name`, `pattern`, `next_sequence`, `reset_policy`, `active`, `is_default`.

### Billing documents

`billing_documents` stores immutable PDF metadata per issued invoice:

- `kind`: `invoice_pdf` or `work_protocol_pdf`
- `storage_path` relative to `LEOTIME_DOCUMENT_ROOT`
- `sha256`, `byte_size`, `mime_type`, `render_version`

The MVP does not promise legal compliance. It gives a professional-looking document that the owner can adjust.

## Money

Store money as integer minor units:

```text
EUR 123.45 -> 12345
USD 123.45 -> 12345
JPY 123    -> 123
```

Store currency as ISO-style uppercase code such as `EUR`, `USD`, `GBP`, or `ARS`.

## Sync Metadata

Tables that can be created offline should eventually include:

- `client_uuid`
- `created_at`
- `updated_at`
- `deleted_at`
- `sync_version`
- `last_modified_device_id`

The first migration includes the most useful timestamps. The full sync fields can be added once `/api/v1/sync` is implemented.

