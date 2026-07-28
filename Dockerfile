FROM golang:1.27.1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/yaplane . && \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/seed-demo ./cmd/seed-demo && \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 yaplane \
    && useradd --uid 10001 --gid yaplane --no-create-home yaplane \
    && mkdir /data && chown yaplane:yaplane /data
WORKDIR /app
COPY --from=build /out/yaplane /app/yaplane
COPY --from=build /out/seed-demo /app/seed-demo
COPY --from=build /out/admin /app/admin
COPY frontend /app/frontend
ENV PORT=8080 DATABASE_PATH=/data/forum.db
USER yaplane
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD if [ -n "$TLS_CERT_FILE" ]; then curl --fail --silent --cacert "$TLS_CERT_FILE" "https://localhost:$PORT/healthz"; else curl --fail --silent "http://127.0.0.1:$PORT/healthz"; fi || exit 1
CMD ["/app/yaplane"]
