# Transaction Monitor

The Transaction monitor runs a sequence of HTTP requests as a single check, allowing each step to extract values from the previous response and inject them into the next request. Use it to validate real user flows — login, then fetch a resource, then perform an action — instead of probing each endpoint in isolation.

If any step fails its expected status or body assertion, the entire monitor is reported as down with the failing step and reason in the message.

## Configuration Options

### Basic Settings

- **Name:** A descriptive name for the flow (e.g. `Login → fetch profile`).
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently the entire transaction is replayed.
- **Timeout (seconds):** Per-request timeout. The transaction also tracks total elapsed time across steps.

### Transaction Specific Settings

- **Steps:** An ordered list of HTTP steps. Each step has:
  - **Method:** HTTP method. Defaults to `GET`.
  - **URL:** Required. May reference variables extracted from earlier steps as `{{var_name}}`.
  - **Headers:** (Optional) Headers to send. Values may also reference `{{var_name}}` placeholders.
  - **Body:** (Optional) Request body. May reference `{{var_name}}` placeholders.
  - **Expected Status:** (Optional) Status code that this step must return.
  - **Expected Body:** (Optional) Substring that must appear in the response body.
  - **Extract:** (Optional) Map of variable name to JSON dot-path (e.g. `token` → `auth.access_token`). Extracted values become available to subsequent steps as `{{token}}`.
- **Skip TLS Verification:** (Optional) Disable certificate validation for the entire flow.

If a step references an `{{undefined}}` variable, the monitor fails immediately with the offending step and variable name in the message — making debugging straightforward.

## Example Use Cases

- **Auth-gated dashboards:** Step 1 logs in and extracts a session token; step 2 fetches `/api/me` with that token and asserts the response.
- **Checkout health:** Walk an anonymous cart through "create cart → add item → start checkout" and alert on any step regression.
- **Webhook round-trip:** Submit a job to an internal API, then poll its status endpoint until it returns the expected terminal value.
