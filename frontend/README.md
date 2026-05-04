# updu Frontend

SvelteKit (Svelte 5) + TailwindCSS v4 single-page application.

The production build is embedded into the Go binary via `//go:embed` and served as a static SPA with client-side routing fallback.

## Development

```bash
pnpm install
pnpm run dev        # Dev server with Vite proxy to backend on :3000
```

## Build

```bash
pnpm run build      # Output to build/
pnpm run check      # Type checking
```

The `make build` target in the project root handles building the frontend, copying it into `cmd/updu/frontend/build/`, and compiling the Go binary.

## Local E2E Workflow

From the frontend directory, install browser dependencies once:

```bash
pnpm run test:e2e:install
```

Run baseline local browser tests against the real Go app:

```bash
pnpm run test:e2e
```

The test:e2e scripts run test:e2e:prepare first, which builds frontend assets,
syncs embedded files, and starts the disposable local app test stack.

Useful variants:

```bash
pnpm run test:e2e:headed
pnpm run test:e2e:debug
pnpm run test:e2e:report
pnpm run test:e2e:oidc
pnpm run test:e2e:oidc:headed
pnpm run test:e2e:oidc:debug
```

From the repository root, use these canonical make targets:

```bash
make e2e-frontend
make e2e-frontend-oidc
```
