---
description: "Cognitive-ergonomics patterns for updu's operator UI. Apply when editing Svelte routes, dashboard panels, status/health widgets, or any analytics surface."
applyTo: "frontend/src/**/*.{svelte,ts}"
---
# Operator UI — Cognitive Ergonomics Patterns

These patterns were established during the analytics redesign (`frontend/src/routes/stats/+page.svelte`) and the shared status helpers in [frontend/src/lib/monitor-tones.ts](frontend/src/lib/monitor-tones.ts). Reuse them everywhere monitor health, latency, incidents, or counts are surfaced. They exist to keep a tired, interrupted operator from doing extra mental work.

## Non-negotiables

1. **Never encode status with color alone.** Pair every hue with a glyph + text label. Use the shared helpers — do not reinvent:
   - `statusIcon(status)` → Lucide glyph (Check / X / AlertTriangle / Pause / Circle)
   - `statusLabel(status)` → human label ("Operational", "Down", "Degraded", "Paused", "Pending")
   - `statusTone(status)` / `statusTextClass(status)` → semantic tone class
   - `statusPattern(status)` → CVD-safe fill pattern for bars/donuts
2. **Use semantic color tokens, never raw hex.** Always `text-success | text-warning | text-danger | text-primary | text-text | text-text-muted | text-text-subtle` and matching `bg-*/border-*` variants. Theme overrides only flow through the tokens.
3. **Tabular figures for every number.** Add `font-mono tabular-nums` to any latency, count, percentage, or duration so digits don't jitter when values change. Required in tables, stat cards, leaderboards, and live counters.
4. **Accessible labels on every status region.** Wrap status regions with `aria-label` that combines verdict + sub-line (e.g. `aria-label="System health: Degraded. 2 active incidents."`). Use `<dl>/<dt>/<dd>` for label-value pairs in stat strips.

## Layout hierarchy (top-to-bottom)

Operator screens follow this order so Level-1 situational awareness lands in <5 seconds:

1. **System health verdict bar** — single sentence ("All systems operational" / "Degraded" / "Service outage"), large icon, semantic ring color, plus a 3-metric `<dl>` strip (Down / Active incidents / Avg latency 24h). See the `verdict` derived state and `verdictRing` map for the canonical implementation.
2. **Persistent active-incident banner** — only rendered when `activeIncidents.length > 0` and you are not already on the incidents view. Clickable, jumps to the incidents tab, surfaces severity + relative time.
3. **Tabs** with badges showing only counts that demand attention (active incidents, monitor count). Inactive badges use `bg-surface-elevated`; the incidents badge turns `bg-danger/15 text-danger` only when `> 0`.
4. **Detail panels** (charts, leaderboards, distributions) live below — progressive disclosure.

## Verdict states (reuse for any "system health" widget)

```ts
type Verdict = {
  key: "operational" | "degraded" | "outage";
  tone: "success" | "warning" | "danger";
  icon: ShieldCheck | ShieldAlert | ShieldX;
};
```
Rules:
- `down > 0 || criticalIncidents > 0` → outage / `ShieldX` / danger
- `activeIncidents > 0` → degraded / `ShieldAlert` / warning
- otherwise → operational / `ShieldCheck` / success

Use `lucide-svelte`'s Shield* icons consistently — they are the project's "verdict" glyph family. Do not substitute generic Check/Alert here; those are reserved for per-monitor status.

## Tone thresholds (single source of truth)

Always import from `monitor-tones.ts`. Inline ternaries with magic numbers are a smell.

| Helper | Thresholds |
|---|---|
| `uptimeTone(pct)` | ≥99 success · ≥95 warning · else danger |
| `latencyTone(ms)` | >3000 danger · >1000 warning · else neutral |
| `statusTone(status)` | up→success · down→danger · degraded/warning→warning · else neutral |

If a new metric needs a tone, add a helper to `monitor-tones.ts` rather than inlining thresholds in components.

## Charting rules

- **Prefer bullet graphs over gauges.** Show value + fleet-median reference line. The latency leaderboard in `stats/+page.svelte` is the reference implementation: derive `fleetP95` / `fleetAvg` once, draw the value bar against `max`, and overlay a 1px median tick.
- **Donuts only for completion/uptime ratios** — use the existing [`StatusDonut`](frontend/src/lib/components/charts/status-donut.svelte) component; do not roll your own.
- **No chartjunk.** No 3D, no gradients-as-data, no decorative gridlines. One semantic accent per chart.
- **Categorical color sets** belong in a single `Record<string, {label, color}>` per concept (see `typeMeta` and `codeColors` in `stats/+page.svelte`). HSL only, so theme drift stays controlled.
- **Empty/loading/error states are required.** Loading → `<Skeleton>` of the eventual height. Error → `TriangleAlert` icon + message inside `.card`. Never collapse the layout.

## Tables & leaderboards

- Sortable columns toggle direction on re-click; first-click defaults: text → `asc`, numeric → `desc`, uptime → `asc` (worst-first surfaces problems).
- Null/undefined values sort to the worst position, not silently to zero.
- Filter inputs are paired with a status segmented control; empty results render an explicit empty state, never a blank table.
- Right-align numerics, left-align names, use `tabular-nums` everywhere.

## Alarm-fatigue discipline

- Only **active, unresolved, actionable** items get persistent visual weight.
- Resolved incidents collapse to `slice(0, 10)` and use neutral tones.
- Counts of zero render in `text-text` (not green). Green is reserved for an explicit positive verdict.
- Do not render danger styling for paused or pending monitors — those are operator-chosen states, not failures. Use `text-text-subtle`.

## Typography micro-rules

- Labels above numbers: `text-[10px] uppercase tracking-widest text-text-subtle`.
- Primary metric numbers: `font-mono text-2xl font-bold tabular-nums`.
- Secondary fractions (e.g. `2/15`): `text-sm text-text-subtle font-medium` inside the same `<dd>`.
- Section card padding: `p-5` for status bars, `p-6` for chart cards. Keep this rhythm.

## When adding a new operator screen

Checklist before opening the PR:

- [ ] Verdict bar at top with icon + label + sub + 3-metric strip
- [ ] Every status uses `statusIcon` + `statusLabel` + tone helper from `monitor-tones.ts`
- [ ] Every number has `font-mono tabular-nums`
- [ ] Loading skeletons match final layout height
- [ ] Aria labels on every health/status region
- [ ] Sort defaults surface the worst items first
- [ ] No raw hex colors; semantic tokens only
- [ ] No new threshold magic numbers (extend `monitor-tones.ts` instead)
- [ ] Resolved/paused items deprioritized visually
- [ ] CVD check: would the screen still parse in greyscale?

## Anti-patterns to reject in review

- Color-only status pills (no glyph, no label)
- Gauges where a bullet graph would do
- Inline `#hex` colors in templates
- Inline tone ternaries duplicating `monitor-tones.ts` logic
- Proportional digits for live counters (causes width jitter)
- Persistent red banners for resolved or acknowledged incidents
- Empty tables/charts with no skeleton, error, or empty state
- Tab badges that show every count regardless of urgency

## Specialist agents

For deeper review or larger changes, invoke the project's specialist agents:

- [Cognitive Ergonomics Expert](.github/agents/cognitive-ergonomics-expert.agent.md) — UX/cognitive-load review of any operator surface
- [Svelte UI Refiner](.github/agents/svelte-ui-refiner.agent.md) — Svelte 5 / SvelteKit code-level refactors and verification
