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
    "themeMode": "solid",
    "timerStillRunningEnabled": true,
    "timerStillRunningHours": 8,
    "backupEmailOnSuccess": false,
    "backupEmailOnFailure": true,
    "restoreEmailOnSuccess": false,
    "restoreEmailOnFailure": true
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
  "themeMode": "dark",
  "timerStillRunningEnabled": true,
  "timerStillRunningHours": 6,
  "backupEmailOnSuccess": true,
  "backupEmailOnFailure": false,
  "restoreEmailOnSuccess": false,
  "restoreEmailOnFailure": true
}
```

## Change Password Body

```json
{
  "currentPassword": "change-me-now",
  "newPassword": "new-password-123"
}
```

Returns `204 No Content` on success. **All sessions for the user are invalidated** (including the current cookie). The client must log in again with the new password.

## Validation

- `name` is required.
- `email` must be valid and unique.
- `locale` must be `es` or `en`.
- `layoutMode` must be `solid`, `minimal`, or `compact`.
- `themeMode` must be `solid`, `light`, `dark`, or `minimal`.
- `defaultCurrency` must be a 3-letter uppercase code.
- `timezone` must be a valid IANA timezone.
- `timerStillRunningHours` must be between 1 and 24 (defaults to 8 when omitted or zero).
- Password change requires the current password and a new password with at least 8 characters.

## Storage

User fields live in `users`. App preferences live in `app_settings`:

- `task_project_required`
- `default_currency`
- `timezone`
- `theme_mode`
- `timer_still_running_enabled`
- `timer_still_running_hours`
- `backup_email_on_success`
- `backup_email_on_failure`
- `restore_email_on_success`
- `restore_email_on_failure`

Migration `000004_profile_settings.sql` adds profile columns. Migration `000005_email_notifications.sql` adds email outbox and timer notification columns. Migration `000006_password_reset.sql` adds password reset tokens. Migration `000008_backup_email_notifications.sql` adds backup/restore email toggles and extends outbox kinds.

Timer notification behavior: `docs/29-email-notifications.md`. Backup/restore email behavior: `docs/31-s3-daily-backups.md`.

## Frontend

The dashboard includes a profile panel at `#profile` with:

- Account fields: name and email.
- Preferences: language, layout, theme, default currency, timezone, task-project requirement, timer email settings, and backup/restore email toggles.
- Password change form.

On first load, the app hydrates locale, layout, and theme from the saved profile. Saving profile updates the session cache and local UI preferences.

## Sprint 2 experience boundary

Navigation mode and experience preset are local frontend state in Sprint 2.
They intentionally do not appear in profile GET/PATCH requests, profile JSON, or
database storage. The existing `layoutMode` and `themeMode` fields remain the
only profile-synchronized experience dimensions until a later scoped decision.

## Checks

```bash
make test-api
make test-web
make build-web
```

After changing API routes, restart the local API process so the running server picks up the new handlers.
