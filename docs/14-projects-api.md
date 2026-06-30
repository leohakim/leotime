# Projects API

Projects are the second CRUD slice in `leotime`. They are authenticated, owner-scoped, and soft-deleted through `archived_at`.

## Routes

All routes require a valid session cookie.

```text
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/{projectID}
PATCH  /api/v1/projects/{projectID}
DELETE /api/v1/projects/{projectID}
```

List archived projects:

```text
GET /api/v1/projects?includeArchived=true
```

Filter by client:

```text
GET /api/v1/projects?clientId=cli_...
```

## Request Body

Create and update use the same body:

```json
{
  "clientId": "cli_...",
  "name": "Website Redesign",
  "color": "#2563eb",
  "defaultHourlyRateMinor": 7500
}
```

`clientId` can be empty when the project should not belong to a client yet.
`defaultHourlyRateMinor` can be `null` when the project should inherit or avoid a specific rate override.

## Validation

- `name` is required.
- `clientId` is optional, but must reference an active client when present.
- `color` defaults to `#2563eb` when empty.
- `color` must be a hex color like `#2563eb`.
- `defaultHourlyRateMinor` can be null, but must be non-negative when present.

## Delete Behavior

`DELETE` archives the project by setting `archived_at`. This keeps historical time entries and import mappings stable
while hiding the project from the default list.

## Frontend

The dashboard includes a projects workbench that can:

- List active projects.
- Create projects.
- Edit projects.
- Archive projects.
- Assign a project to an active client or leave it unassigned.
- Pick a project color.
- Set an optional project-specific hourly rate.

The UI shows hourly rates as human amounts, for example `75.00`. The API stores money as minor units, so the frontend sends `7500`.

The panel invalidates the projects and overview queries after mutations so counters stay aligned.
