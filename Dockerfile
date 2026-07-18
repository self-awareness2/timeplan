ARG DOCKER_REGISTRY=docker.io

FROM ${DOCKER_REGISTRY}/library/node:20-bookworm-slim AS web-build
WORKDIR /src/client/web
COPY client/web/package.json client/web/package-lock.json ./
RUN npm ci
COPY client/web/ ./
RUN npm run build

FROM ${DOCKER_REGISTRY}/library/golang:1.22-bookworm AS server-build
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/chrona ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/chrona-backup ./cmd/backup

FROM ${DOCKER_REGISTRY}/library/debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home chrona
WORKDIR /app
COPY --from=server-build /out/chrona /app/chrona
COPY --from=server-build /out/chrona-backup /app/chrona-backup
COPY --from=web-build /src/client/web/dist /app/client/web/dist
RUN mkdir -p /app/data/server && chown -R chrona:chrona /app
USER chrona
ENV CHRONA_ROOT=/app \
    CHRONA_DATA_DIR=/app/data/server \
    CHRONA_DIST_DIR=/app/client/web/dist \
    PORT=8765
EXPOSE 8765
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD curl --fail --silent http://127.0.0.1:8765/healthz || exit 1
CMD ["/app/chrona"]
