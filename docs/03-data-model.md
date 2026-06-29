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

Overlaps are allowed. The UI and reports should warn, not reject.

## Invoices

Invoices are simple but should look official:

- Invoice number.
- Issue date.
- Due date.
- Currency.
- Seller name, tax ID, and address.
- Client name, tax ID, and address.
- Tax lines such as IVA.
- Optional withholding lines such as IRPF.
- Status: draft, issued, paid, cancelled.

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

