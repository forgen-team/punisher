# syntax=docker/dockerfile:1
# Punisher as a Witems app (PUNISHER_MODE=witems; docs/WITEMS-APP.md).
# The standalone app is a single binary (make build) and does not use this image.
#   NODE_AUTH_TOKEN=$(gh auth token) docker build --secret id=npm_token,env=NODE_AUTH_TOKEN \
#     -t punisher:dev --build-arg VERSION=$(git describe --tags --always) .
# @forgen-team/* packages come from GitHub Packages (web/.npmrc): the token is a BuildKit secret, never a layer.

# 1) Module Federation remote → internal/witems/dist
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN --mount=type=secret,id=npm_token NODE_AUTH_TOKEN=$(cat /run/secrets/npm_token) npm ci --no-audit --no-fund
COPY web/ ./
RUN mkdir -p ../internal/witems/dist && npm run build:witems \
 && if ls ../internal/witems/dist/assets/*.css >/dev/null 2>&1; then echo "stray CSS chunks (see Makefile witems-web)"; exit 1; fi

# 2) Go binaries (pure Go SQLite, no CGO)
FROM golang:1.26-alpine AS go
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/witems/dist ./internal/witems/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/punisher ./cmd/punisher \
 && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/punisher-witems-import ./cmd/punisher-witems-import

# 3) runtime
FROM alpine:3.22
RUN apk add --no-cache tzdata ca-certificates \
 && adduser -D -H -u 10001 punisher \
 && mkdir -p /data && chown punisher /data
COPY --from=go /out/punisher /out/punisher-witems-import /usr/local/bin/
ENV PUNISHER_MODE=witems \
    PUNISHER_DATA_DIR=/data \
    PUNISHER_LISTEN=0.0.0.0:8080 \
    PORT=8080 \
    HOSTNAME=0.0.0.0 \
    TZ=Asia/Seoul
USER punisher
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["punisher"]
