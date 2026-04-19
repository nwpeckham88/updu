# Releasing updu

This document describes how to cut a new release.

## Version source of truth

- The Go binary's reported version is whatever `git describe --tags --always --dirty` produces at build time.
- The Makefile passes that into `internal/version` via ldflags.
- CI overrides it with `VERSION=$GITHUB_REF_NAME` so the published binary's `version` subcommand always equals the pushed tag.
- Display-only references (the marketing site, README install snippet, frontend `package.json`) are bumped by `make release-prep`.

## Tag format

Tags **must** match `vX.Y.Z` or `vX.Y.Z-suffix`, e.g.:

- `v0.6.0` → stable release
- `v0.6.0-beta` / `v0.6.0-rc.1` → marked as a GitHub prerelease

The release workflow's `validate-tag` job rejects anything else.

## Step-by-step

1. **Pick a version.** Decide on `vX.Y.Z` (or a `-beta`/`-rc.N` prerelease).

2. **Run release-prep on a clean tree.**

   ```bash
   make release-prep VERSION=v0.6.0
   git diff --stat   # sanity-check the touched files
   ```

   This updates only display strings: `frontend/package.json`, the install snippet in `README.md`, and the marketing copy in `site/bauhaus/index.html`. The validator rejects malformed versions.

3. **Run the local CI gate.**

   ```bash
   make ci-local            # vet + test + vuln
   make build-all           # cross-compile every platform/variant
   make print-version       # confirm the resolved string
   ```

4. **Commit and tag.**

   ```bash
   git commit -am "chore: prepare v0.6.0"
   git tag -a v0.6.0 -m "v0.6.0"
   git push origin main v0.6.0
   ```

5. **Watch the release workflow.** It runs:
   - `validate-tag` — enforces tag regex and computes prerelease flag.
   - `build` matrix — eight binaries via `make build-<target>`.
   - `smoke` — downloads `updu-linux-amd64`, runs `updu version`, fails if the output does not contain the tag.
   - `release` — uploads binaries + `checksums.txt`, generates GitHub release notes automatically (`generate_release_notes: true`), marks prerelease iff the tag contains `-`.
   - `docker` — multi-arch GHCR publish (`linux/amd64,linux/arm64`), only when the `PUBLISH_DOCKER` repo variable is set to `true`. Stable releases push both `:vX.Y.Z` and `:latest`; prereleases push only `:vX.Y.Z`.

## GHCR publishing

GHCR publishing is opt-in to avoid surprises in forks.

- Enable it: **Settings → Secrets and variables → Actions → Variables → New repository variable** → `PUBLISH_DOCKER = true`.
- Disable it: delete the variable or set it to anything other than `true`.

The job uses the workflow-scoped `GITHUB_TOKEN`, so no PAT is needed.

## If something goes wrong

- **Tag rejected by `validate-tag`** — delete the tag (`git tag -d X && git push origin :refs/tags/X`), fix it, retag.
- **Smoke test failed** — the binary's compiled-in version did not match the tag. Verify the build job used `VERSION=$VERSION` from `validate-tag` and that the Makefile change has not regressed.
- **Release published but assets are wrong** — re-run failed jobs from the Actions tab; the release step uses `softprops/action-gh-release@v2`, which is idempotent for matching files.

## Local verification before tagging

```bash
make ci-local
make build-all                # all 8 binaries, sanity-check sizes
./bin/updu-linux-amd64 version
docker build --build-arg VERSION=v0.6.0-test \
             --build-arg COMMIT=$(git rev-parse --short HEAD) \
             --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
             -t updu:v0.6.0-test .
docker run --rm updu:v0.6.0-test version
```
