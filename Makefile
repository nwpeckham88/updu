.PHONY: all build run test clean dev

BINARY_NAME=updu
FRONTEND_DIR=frontend
GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

all: build

# Generate Tailwind, build SvelteKit, then build Go
build: build-frontend
	@echo "Building Go backend..."
	CGO_ENABLED=1 $(GO) build -o bin/$(BINARY_NAME) ./cmd/updu

build-arm: build-frontend
	@echo "Building Go backend for ARMv6 (Raspberry Pi Zero W)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 CC=arm-linux-gnueabihf-gcc $(GO) build -o bin/$(BINARY_NAME)-arm ./cmd/updu

build-arm-oidc: build-frontend
	@echo "Building Go backend for ARMv6 with OIDC support..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 CC=arm-linux-gnueabihf-gcc $(GO) build -tags oidc -o bin/$(BINARY_NAME)-arm-oidc ./cmd/updu

build-arm64: build-frontend
	@echo "Building Go backend for ARM64 (Raspberry Pi 3/4/5, AWS Graviton)..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc $(GO) build -o bin/$(BINARY_NAME)-arm64 ./cmd/updu

build-arm64-oidc: build-frontend
	@echo "Building Go backend for ARM64 with OIDC support..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc $(GO) build -tags oidc -o bin/$(BINARY_NAME)-arm64-oidc ./cmd/updu

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
