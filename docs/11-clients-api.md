# Clients API

Clients are the first real CRUD slice in `leotime`. They are authenticated, owner-scoped, and soft-deleted through `archived_at`.

## Routes

All routes require a valid session cookie.

```text
GET    /api/v1/clients
POST   /api/v1/clients
GET    /api/v1/clients/{clientID}
PATCH  /api/v1/clients/{clientID}
DELETE /api/v1/clients/{clientID}
POST   /api/v1/clients/{clientID}/restore
```

List archived clients:

```text
GET /api/v1/clients?includeArchived=true
```

## Request Body

Create and update use the same body:

```json
{
  "name": "Client One",
  "email": "billing@example.com",
  "taxId": "B12345678",
  "billingAddress": "Madrid",
  "defaultCurrency": "EUR",
  "defaultHourlyRateMinor": 7500
}
```

## Validation

- `name` is required.
- `email` is optional, but must be valid when present.
- `defaultCurrency` defaults to `EUR` when empty.
- `defaultCurrency` must be a 3-letter uppercase currency code after normalization.
- `defaultHourlyRateMinor` must be non-negative.
- Optional text fields are trimmed and stored as null when empty.

## Delete Behavior

`DELETE` archives the client by setting `archived_at`. This keeps imported and historical reporting data stable while hiding the client from the default list.

`POST /restore` clears `archived_at` and returns the restored client.

## Frontend

The dashboard includes a clients workbench with a directory on the left and an editor on the right. It can:

- List active and archived clients (`includeArchived=true`).
- Create clients.
- Edit clients.
- Archive clients from the list or by unchecking **Active client** in the edit form.
- Reactivate archived clients from the edit form.

The form validates before submit:

- Required name with at least 2 characters.
- Optional email format.
- Required 3-letter currency code.
- Optional hourly rate with up to 2 decimals.

The UI shows hourly rates as human amounts, for example `75.00`. The API still stores money as minor units, so the frontend sends `7500`.

The panel invalidates the clients and overview queries after mutations so counters stay aligned.
