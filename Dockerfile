# syntax=docker/dockerfile:1.7

# ---- Frontend build stage ----
FROM node:20-slim AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm run build

# ---- Go build stage ----
FROM golang:1.26.1-bookworm AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/build cmd/updu/frontend/build/

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
ARG BUILD_TAGS=""

RUN set -eux; \
    LDFLAGS="-X github.com/updu/updu/internal/version.Version=${VERSION} \
             -X github.com/updu/updu/internal/version.GitCommit=${COMMIT} \
             -X github.com/updu/updu/internal/version.BuildDate=${BUILD_DATE}"; \
    if [ -n "${BUILD_TAGS}" ]; then \
        LDFLAGS="${LDFLAGS} -X github.com/updu/updu/internal/version.BuildTags=${BUILD_TAGS}"; \
        CGO_ENABLED=0 go build -tags "${BUILD_TAGS}" -ldflags "${LDFLAGS}" -o /updu ./cmd/updu; \
    else \
        CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o /updu ./cmd/updu; \
    fi

# ---- Runtime stage ----
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates iputils-ping wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=backend /updu /usr/local/bin/updu
RUN mkdir -p /data

ENV UPDU_DB_PATH=/data/updu.db \
    UPDU_HOST=0.0.0.0 \
    UPDU_PORT=3000

EXPOSE 3000
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["wget", "--no-verbose", "--spider", "http://localhost:3000/healthz"] || exit 1

ENTRYPOINT ["updu"]
