# VPS Deployment

The production path is Docker first.

## Requirements

- A small VPS.
- Docker and Docker Compose.
- A domain pointing to the VPS.
- A reverse proxy such as Caddy or Traefik.
- A backup target for the SQLite database.

## Basic Deployment

Copy the repository to the server and create a production `.env` file:

```bash
cp .env.example .env
```

Change at least:

```text
LEOTIME_BOOTSTRAP_EMAIL
LEOTIME_BOOTSTRAP_PASSWORD
LEOTIME_COOKIE_SECURE=true
LEOTIME_PUBLIC_BASE_URL=https://leotime.example.com
LEOTIME_MAIL_MODE=smtp
LEOTIME_MAIL_FROM=no-reply@your-domain.com
LEOTIME_SMTP_HOST=...
LEOTIME_SMTP_PORT=587
LEOTIME_SMTP_USERNAME=...
LEOTIME_SMTP_PASSWORD=...
```

Still-running timer emails use the in-process scheduler (enabled by default). Full mail and scheduler reference: `docs/29-email-notifications.md`.

Start:

```bash
docker compose up -d --build
```

## Data

The SQLite database lives in the Docker volume mounted at `/data`.

The important files are:

```text
leotime.db
leotime.db-wal
leotime.db-shm
```

Backups must account for SQLite WAL mode. The safest first approach is to stop the container briefly or use SQLite's backup API through a future `leotime backup` command.

## Reverse Proxy

An example Caddy config is in:

```text
deploy/caddy.example
```

## Backup Plan

Initial recommended backup flow:

1. Nightly job pauses or asks the app for a consistent backup.
2. Backup file is compressed.
3. Backup is copied with restic, rclone, or another offsite tool.
4. Restore is tested regularly.

The app should later expose:

- Manual backup download.
- Manual restore upload.
- Scheduled backup status.

## Upgrade Plan

For a one-container deployment:

```bash
git pull
docker compose up -d --build
```

Migrations run automatically on startup.

