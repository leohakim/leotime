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
LEOTIME_SECRETS_KEY=...          # openssl rand -base64 32
LEOTIME_BACKUP_SCHEDULER_ENABLED=true
```

Still-running timer emails use the in-process scheduler (enabled by default). Full mail and scheduler reference: `docs/29-email-notifications.md`.

Daily S3 backups use the same scheduler process. Configure the bucket in **Settings → Backups** after deploy, or use the CLI. Full reference: `docs/31-s3-daily-backups.md`.

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

Backups use the SQLite backup API (safe with WAL mode). The app creates a gzip tar archive with the database, official invoice PDFs under `LEOTIME_DOCUMENT_ROOT`, and a `manifest.json` integrity manifest, then uploads it to your private S3 bucket once per day (default **01:00** in your profile timezone, **365-day** retention).

## Reverse Proxy

An example Caddy config is in:

```text
deploy/caddy.example
```

## Backup Plan

Recommended production flow:

1. Set `LEOTIME_SECRETS_KEY` in `.env` (back up this key separately from S3).
2. Open **Settings → Backups** and enter S3 credentials (AWS or S3-compatible endpoint).
3. Click **Test connection**, then **Run now** to verify the first upload.
4. Leave automatic backup enabled (default schedule: **01:00** local time, **365 days** retention in S3).
5. Test restore on a staging copy before you need it in production.

### CLI (same container)

```bash
docker compose exec leotime /app/leotime backup run
docker compose exec leotime /app/leotime backup list
docker compose exec leotime /app/leotime backup restore --latest --force
```

### Cold recovery (UI unavailable)

If the app will not start:

1. Download the latest `leotime-*.db.gz` from S3.
2. Stop the container: `docker compose down`.
3. Decompress and replace `/data/leotime.db` in the `leotime-data` volume.
4. Remove stale WAL files if present: `leotime.db-wal`, `leotime.db-shm`.
5. Start again: `docker compose up -d`.

Prefer in-app restore (`POST /api/v1/backups/restore` or Settings UI) when the server is healthy; it creates a local safety snapshot before replacing data. Full `.tar.gz` restores stage and validate invoice PDFs before promotion; failed restores keep maintenance mode active until restart.

Full API, metrics, and provider examples: `docs/31-s3-daily-backups.md`.

## Upgrade Plan

For a one-container deployment:

```bash
git pull
docker compose up -d --build
```

Migrations run automatically on startup.

