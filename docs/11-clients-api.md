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
- `defaultCurrency` defaults to `EUR` when empty.
- `defaultCurrency` must be a 3-letter uppercase currency code after normalization.
- `defaultHourlyRateMinor` must be non-negative.
- Optional text fields are trimmed and stored as null when empty.

## Delete Behavior

`DELETE` archives the client by setting `archived_at`. This keeps imported and historical reporting data stable while hiding the client from the default list.

## Frontend

The dashboard includes a clients panel that can:

- List active clients.
- Create clients.
- Edit clients.
- Archive clients.

The panel invalidates the clients and overview queries after mutations so counters stay aligned.

