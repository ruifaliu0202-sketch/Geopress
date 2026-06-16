# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/geopress-api ./cmd/api

FROM node:26-bookworm-slim AS runtime

ENV APP_ENV=production \
    HTTP_ADDR=:18080 \
    GEOPRESS_PROJECT_ROOT=/app \
    GEOPRESS_NODE_BIN=/usr/local/bin/node \
    GEOPRESS_CHROME_PATH=/usr/bin/chromium \
    GEOPRESS_BROWSER_HEADLESS=true \
    GEOPRESS_CHROMIUM_NO_SANDBOX=true \
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        chromium \
        fonts-noto-cjk \
        fonts-noto-color-emoji \
        postgresql-client \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/geopress-api /app/geopress-api
COPY scripts/ /app/scripts/
COPY backend/migrations/ /app/backend/migrations/
COPY frontend/package.json /tmp/frontend-package.json

RUN node -p "require('/tmp/frontend-package.json').devDependencies.playwright.replace(/^[^0-9]*/, '')" > /tmp/playwright-version \
    && npm init -y >/dev/null 2>&1 \
    && npm install --no-audit --no-fund "playwright@$(cat /tmp/playwright-version)" \
    && rm -f /tmp/frontend-package.json /tmp/playwright-version \
    && mkdir -p /app/runtime \
    && chown -R node:node /app

USER node

EXPOSE 18080

CMD ["/app/geopress-api"]
