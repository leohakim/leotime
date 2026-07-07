# Password Reset

leotime supports self-service password reset through the same SQLite email outbox used for timer notifications.

## Flow

1. User opens the login screen and chooses **Forgot your password?**
2. Frontend calls `POST /api/v1/auth/forgot-password` with `{ "email": "..." }`
3. API always returns `204 No Content` to avoid email enumeration
4. When the user exists, the API:
   - invalidates previous unused reset tokens for that user
   - stores a hashed one-time token in `password_reset_tokens`
   - enqueues an outbox row with kind `password_reset`
5. The background outbox loop sends the email through SMTP or log mode
6. The email link opens `#reset-password?token=...` in the SPA
7. User submits a new password via `POST /api/v1/auth/reset-password`
8. API updates the password, marks the token used, and clears all sessions for that user

## Configuration

Uses the same mail settings as `docs/29-email-notifications.md`, plus:

| Variable | Default | Description |
| --- | --- | --- |
| `LEOTIME_PASSWORD_RESET_TTL` | `1h` | Reset link lifetime |

Set `LEOTIME_PUBLIC_BASE_URL` to the URL users open in the browser so the email link is correct.

Example link format:

```text
https://leotime.example.com#reset-password?token=...
```

## API

| Method | Path | Auth | Body |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/forgot-password` | Public | `{ "email": "user@example.com" }` |
| `POST` | `/api/v1/auth/reset-password` | Public | `{ "token": "...", "newPassword": "..." }` |

Password rules match profile password change: minimum 8 characters.

## Data model

Migration `000006_password_reset.sql` adds:

- `password_reset_tokens`
- outbox kind `password_reset`

## Manual verification

1. Start the stack with `LEOTIME_MAIL_MODE=log`
2. Open the login screen and request a reset for a known user
3. Check logs for the reset link body
4. Open the link, set a new password, and sign in

## Code map

| Area | Location |
| --- | --- |
| Token store | `apps/api/internal/store/password_reset.go` |
| Email enqueue | `apps/api/internal/notify/password_reset.go` |
| HTTP handlers | `apps/api/internal/httpapi/password_reset.go` |
| Login UI | `apps/web/src/lib/authUi.tsx` |
