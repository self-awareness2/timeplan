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

## Upload a Linux image from Windows

Build a Linux AMD64 image locally, export it as a tar archive, and upload the archive together with the server Compose file. This keeps build tools and source code off the production server.

```powershell
$version = "2.0.0"
docker buildx build --platform linux/amd64 --tag "chrona:$version" --load .
docker save --output ".\build\chrona-$version-linux-amd64.tar" "chrona:$version"
scp ".\build\chrona-$version-linux-amd64.tar" <user>@47.120.36.243:/opt/timeplanner/
scp .\docker-compose.server.yml <user>@47.120.36.243:/opt/timeplanner/
scp .env.example <user>@47.120.36.243:/opt/timeplanner/.env.example
```

If Docker Hub is unreachable, keep the Dockerfile unchanged and select a compatible mirror at build time:

```powershell
docker buildx build --platform linux/amd64 --build-arg DOCKER_REGISTRY=docker.m.daocloud.io --tag "chrona:$version" --load .
```

On the server, create `/opt/timeplanner/.env` from `.env.example` without copying secrets into shell history. For a fresh deployment with no retained users, remove or archive the old `data` directory before starting the container.

```bash
cd /opt/timeplanner
sudo systemctl disable --now timeplanner || true
sudo mv data "data-legacy-$(date +%F-%H%M%S)" 2>/dev/null || true
sudo mkdir -p data/server
sudo chown -R 10001:10001 data
cp .env.example .env
# Edit .env and replace both placeholder secrets before continuing.
docker load --input chrona-2.0.0-linux-amd64.tar
docker compose --env-file .env -f docker-compose.server.yml up -d
docker compose --env-file .env -f docker-compose.server.yml ps
curl --fail http://127.0.0.1:8765/healthz
```

Keep the host Nginx proxy pointed at `http://127.0.0.1:8765/`. When the public application lives at `/timeplan/`, the trailing slash in `proxy_pass http://127.0.0.1:8765/;` strips that prefix before forwarding requests to Chrona.

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
