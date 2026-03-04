.PHONY: all build run test clean dev

BINARY_NAME=updu
FRONTEND_DIR=frontend
GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -X github.com/updu/updu/internal/version.Version=$(VERSION) \
          -X github.com/updu/updu/internal/version.GitCommit=$(COMMIT) \
          -X github.com/updu/updu/internal/version.BuildDate=$(BUILD_DATE)

all: build

# Generate Tailwind, build SvelteKit, then build Go
build: build-frontend
	@echo "Building Go backend ($(VERSION))..."
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/updu

build-arm: build-frontend
	@echo "Building Go backend for ARMv6 (Raspberry Pi Zero W)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-arm ./cmd/updu

build-arm-oidc: build-frontend
	@echo "Building Go backend for ARMv6 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build -tags oidc -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-arm-oidc ./cmd/updu

build-arm64: build-frontend
	@echo "Building Go backend for ARM64 (Raspberry Pi 3/4/5, AWS Graviton)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-arm64 ./cmd/updu

build-arm64-oidc: build-frontend
	@echo "Building Go backend for ARM64 with OIDC support..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -tags oidc -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-arm64-oidc ./cmd/updu

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

clean:
	$(GO) clean
	rm -rf bin/
	rm -rf cmd/updu/frontend/build
