# Demo Workspace

This directory is the repo-local sandbox for running `updu` with the latest local binary, the canonical demo config, and a self-contained SQLite database.

What lives here:

- `updu` -> `../bin/updu`
- `updu.conf` -> `../sample.updu.conf`
- `data/` for the local demo database and runtime files

Recommended quick start:

```bash
make demo-run
```

Open `http://localhost:3000` and register a local admin account. The demo config points at public example endpoints, and the database stays under `demo/data/` so it does not interfere with other local runs.

Manual workflow when needed:

```bash
make build
make sync-demo-dir
cd demo
./updu
```

Run `make sync-demo-dir` to refresh links after deleting, moving, or updating demo files so the workspace points back to the latest local binary and canonical config.