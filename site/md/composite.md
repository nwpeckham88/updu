# Composite Monitor

The Composite monitor rolls up the status of several existing monitors into a single derived monitor. It does **not** perform any network checks of its own — it reads the latest status of the referenced monitors from updu's storage and applies a quorum rule.

Use it to express service-level health on top of probe-level monitors (for example: "checkout is healthy if at least 2 of the 3 web nodes are up").

## Configuration Options

### Basic Settings

- **Name:** A descriptive name for your composite (e.g. `Checkout (any node up)`).
- **Group:** Optional group assignment for organizing monitors.
- **Interval (seconds):** How frequently updu re-evaluates the composite. Composite checks are inexpensive — short intervals are fine.

### Composite Specific Settings

- **Monitor IDs:** The set of underlying monitor IDs to evaluate. Required.
- **Mode:** How updu decides whether the composite is up:
  - `all_up` — every referenced monitor must be `up`.
  - `any_up` — at least one referenced monitor must be `up`.
  - `quorum` — at least *Quorum* of the referenced monitors must be `up`.
- **Quorum:** Required when `mode` is `quorum`. The minimum number of underlying monitors that must be `up`.

If a referenced monitor has no recorded status yet (for example because it has never run), it is treated as not-up for quorum purposes.

## Example Use Cases

- **Cluster health:** Express "API cluster healthy" as `quorum=2` over three regional probes — one slow region won't trigger a page.
- **Failover paths:** Use `any_up` to roll up "primary or backup" connectivity into a single rollup status.
- **Critical-path gating:** Use `all_up` to gate a status page section behind every dependency being up.
