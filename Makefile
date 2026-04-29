# updu — Makefile
#
# `make help` lists all available targets.
#
# Version handling:
#   VERSION is derived from `git describe --tags --always --dirty` with a
#   `dev` fallback. CI overrides it with `VERSION=$GITHUB_REF_NAME`. Release
#   builds inject VERSION/COMMIT/BUILD_DATE/BUILD_TAGS into the Go binary via
#   ldflags so the running binary reports the same string as the git tag.

# ── Default goal ────────────────────────────────────────────
.DEFAULT_GOAL := help

# ── Configuration ───────────────────────────────────────────
BINARY_NAME    := updu
FRONTEND_DIR   := frontend
DEMO_DIR       := demo
BIN_DIR        := bin
SYNC_DEMO_SCRIPT := ./scripts/sync-demo-dir.sh

GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

# Version is derived lazily so `make help` does not need to invoke git.
GIT_DESCRIBE   = $(shell git describe --tags --always --dirty 2>/dev/null)
VERSION       ?= $(if $(GIT_DESCRIBE),$(GIT_DESCRIBE),dev)
RAW_COMMIT     = $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
WORKTREE_DIRTY = $(shell if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
                          if git diff --quiet --ignore-submodules HEAD >/dev/null 2>&1; then printf ''; \
                          else printf -- '-dirty'; fi; \
                        fi)
COMMIT        ?= $(RAW_COMMIT)$(WORKTREE_DIRTY)
BUILD_DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

VERSION_PKG := github.com/updu/updu/internal/version
LDFLAGS      = -X $(VERSION_PKG).Version=$(VERSION) \
               -X $(VERSION_PKG).GitCommit=$(COMMIT) \
               -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

# Cross-compilation matrix.
PLATFORMS := linux-amd64 linux-armv6 linux-armv7 linux-arm64

GOOS_linux-amd64 := linux
GOARCH_linux-amd64 := amd64
GOARM_linux-amd64 :=

GOOS_linux-armv6 := linux
GOARCH_linux-armv6 := arm
GOARM_linux-armv6 := 6

GOOS_linux-armv7 := linux
GOARCH_linux-armv7 := arm
GOARM_linux-armv7 := 7

GOOS_linux-arm64 := linux
GOARCH_linux-arm64 := arm64
GOARM_linux-arm64 :=

# Docker
DOCKER       ?= docker
DOCKER_IMAGE ?= updu

# ── Help ────────────────────────────────────────────────────
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-26s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Version

.PHONY: version print-version
version: ## Print the version that would be embedded in builds
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "BuildDate:  $(BUILD_DATE)"

print-version: ## Print only the version string (for scripting)
	@echo "$(VERSION)"

##@ Build

.PHONY: all build build-oidc build-mongo build-frontend build-all
all: build ## Default: build the local binary

build: build-frontend ## Build the local binary (no OIDC, no Mongo)
	@echo "Building Go backend ($(VERSION))..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/updu

build-oidc: build-frontend ## Build the local binary with OIDC support
	@echo "Building Go backend with OIDC ($(VERSION))..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -tags oidc \
		-ldflags "$(LDFLAGS) -X $(VERSION_PKG).BuildTags=oidc" \
		-o $(BIN_DIR)/$(BINARY_NAME)-oidc ./cmd/updu

build-mongo: build-frontend ## Build the local binary with MongoDB support
	@echo "Building Go backend with Mongo ($(VERSION))..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -tags mongo \
		-ldflags "$(LDFLAGS) -X $(VERSION_PKG).BuildTags=mongo" \
		-o $(BIN_DIR)/$(BINARY_NAME)-mongo ./cmd/updu

build-frontend: ## Build the SvelteKit frontend and sync into the embed dir
	@echo "Building SvelteKit frontend..."
	cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile && pnpm run build
	@echo "Syncing frontend build to embed directory..."
	rm -rf cmd/updu/frontend/build
	cp -r $(FRONTEND_DIR)/build cmd/updu/frontend/build

# ── Cross-platform builds ───────────────────────────────────
.PHONY: $(addprefix build-,$(PLATFORMS)) \
        $(addprefix build-,$(addsuffix -oidc,$(PLATFORMS))) \
        $(addprefix build-,$(addsuffix -mongo,$(PLATFORMS)))

# build_target(platform, suffix, tags)
# - suffix: empty | oidc | mongo
# - tags:   "" | oidc | mongo
define build_target
@echo "Building $(BINARY_NAME)-$(1)$(if $(2),-$(2)) ($(VERSION))..."
@mkdir -p $(BIN_DIR)
CGO_ENABLED=0 GOOS=$(GOOS_$(1)) GOARCH=$(GOARCH_$(1)) GOARM=$(GOARM_$(1)) \
	$(GO) build $(if $(3),-tags "$(3)") \
		-ldflags "$(LDFLAGS)$(if $(3), -X $(VERSION_PKG).BuildTags=$(3))" \
		-o $(BIN_DIR)/$(BINARY_NAME)-$(1)$(if $(2),-$(2)) ./cmd/updu
