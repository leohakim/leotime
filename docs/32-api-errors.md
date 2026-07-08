# API error responses

All JSON error responses use a structured envelope:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "name is required",
    "fields": [
      {
        "field": "name",
        "code": "required",
        "message": "name is required"
      }
    ]
  }
}
```

## Fields

| Property | Meaning |
| --- | --- |
| `code` | Stable machine-readable identifier (`validation_failed`, `client_not_found`, `invalid_json`, …) |
| `message` | Human-readable summary (English, server-side) |
| `fields` | Optional list of field-level validation issues |

Each field entry includes:

- `field` — input property name (`name`, `email`, `defaultCurrency`, …)
- `code` — validation kind (`required`, `invalid`, `duplicate`)
- `message` — detail for that field

## Validation errors

Store validation returns `validation_failed` with one `fields` entry. Examples:

| Endpoint | Field | Code | When |
| --- | --- | --- | --- |
| `POST /api/v1/clients` | `name` | `required` | Empty name |
| `POST /api/v1/clients` | `email` | `invalid` | Bad email format |
| `POST /api/v1/tags` | `name` | `duplicate` | Tag name already exists |

## Domain errors

Resource and auth errors omit `fields`:

```json
{ "error": { "code": "client_not_found", "message": "client not found" } }
```

Common codes: `invalid_json`, `authentication_required`, `invalid_credentials`, `email_taken`, `backup_busy`, `backup_secrets_key_missing`.

## Frontend

`apps/web/src/lib/api.ts` exposes `ApiError` with `code`, `status`, `message`, and `fields`. Mutations that use `apiJSON` throw `ApiError` on failure. Client-side forms still validate locally; server `fields` are available for future i18n mapping.
