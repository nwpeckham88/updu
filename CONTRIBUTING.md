# Contributing to updu

Thank you for your interest in contributing to **updu**! Here are the guidelines for setting up the environment and submitting your pull requests.

## Start Here

Use this order for a smooth first run:

1. Follow **Local Development Setup** below.
2. Use `make demo-run` to verify the full embedded app starts from the repo-local demo workspace.
3. Run `go test -v ./...` and `pnpm --dir frontend run check` before opening a PR.
4. For UI-affecting changes, also run browser E2E (`make e2e-frontend`) before opening your PR.

## Tech Stack

- **Backend:** Go (`1.26+`), SQLite (pure Go via `modernc.org/sqlite`, `CGO_ENABLED=0`)
- **Frontend:** SvelteKit (`Svelte 5`), TailwindCSS (`v4`)
- **Communication:** REST APIs & Server-Sent Events (SSE)

## Local Development Setup

To get up and running locally, you'll need both `Go` and `Node.js`/`pnpm` installed.

1. **Clone the repository:**

   ```bash
   git clone https://github.com/nwpeckham88/updu.git
   cd updu
   ```

2. **Install frontend dependencies:**

   ```bash
   cd frontend
   pnpm install
   ```

3. **Run the development servers:**
   Open two terminals.

   *Terminal 1 (Backend):*

   ```bash
   make dev-backend
   ```

   *Terminal 2 (Frontend):*

   ```bash
   make dev-frontend
   ```

   The backend API will run on `localhost:3000` and the SvelteKit frontend will run on its dev port with a Vite proxy pointing to the backend API.

## Code Quality and Testing

Before submitting a pull request, please ensure your code meets our quality standards:

### Frontend

- Components should use Tailwind utility classes.
- Ensure `pnpm run check` and `pnpm run build` pass without warnings.
- Run the browser E2E suite for UI-affecting changes:

   ```bash
   pnpm --dir frontend run test:e2e:install
   make e2e-frontend
   ```

   This runs Playwright against the real Go binary with the embedded frontend and a disposable SQLite database.
   The E2E command path already runs the prepare/build step, so no separate manual build is required.

   For auth or OIDC-affecting UI changes, also run:

   ```bash
   make e2e-frontend-oidc
   ```

### Backend

- Ensure your code passes standard Go formatting and vetting (`go fmt ./...`, `go vet ./...`).
- Tests should be written for new features. Ensure all tests pass (`go test -v ./...`).
- For OIDC or auth-path changes, also run `go test -v -race -tags oidc ./internal/api -run TestAPI_OIDC_`.
- We use a specific Makefile logic to embed the built SvelteKit SPA within the Go executable.

### Updates and Release Channels

- Self-update behavior is now channel-aware. `Settings -> System` lets admins choose `Stable only` or `Include prereleases`.
- If you change updater behavior, cover both channel selection and fallback behavior in `internal/updater` and `internal/api` tests.
- Beta release notes should call out whether the release is intended for the prerelease channel only.

## Repository Hygiene

Do not commit generated or ephemeral artifacts. In particular, keep these untracked:

- frontend build output and local framework caches
- Playwright reports, test-results, and auth/session state files
- temporary local runtime directories such as `.tmp/`

If you accidentally stage generated files, unstage and remove tracked copies before opening your PR.

For local demo runs, prefer `make demo-run` and `make sync-demo-dir` over manual symlink edits.

## Pull Request Process

1. **Branch off `main`:** Create a descriptively named branch for your feature or fix (e.g., `feature/custom-webhooks` or `fix/auth-cookie`).
2. **Commit locally:** Write clear, concise commit messages.
3. **Open a PR against `main`:** Follow the pull request template carefully. Provide context, screenshots if applicable, and explain your testing methodology.
4. **CI Checks:** Automated GitHub Actions will run tests, linting, and a vulnerability scan on your PR. All checks must be green before merging.