endef

build-linux-amd64: build-frontend ## Cross-compile linux/amd64
	$(call build_target,linux-amd64,,)
build-linux-amd64-oidc: build-frontend ## Cross-compile linux/amd64 with OIDC
	$(call build_target,linux-amd64,oidc,oidc)
build-linux-amd64-mongo: build-frontend ## Cross-compile linux/amd64 with Mongo
	$(call build_target,linux-amd64,mongo,mongo)
build-linux-armv6: build-frontend ## Cross-compile linux/armv6 (Pi Zero W)
	$(call build_target,linux-armv6,,)
build-linux-armv6-oidc: build-frontend ## Cross-compile linux/armv6 with OIDC
	$(call build_target,linux-armv6,oidc,oidc)
build-linux-armv6-mongo: build-frontend ## Cross-compile linux/armv6 with Mongo
	$(call build_target,linux-armv6,mongo,mongo)
build-linux-armv7: build-frontend ## Cross-compile linux/armv7 (Pi 2/3)
	$(call build_target,linux-armv7,,)
build-linux-armv7-oidc: build-frontend ## Cross-compile linux/armv7 with OIDC
	$(call build_target,linux-armv7,oidc,oidc)
build-linux-armv7-mongo: build-frontend ## Cross-compile linux/armv7 with Mongo
	$(call build_target,linux-armv7,mongo,mongo)
build-linux-arm64: build-frontend ## Cross-compile linux/arm64 (Pi 3+, Graviton)
	$(call build_target,linux-arm64,,)
build-linux-arm64-oidc: build-frontend ## Cross-compile linux/arm64 with OIDC
	$(call build_target,linux-arm64,oidc,oidc)
build-linux-arm64-mongo: build-frontend ## Cross-compile linux/arm64 with Mongo
	$(call build_target,linux-arm64,mongo,mongo)

build-all: $(addprefix build-,$(PLATFORMS)) \
           $(addprefix build-,$(addsuffix -oidc,$(PLATFORMS))) \
           $(addprefix build-,$(addsuffix -mongo,$(PLATFORMS))) ## Build every platform/variant
	@echo "All platform builds complete."

# Backwards-compatible aliases for legacy target names.
.PHONY: build-amd64 build-amd64-oidc build-arm build-arm-oidc build-armv7 build-armv7-oidc build-arm64 build-arm64-oidc
build-amd64: build-linux-amd64
build-amd64-oidc: build-linux-amd64-oidc
build-arm: build-linux-armv6
build-arm-oidc: build-linux-armv6-oidc
build-armv7: build-linux-armv7
build-armv7-oidc: build-linux-armv7-oidc
build-arm64: build-linux-arm64
build-arm64-oidc: build-linux-arm64-oidc

##@ Run

.PHONY: run dev dev-backend dev-frontend demo-run sync-demo-dir
run: build ## Build then run the local binary
	./$(BIN_DIR)/$(BINARY_NAME)

dev-backend: ## Run the Go backend with live source (no embed rebuild)
	$(GO) run ./cmd/updu

dev-frontend: ## Run the SvelteKit dev server
	cd $(FRONTEND_DIR) && pnpm run dev

dev: ## Print instructions for running both dev servers
	@echo "Run these in two terminals:"
	@echo "  make dev-backend   # Go API on :3000"
	@echo "  make dev-frontend  # SvelteKit dev server"

sync-demo-dir: ## Sync configs/binary into the demo directory
	@bash $(SYNC_DEMO_SCRIPT) $(DEMO_DIR)

demo-run: build ## Build then run from the demo directory
	@$(MAKE) sync-demo-dir
	@echo "Starting updu from $(DEMO_DIR)/..."
	@cd $(DEMO_DIR) && ./$(BINARY_NAME)

##@ Quality

.PHONY: fmt vet tidy lint vuln test cover cover-html cover-check e2e-frontend e2e-frontend-oidc test-e2e-update ci-local
fmt: ## Format Go sources
	gofmt -s -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || \
		echo "goimports not installed; skipping (go install golang.org/x/tools/cmd/goimports@latest)"

vet: ## go vet (default, oidc, and mongo tags)
	$(GO) vet ./...
	$(GO) vet -tags oidc ./...
	$(GO) vet -tags mongo ./...

tidy: ## go mod tidy
	$(GO) mod tidy

lint: ## golangci-lint (skipped gracefully if not installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping (https://golangci-lint.run/welcome/install/)"; \
	fi

vuln: ## govulncheck
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...

test: ## Run unit tests
	$(GO) test -v ./...

COVER_PROFILE  ?= coverage.out
COVER_HTML     ?= coverage.html
COVER_MIN      ?= 0   # set >0 to enforce; CI starts at 0 and ratchets

