# Demo Workspace

This directory is the repo-local sandbox for running `updu` with the latest local binary, the canonical demo config, and a self-contained SQLite database.

What lives here:

- `updu` -> `../bin/updu`
- `updu.conf` -> `../sample.updu.conf`
- `data/` for the local demo database and runtime files

Quick start:

```bash
make build
cd demo
./updu
```

Open `http://localhost:3000` and register a local admin account. The demo config points at public example endpoints, and the database stays under `demo/data/` so it does not interfere with other local runs.

If you ever remove the symlinks locally, run `make sync-demo-dir` to recreate them. `make demo-run` builds and starts the app from this directory in one step.