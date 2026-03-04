.PHONY: all build run test clean dev

BINARY_NAME=updu
FRONTEND_DIR=frontend

all: build

# Generate Tailwind, build SvelteKit, then build Go
build: build-frontend
	@echo "Building Go backend..."
	CGO_ENABLED=1 go build -o bin/$(BINARY_NAME) ./cmd/updu

build-frontend:
	@echo "Building SvelteKit frontend..."
	cd frontend && pnpm run build
	@echo "Syncing frontend build to embed directory..."
	rm -rf cmd/updu/frontend/build
	cp -r frontend/build cmd/updu/frontend/build

run: build
	./bin/$(BINARY_NAME)

dev-backend:
	go run ./cmd/updu

dev-frontend:
	cd $(FRONTEND_DIR) && pnpm run dev

test:
	go test -v ./...

clean:
	go clean
	rm -rf bin/
	rm -rf cmd/updu/frontend/build