cover: ## Run unit tests with coverage; writes $(COVER_PROFILE)
	$(GO) test -race -covermode=atomic -coverprofile=$(COVER_PROFILE) ./...
	@echo
	@$(GO) tool cover -func=$(COVER_PROFILE) | tail -n 1

cover-html: cover ## Render coverage as HTML at $(COVER_HTML)
	$(GO) tool cover -cover html=$(COVER_PROFILE) -o $(COVER_HTML)
	@echo "Wrote $(COVER_HTML)"

cover-check: cover ## Fail if total coverage < COVER_MIN
	@total=$$($(GO) tool cover -func=$(COVER_PROFILE) | awk '/^total:/ {print $$3}' | tr -d '%'); \
	awk -v t="$$total" -v m="$(COVER_MIN)" 'BEGIN { if (t+0 < m+0) { printf("coverage %.1f%% < required %.1f%%\n", t, m); exit 1 } else { printf("coverage %.1f%% >= required %.1f%%\n", t, m) } }'

e2e-frontend: ## Run frontend Playwright E2E
	cd $(FRONTEND_DIR) && pnpm run test:e2e

e2e-frontend-oidc: ## Run frontend Playwright E2E with OIDC
	cd $(FRONTEND_DIR) && pnpm run test:e2e:oidc

test-e2e-update: ## Run the self-update E2E test
	@echo "Running self-update E2E test..."
	$(GO) test -v -tags=e2e -timeout=5m ./internal/updater

ci-local: vet test vuln ## Mirror the core CI gate locally

##@ Docker

.PHONY: docker docker-oidc docker-mongo
docker: ## Build the runtime Docker image (no OIDC, no Mongo)
	$(DOCKER) build \
		--build-arg BUILD_TAGS= \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest .

docker-oidc: ## Build the runtime Docker image with OIDC
	$(DOCKER) build \
		--build-arg BUILD_TAGS=oidc \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION)-oidc \
		-t $(DOCKER_IMAGE):oidc .

docker-mongo: ## Build the runtime Docker image with Mongo
	$(DOCKER) build \
		--build-arg BUILD_TAGS=mongo \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION)-mongo \
		-t $(DOCKER_IMAGE):mongo .

##@ Docs

.PHONY: docs
docs: ## Regenerate marketing docs HTML from site/md/*.md
	@echo "Building docs from site/md/ -> site/docs/..."
	@cd scripts/build-docs && $(GO) run .

##@ Release

.PHONY: release-prep
# release-prep VERSION=vX.Y.Z[-suffix]
# Updates display-only version strings across the repo. Build-time version is
# always derived from `git describe`, so this target only touches user-facing
# documentation, the marketing site, and frontend/package.json.
release-prep: ## Bump display-only version strings (VERSION=vX.Y.Z required)
	@if ! echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$$'; then \
		echo "ERROR: VERSION must look like vX.Y.Z or vX.Y.Z-suffix (got '$(VERSION)')"; \
		exit 1; \
	fi
	@echo "Preparing release strings for $(VERSION)..."
	@VTAG="$(VERSION)"; BARE="$${VTAG#v}"; \
	sed -i -E 's/^([[:space:]]*"version":[[:space:]]*")[^"]+(",)/\1'"$$BARE"'\2/' \
		$(FRONTEND_DIR)/package.json; \
	sed -i -E 's@releases/download/v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?/updu-linux-amd64@releases/download/'"$$VTAG"'/updu-linux-amd64@g' \
		README.md; \
	sed -i -E 's@(<span class="meta-line"><span class="meta-dot"></span> )v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?@\1'"$$VTAG"'@' \
		site/bauhaus/index.html; \
	sed -i -E 's@(<strong class="release-tag">)v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?@\1'"$$VTAG"'@' \
		site/bauhaus/index.html; \
	sed -i -E 's@(releases/tag/)v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?@\1'"$$VTAG"'@g' \
		site/bauhaus/index.html; \
	sed -i -E 's@(VER=)v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?@\1'"$$VTAG"'@' \
		site/bauhaus/index.html
	@echo
	@echo "Done. Review the changes:"
	@git diff --stat -- $(FRONTEND_DIR)/package.json README.md site/bauhaus/index.html
	@echo
	@echo "Next: commit, then tag with:"
	@echo "  git commit -am 'chore: prepare $(VERSION)'"
	@echo "  git tag -a $(VERSION) -m '$(VERSION)'"
	@echo "  git push origin main $(VERSION)"

##@ Cleanup

.PHONY: clean
clean: ## Remove build artifacts
	$(GO) clean
	rm -rf $(BIN_DIR)/
	rm -rf cmd/updu/frontend/build
	rm -rf $(FRONTEND_DIR)/build
	rm -rf $(FRONTEND_DIR)/.svelte-kit
	rm -rf release-binaries artifacts
