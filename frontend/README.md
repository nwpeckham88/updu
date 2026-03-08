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
