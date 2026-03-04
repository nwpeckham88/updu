.PHONY: all build run test clean dev

BINARY_NAME=updu
FRONTEND_DIR=frontend
GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

all: build

# Generate Tailwind, build SvelteKit, then build Go
build: build-frontend
	@echo "Building Go backend..."
	CGO_ENABLED=1 $(GO) build -o bin/$(BINARY_NAME) ./cmd/updu

build-oidc: build-frontend
	@echo "Building Go backend with OIDC support..."
	CGO_ENABLED=1 $(GO) build -tags oidc -o bin/$(BINARY_NAME)-oidc ./cmd/updu

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
