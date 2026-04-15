## Checklist: Playwright Frontend Test Baseline

Use this as the execution handoff for adding local-first browser E2E coverage to updu.

**Goal**
- [ ] Add Playwright browser E2E coverage for the embedded Svelte frontend against the real Go app.
- [ ] Keep the first pass local-first and deterministic.
- [ ] Cover: login/session, monitor list search/sort/filter, monitor CRUD, settings smoke, incidents/status smoke.

**Phase 1: Test Harness**
- [ ] Add Playwright dependencies in `frontend/package.json`.
- [ ] Add frontend scripts for browser install, test run, headed/debug run, and report viewing in `frontend/package.json`.
- [ ] Add Playwright config in the frontend workspace with failure artifacts enabled: trace on retry, screenshot on failure, video on failure.
- [ ] Decide the canonical local startup path: built Go binary or `go run` with prebuilt embedded frontend.
- [ ] Add a single root command or make target in `Makefile` that builds frontend assets, syncs the embed directory, starts the app with test env vars, and runs Playwright.

**Phase 1 Exit Criteria**
- [ ] A developer can run one documented command and reach a green Playwright smoke test locally.
- [ ] The app starts with a disposable SQLite DB and does not depend on manual setup.

**Phase 2: Environment Control**
- [ ] Standardize test env vars using the config surface in `internal/config/config.go`.
- [ ] Set fixed values for auth secret, admin username, admin password, DB path, port, host, and base URL.
- [ ] Ensure the chosen startup path uses a fresh DB per run or a clearly isolated temp DB.
- [ ] Confirm the admin bootstrap path works without manual registration.
- [ ] Confirm session cookies work correctly under the chosen base URL and port.

**Phase 2 Exit Criteria**
- [ ] Login works on a clean run without manual intervention.
- [ ] Re-running the suite does not reuse stale state unless explicitly intended.

**Phase 3: Automatable UI Surface**
- [ ] Audit `frontend/src/routes/login/+page.svelte` for accessible locators that Playwright can target reliably.
- [ ] Audit `frontend/src/routes/monitors/+page.svelte` for stable locators around search, sort headers, row actions, dialogs, and empty/loading states.
- [ ] Audit `frontend/src/routes/settings/+page.svelte` for a stable page-ready assertion.
- [ ] Audit `frontend/src/routes/incidents/+page.svelte` for a stable page-ready assertion.
- [ ] Audit the status pages route for a stable page-ready assertion.
- [ ] Add explicit test ids only where roles, labels, or visible text are too ambiguous or too unstable.
- [ ] Add stable hooks for repeated monitor row actions if dropdown/menu targeting is otherwise brittle.

**Phase 3 Exit Criteria**
- [ ] Each planned flow has a reliable locator strategy documented before spec implementation starts.
- [ ] No assertion depends on CSS classes or fragile DOM order unless unavoidable.

**Phase 4: Shared Playwright Helpers**
- [ ] Add a shared auth helper for login and post-login ready checks.
- [ ] Add a helper for first-run setup handling only if bootstrap env vars do not fully remove setup branching.
- [ ] Add a monitor data helper to create prerequisite records quickly when a test does not need full UI setup.
- [ ] Keep at least one full monitor CRUD test entirely UI-driven.
- [ ] Add cleanup conventions so tests remain isolated and repeatable.

**Phase 4 Exit Criteria**
- [ ] Specs can share auth and seeded state without duplicating setup logic.
- [ ] Helpers reduce setup time without hiding the main user journey under test.

**Phase 5: First Spec Set**
- [ ] Add login/session coverage: login page renders, valid login succeeds, post-login landing is correct, session persists across navigation, logout redirects correctly if included.
- [ ] Add monitor list coverage: page loads, seeded monitors render, search filters correctly, sort toggles correctly for name/status/latency, empty state behaves correctly.
- [ ] Add monitor CRUD coverage: create a monitor through the UI, edit it, pause or resume it, and delete it.
- [ ] Add settings smoke coverage: authenticated navigation works and the page reaches a stable ready state.
- [ ] Add incidents smoke coverage: authenticated navigation works and the page reaches a stable ready state.
- [ ] Add status page smoke coverage: target the intended public or managed status route and confirm the page renders as expected.

**Phase 5 Exit Criteria**
- [ ] The initial flow set passes against a clean local environment.
- [ ] At least one spec exercises the full UI path for monitor lifecycle changes.

**Phase 6: Local Workflow and Debugging**
- [ ] Document local prerequisites: `pnpm install`, Playwright browser install, and the single test command.
- [ ] Ensure failure artifacts are easy to inspect locally.
- [ ] Add a headed/debug workflow for investigating flaky UI interactions.
- [ ] Verify the suite can be re-run without manual teardown steps.

**Phase 6 Exit Criteria**
- [ ] A new contributor can run and debug the suite from the repo instructions alone.

**Verification Checklist**
- [ ] Run the chosen local E2E command from a clean state.
- [ ] Confirm the app is reachable before tests start.
- [ ] Run the full initial suite once and confirm all tests pass.
- [ ] Re-run the monitor-focused specs multiple times to probe for timing issues.
- [ ] Confirm traces, screenshots, and videos are produced on forced failure.
- [ ] Confirm no test relies on OIDC, SSE timing, or pre-existing production-like data.

**Scope Guardrails**
- [ ] Do not add frontend unit/component tests in this pass.
- [ ] Do not add OIDC browser coverage in this pass.
- [ ] Do not add realtime/SSE assertions in this pass.
- [ ] Do not make CI gating mandatory until the local suite is stable.

**Reference Files**
- `frontend/package.json`
- `Makefile`
- `frontend/src/routes/login/+page.svelte`
- `frontend/src/routes/monitors/+page.svelte`
- `frontend/src/routes/settings/+page.svelte`
- `frontend/src/routes/incidents/+page.svelte`
- `cmd/updu/main.go`
- `internal/config/config.go`
- `internal/auth/auth.go`
- `internal/api/api_test.go`

**Deferred Follow-Ups**
- [ ] Add CI execution in `.github/workflows/ci.yml` after local stability is proven.
- [ ] Add deeper incidents/status assertions once fixture setup is simple.
- [ ] Evaluate realtime/SSE coverage once the baseline suite is stable.