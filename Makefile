.PHONY: all build build-oidc build-amd64 build-amd64-oidc build-arm build-arm-oidc build-armv7 build-armv7-oidc build-arm64 build-arm64-oidc build-all build-frontend run test clean dev dev-backend dev-frontend e2e-frontend test-e2e-update

BINARY_NAME=updu
FRONTEND_DIR=frontend
GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
VERSION_BASE ?= v0.5.1
RAW_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
WORKTREE_DIRTY ?= $(shell if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then if git diff --quiet --ignore-submodules HEAD >/dev/null 2>&1; then printf ''; else printf -- '-dirty'; fi; fi)
COMMIT ?= $(RAW_COMMIT)$(WORKTREE_DIRTY)
VERSION ?= $(VERSION_BASE)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -X github.com/updu/updu/internal/version.Version=$(VERSION) \
          -X github.com/updu/updu/internal/version.GitCommit=$(COMMIT) \
          -X github.com/updu/updu/internal/version.BuildDate=$(BUILD_DATE)

all: build

# Generate Tailwind, build SvelteKit, then build Go
build: build-frontend
	@echo "Building Go backend ($(VERSION))..."
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/updu

build-oidc: build-frontend
	@echo "Building Go backend with OIDC ($(VERSION))..."
	CGO_ENABLED=0 $(GO) build -tags oidc -ldflags "$(LDFLAGS) -X github.com/updu/updu/internal/version.BuildTags=oidc" -o bin/$(BINARY_NAME)-oidc ./cmd/updu

build-amd64: build-frontend
	@echo "Building Go backend for AMD64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/updu

build-amd64-oidc: build-frontend
	@echo "Building Go backend for AMD64 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -tags oidc -ldflags "$(LDFLAGS) -X github.com/updu/updu/internal/version.BuildTags=oidc" -o bin/$(BINARY_NAME)-linux-amd64-oidc ./cmd/updu

build-arm: build-frontend
	@echo "Building Go backend for ARMv6 (Raspberry Pi Zero W)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-armv6 ./cmd/updu

build-arm-oidc: build-frontend
	@echo "Building Go backend for ARMv6 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build -tags oidc -ldflags "$(LDFLAGS) -X github.com/updu/updu/internal/version.BuildTags=oidc" -o bin/$(BINARY_NAME)-linux-armv6-oidc ./cmd/updu

build-armv7: build-frontend
	@echo "Building Go backend for ARMv7 (Raspberry Pi 2/3)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-armv7 ./cmd/updu

build-armv7-oidc: build-frontend
	@echo "Building Go backend for ARMv7 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build -tags oidc -ldflags "$(LDFLAGS) -X github.com/updu/updu/internal/version.BuildTags=oidc" -o bin/$(BINARY_NAME)-linux-armv7-oidc ./cmd/updu

build-arm64: build-frontend
	@echo "Building Go backend for ARM64 (Raspberry Pi 3/4/5, AWS Graviton)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/updu

build-arm64-oidc: build-frontend
	@echo "Building Go backend for ARM64 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -tags oidc -ldflags "$(LDFLAGS) -X github.com/updu/updu/internal/version.BuildTags=oidc" -o bin/$(BINARY_NAME)-linux-arm64-oidc ./cmd/updu

build-all: build-amd64 build-amd64-oidc build-arm build-arm-oidc build-armv7 build-armv7-oidc build-arm64 build-arm64-oidc
	@echo "All platform builds complete."

build-frontend:
	@echo "Building SvelteKit frontend..."
	cd frontend && pnpm run build
	@echo "Syncing frontend build to embed directory..."
	rm -rf cmd/updu/frontend/build
	cp -r frontend/build cmd/updu/frontend/build

run: build
	./bin/$(BINARY_NAME)

dev-backend:
	$(GO) run ./cmd/updu

dev-frontend:
	cd $(FRONTEND_DIR) && pnpm run dev

test:
	$(GO) test -v ./...

e2e-frontend:
	cd $(FRONTEND_DIR) && pnpm run test:e2e

# Run E2E self-update test
test-e2e-update:
	@echo "Running self-update E2E test..."
	$(GO) test -v -tags=e2e -timeout=5m ./internal/updater

clean:
	$(GO) clean
	rm -rf bin/
	rm -rf cmd/updu/frontend/build
