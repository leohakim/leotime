# Profile Settings API

Profile settings let the single owner update account details, app preferences, and password.

## Routes

All routes require a valid session cookie.

```text
GET  /api/v1/profile
PATCH /api/v1/profile
POST /api/v1/profile/change-password
```

## Profile Response

```json
{
  "id": "usr_...",
  "email": "admin@example.com",
  "name": "Administrador",
  "locale": "es",
  "layoutMode": "solid",
  "settings": {
    "taskProjectRequired": false,
    "defaultCurrency": "EUR",
    "timezone": "Europe/Madrid",
    "themeMode": "solid"
  },
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

## Update Body

Create and update use the same body:

```json
{
  "name": "Leo",
  "email": "leo@example.com",
  "locale": "en",
  "layoutMode": "compact",
  "taskProjectRequired": true,
  "defaultCurrency": "USD",
  "timezone": "America/New_York",
  "themeMode": "dark"
}
```

## Change Password Body

```json
{
  "currentPassword": "change-me-now",
  "newPassword": "new-password-123"
}
```

Returns `204 No Content` on success.

## Validation

- `name` is required.
- `email` must be valid and unique.
- `locale` must be `es` or `en`.
- `layoutMode` must be `solid`, `minimal`, or `compact`.
- `themeMode` must be `solid`, `light`, `dark`, or `minimal`.
- `defaultCurrency` must be a 3-letter uppercase code.
- `timezone` must be a valid IANA timezone.
- Password change requires the current password and a new password with at least 8 characters.

## Storage

User fields live in `users`. App preferences live in `app_settings`:

- `task_project_required`
- `default_currency`
- `timezone`
- `theme_mode`

Migration `000004_profile_settings.sql` adds the new `app_settings` columns.

## Frontend

The dashboard includes a profile panel at `#profile` with:

- Account fields: name and email.
- Preferences: language, layout, theme, default currency, timezone, and task-project requirement.
- Password change form.

On first load, the app hydrates locale, layout, and theme from the saved profile. Saving profile updates the session cache and local UI preferences.

## Checks

```bash
make test-api
make test-web
make build-web
```

After changing API routes, restart the local API process so the running server picks up the new handlers.
