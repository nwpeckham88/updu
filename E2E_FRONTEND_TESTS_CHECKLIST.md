## Playwright Frontend Baseline (Archive)

Status: completed baseline, kept as an archive marker.

This checklist file previously tracked the initial local-first Playwright rollout.
That baseline is now implemented and maintained through active docs and commands.

Use these current sources of truth:

- frontend/README.md for frontend-local E2E commands
- CONTRIBUTING.md for contributor quality gates and required test runs
- Makefile targets e2e-frontend and e2e-frontend-oidc for repo-root execution

Current baseline outcomes:

- Real-app Playwright lane runs against the embedded Go binary
- Auth/session, monitors, settings, incidents, and status-page flows are covered
- OIDC lane is available for auth-path validation
- Failure artifacts are available through Playwright reporting

Follow-up work should be tracked in issues or PR scope docs instead of reviving this file as a task checklist.
