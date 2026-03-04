# ---- Build Stage ----
FROM node:20-slim AS frontend

WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm run build

# ---- Go Build Stage ----
FROM golang:1.23-bookworm AS backend

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Inject built frontend into embed directory
COPY --from=frontend /app/frontend/build cmd/updu/frontend/build/

ARG BUILD_TAGS=""
RUN CGO_ENABLED=1 go build -tags "${BUILD_TAGS}" -o /updu ./cmd/updu

# ---- Runtime Stage ----
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates iputils-ping \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend /updu /usr/local/bin/updu

RUN mkdir -p /data

ENV UPDU_DB_PATH=/data/updu.db
ENV UPDU_HOST=0.0.0.0
ENV UPDU_PORT=3000

EXPOSE 3000

VOLUME ["/data"]

ENTRYPOINT ["updu"]
