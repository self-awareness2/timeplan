# Deployment and upgrades

Chrona is packaged as one container. It serves the compiled web client, API, and SQLite database from the same versioned image. Persistent data lives in `./data`, outside the image.

## First deployment

1. Copy `.env.example` to `.env` and set a long `CHRONA_SECRET`.
2. Set `CHRONA_ALLOWED_ORIGINS` to the HTTPS origins that are permitted to call the API. Do not use a wildcard when browser credentials are enabled.
3. Build and start the service:

```powershell
docker compose up -d --build
```

The health endpoint is available at `/healthz`. Put a TLS reverse proxy such as Caddy, Nginx, or your cloud load balancer in front of the container before exposing it publicly.

## Upgrade procedure

The application records each SQLite migration in `schema_migrations`, so a newer image applies only migrations that have not already run.

```powershell
# Create a consistent SQLite snapshot first.
docker compose run --rm chrona /app/chrona-backup

# Build or pull the new image, then replace the running container.
docker compose up -d --build
```

Backups are written to `data/server/backups`. Keep this directory in your normal server backup policy. To restore, stop the container, replace `data/server/chrona.sqlite` with a backup copy, and start it again.

## Native process deployment

When not using Docker, build the web app before the Go service and set these environment variables:

```text
CHRONA_ENV=production
CHRONA_SECRET=<long-random-value>
CHRONA_DATA_DIR=/srv/chrona/data
CHRONA_BACKUP_DIR=/srv/chrona/backups
CHRONA_DIST_DIR=/srv/chrona/web-dist
CHRONA_ALLOWED_ORIGINS=https://chrona.example.com
```

Run `chrona-backup` before replacing the server binary. The migration system runs during normal server startup.
