<script lang="ts">
    import { onMount } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import {
        BarChart3,
        TrendingUp,
        Zap,
        Clock,
        TriangleAlert,
        Server,
        Gauge,
        Activity,
        LayoutDashboard,
        ListOrdered,
        AlertCircle,
        ArrowUp,
        ArrowDown,
        ArrowUpDown,
        ArrowRight,
        Search,
        CheckCircle2,
        ShieldCheck,
        ShieldAlert,
        ShieldX,
    } from "lucide-svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Stat from "$lib/components/ui/stat.svelte";
    import StatusDonut from "$lib/components/charts/status-donut.svelte";
    import {
        statusIcon,
        statusLabel,
        statusTextClass,
        uptimeTone,
        uptimeTextClass,
        uptimeBarVar,
        latencyTone,
        latencyBucketTone,
    } from "$lib/monitor-tones";
    import type { Tone } from "$lib/monitor-tones";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { format, parseISO, formatDistanceToNow } from "date-fns";

    let stats = $state<any>(null);
    let incidents = $state<any[]>([]);
    let loading = $state(true);
    let error = $state("");

    // ── Tabs ─────────────────────────────────────────────────
    type TabValue = "overview" | "performance" | "monitors" | "incidents";
    let activeTab = $state<TabValue>("overview");

    const tabs: { value: TabValue; label: string; icon: any }[] = [
        { value: "overview", label: "Overview", icon: LayoutDashboard },
        { value: "performance", label: "Performance", icon: Gauge },
        { value: "monitors", label: "Monitors", icon: ListOrdered },
        { value: "incidents", label: "Incidents", icon: AlertCircle },
    ];

    function tabButtonId(value: TabValue): string {
        return `stats-tab-${value}`;
    }

    function tabPanelId(value: TabValue): string {
        return `stats-panel-${value}`;
    }

    function activateTab(value: TabValue) {
        activeTab = value;
        requestAnimationFrame(() => {
            document.getElementById(tabButtonId(value))?.focus();
        });
    }

    function handleTabKeydown(event: KeyboardEvent, value: TabValue) {
        const index = tabs.findIndex((tab) => tab.value === value);
        if (index < 0) return;

        const lastIndex = tabs.length - 1;
        const keyTargets: Record<string, TabValue> = {
            ArrowRight: tabs[index === lastIndex ? 0 : index + 1].value,
            ArrowLeft: tabs[index === 0 ? lastIndex : index - 1].value,
            Home: tabs[0].value,
            End: tabs[lastIndex].value,
        };
        const next = keyTargets[event.key];
        if (!next) return;

        event.preventDefault();
        activateTab(next);
    }

    // ── Leaderboard sort + filter ────────────────────────────
    type SortKey =
        | "name"
        | "status"
        | "uptime_24h"
        | "avg_latency"
        | "p95_latency"
        | "min_latency"
        | "max_latency"
        | "total_checks";
    let sortKey = $state<SortKey>("uptime_24h");
    let sortDir = $state<"asc" | "desc">("asc");
    let monitorFilter = $state("");
    let statusFilter = $state<"all" | "up" | "down" | "paused" | "pending">(
        "all",
    );

    function toggleSort(key: SortKey) {
        if (sortKey === key) {
            sortDir = sortDir === "asc" ? "desc" : "asc";
        } else {
            sortKey = key;
            // Sensible defaults: text asc, numeric desc (worst-first for uptime)
            sortDir =
                key === "name" || key === "status" || key === "uptime_24h"
                    ? "asc"
                    : "desc";
        }
    }

    const statusRank: Record<string, number> = {
        down: 0,
        pending: 1,
        paused: 2,
        up: 3,
    };

    function compareValues(a: any, b: any, key: SortKey): number {
        if (key === "name") {
            return (a.name ?? "").localeCompare(b.name ?? "");
        }
        if (key === "status") {
            return (
                (statusRank[a.status] ?? 99) - (statusRank[b.status] ?? 99)
            );
        }
        const av = a[key];
        const bv = b[key];
        // Treat null/undefined as worst (sorts last asc, first desc)
        const aNull = av == null;
        const bNull = bv == null;
        if (aNull && bNull) return 0;
        if (aNull) return 1;
        if (bNull) return -1;
        return av - bv;
    }

    const sortedMonitors = $derived.by(() => {
        const list = stats?.monitors ?? [];
        const filtered = list.filter((m: any) => {
            if (statusFilter !== "all" && m.status !== statusFilter)
                return false;
            if (monitorFilter.trim() === "") return true;
            const q = monitorFilter.toLowerCase();
            return (
                m.name?.toLowerCase().includes(q) ||
                m.type?.toLowerCase().includes(q) ||
                m.group?.toLowerCase().includes(q)
            );
        });
        const sorted = [...filtered].sort((a, b) => {
            const cmp = compareValues(a, b, sortKey);
            return sortDir === "asc" ? cmp : -cmp;
        });
        return sorted;
    });

    // ── Incidents derived buckets ────────────────────────────
    const activeIncidents = $derived(
        (incidents ?? []).filter((i: any) => i.status !== "resolved"),
    );
    const resolvedIncidents = $derived(
        (incidents ?? [])
            .filter((i: any) => i.status === "resolved")
            .slice(0, 10),
    );

    const severityMeta: Record<
        string,
        { label: string; tone: string; ring: string }
    > = {
        critical: {
            label: "Critical",
            tone: "text-danger",
            ring: "bg-danger/10 border-danger/30",
        },
        major: {
            label: "Major",
            tone: "text-warning",
            ring: "bg-warning/10 border-warning/30",
        },
        minor: {
            label: "Minor",
            tone: "text-text-muted",
            ring: "bg-surface-elevated border-border",
        },
    };

    const incidentStatusMeta: Record<string, { label: string; tone: string }> =
        {
            investigating: { label: "Investigating", tone: "text-danger" },
            identified: { label: "Identified", tone: "text-warning" },
            monitoring: { label: "Monitoring", tone: "text-primary" },
            resolved: { label: "Resolved", tone: "text-success" },
        };

    function relativeTime(iso?: string): string {
        if (!iso) return "—";
        try {
            return formatDistanceToNow(parseISO(iso), { addSuffix: true });
        } catch {
            return iso;
        }
    }

    onMount(async () => {
        try {
            const [statsResp, incidentsResp] = await Promise.all([
                fetchAPI("/api/v1/stats"),
                fetchAPI("/api/v1/incidents").catch(() => []),
            ]);
            stats = statsResp;
            incidents = incidentsResp ?? [];
        } catch (e: any) {
            error = e.message || "Failed to load analytics";
        } finally {
            loading = false;
        }
    });

    function pct(n: number) {
        return Math.min(100, Math.max(0, n));
    }

    // CVD-safe diagonal stripe overlay for the "down" segment of the hourly
    // timeline. Pairs with the danger color so users in greyscale (or with
    // red/green CVD) can still distinguish failed checks from healthy ones.
    const downStripe =
        "repeating-linear-gradient(135deg, var(--color-danger) 0 3px, color-mix(in oklab, var(--color-danger) 70%, transparent) 3px 6px)";

    // Tone -> bar fill CSS color (semantic tokens only). Used by the latency
    // distribution so problem buckets pop pre-attentively without a custom
    // palette.
    function toneBarColor(t: Tone): string {
        switch (t) {
            case "success":
                return "var(--color-success)";
            case "primary":
                return "var(--color-primary)";
            case "warning":
                return "var(--color-warning)";
            case "danger":
                return "var(--color-danger)";
            default:
                return "var(--color-text-subtle)";
        }
    }

    // ── System health verdict (Level-1 situational awareness) ──
    type Verdict = {
        key: "operational" | "degraded" | "outage";
        label: string;
        sub: string;
        tone: "success" | "warning" | "danger";
        icon: any;
    };

    const downCount = $derived(
        (stats?.monitors ?? []).filter((m: any) => m.status === "down").length,
    );
    const monitorTotal = $derived(stats?.monitors?.length ?? 0);
    const criticalIncidents = $derived(
        activeIncidents.filter((i: any) => i.severity === "critical").length,
    );

    const verdict = $derived.by<Verdict>(() => {
        if (downCount > 0 || criticalIncidents > 0) {
            return {
                key: "outage",
                label: "Service outage",
                sub:
                    downCount > 0
                        ? `${downCount} of ${monitorTotal} monitors down`
                        : `${criticalIncidents} critical incident${criticalIncidents === 1 ? "" : "s"}`,
                tone: "danger",
                icon: ShieldX,
            };
        }
        if (activeIncidents.length > 0) {
            return {
                key: "degraded",
                label: "Degraded",
                sub: `${activeIncidents.length} active incident${activeIncidents.length === 1 ? "" : "s"}`,
                tone: "warning",
                icon: ShieldAlert,
            };
        }
        return {
            key: "operational",
            label: "All systems operational",
            sub:
                monitorTotal > 0
                    ? `${monitorTotal} monitor${monitorTotal === 1 ? "" : "s"} reporting healthy`
                    : "No monitors configured",
            tone: "success",
            icon: ShieldCheck,
        };
    });

    const verdictRing: Record<Verdict["tone"], string> = {
        success: "border-success/40 bg-success/10",
        warning: "border-warning/40 bg-warning/10",
        danger: "border-danger/50 bg-danger/15",
    };
    const verdictIconClass: Record<Verdict["tone"], string> = {
        success: "text-success",
        warning: "text-warning",
        danger: "text-danger",
    };

    // ── Latency reference (fleet median for bullet-graph context) ──
    const fleetP95 = $derived.by(() => {
        const vals = (stats?.monitors ?? [])
            .map((m: any) => m.p95_latency)
            .filter((v: any) => v != null) as number[];
        if (vals.length === 0) return { median: null, max: 1 };
        const sorted = [...vals].sort((a, b) => a - b);
        const median = sorted[Math.floor(sorted.length / 2)];
        const max = Math.max(...sorted, 1);
        return { median, max };
    });
    const fleetAvg = $derived.by(() => {
        const vals = (stats?.monitors ?? [])
            .map((m: any) => m.avg_latency)
            .filter((v: any) => v != null) as number[];
        if (vals.length === 0) return { median: null, max: 1 };
        const sorted = [...vals].sort((a, b) => a - b);
        const median = sorted[Math.floor(sorted.length / 2)];
        const max = Math.max(...sorted, 1);
        return { median, max };
    });

    function maxHourly(): number {
        if (!stats?.hourly_timeline) return 1;
        return Math.max(
            1,
            ...stats.hourly_timeline.map((h: any) => h.up + h.down),
        );
    }

    function maxLatDist(): number {
        if (!stats?.latency_distribution) return 1;
        return Math.max(
            1,
            ...stats.latency_distribution.map((b: any) => b.count),
        );
    }

    function formatHour(iso: string): string {
        try {
            return format(parseISO(iso), "HH:mm");
        } catch {
            return iso?.slice(11, 16) ?? "";
        }
    }

    function totalOfArray(arr: any[], key: string): number {
        return (arr || []).reduce((s: number, x: any) => s + (x[key] || 0), 0);
    }

    const typeMeta: Record<string, { label: string; color: string }> = {
        http: { label: "HTTP", color: "hsl(217 91% 60%)" },
        tcp: { label: "TCP", color: "hsl(280 70% 60%)" },
        ping: { label: "Ping", color: "hsl(142 71% 45%)" },
        dns: { label: "DNS", color: "hsl(38 92% 50%)" },
        ssl: { label: "SSL", color: "hsl(330 70% 55%)" },
        ssh: { label: "SSH", color: "hsl(170 70% 45%)" },
        json: { label: "JSON API", color: "hsl(200 80% 55%)" },
        push: { label: "Push", color: "hsl(150 55% 48%)" },
        websocket: { label: "WebSocket", color: "hsl(262 83% 65%)" },
        smtp: { label: "SMTP", color: "hsl(16 85% 58%)" },
        udp: { label: "UDP", color: "hsl(28 92% 55%)" },
        redis: { label: "Redis", color: "hsl(350 73% 57%)" },
        postgres: { label: "PostgreSQL", color: "hsl(212 63% 52%)" },
        mysql: { label: "MySQL", color: "hsl(192 62% 47%)" },
        mongo: { label: "MongoDB", color: "hsl(132 43% 42%)" },
        https: { label: "HTTPS", color: "hsl(228 72% 62%)" },
        composite: { label: "Composite", color: "hsl(44 88% 58%)" },
        transaction: { label: "Transaction", color: "hsl(292 66% 60%)" },
        dns_http: { label: "DNS+HTTP", color: "hsl(186 72% 50%)" },
    };

    const typeColors: Record<string, string> = Object.fromEntries(
        Object.entries(typeMeta).map(([type, meta]) => [type, meta.color]),
    );

    function formatTypeLabel(type: string): string {
        return (
            typeMeta[type]?.label ??
            type
                .replace(/_/g, " ")
                .replace(/\b\w/g, (char) => char.toUpperCase())
        );
    }

    const codeColors: Record<string, string> = {
        "2xx": "hsl(142 71% 45%)",
        "3xx": "hsl(217 91% 60%)",
        "4xx": "hsl(38 92% 50%)",
        "5xx": "hsl(0 84% 60%)",
        other: "hsl(215 15% 45%)",
    };
</script>

<svelte:head>
    <title>Analytics – updu</title>
</svelte:head>

<div class="stats-page-shell" data-testid="stats-page">
    <!-- System Health Bar (Level-1 SA: scannable in <5s) -->
    {#if loading}
        <div class="card stats-card"><Skeleton class="h-20 w-full" /></div>
    {:else if stats}
        {@const VIcon = verdict.icon}
        <section
            class="card stats-card flex flex-col gap-4 border sm:flex-row sm:items-center sm:justify-between {verdictRing[verdict.tone]}"
            aria-label="System health: {verdict.label}. {verdict.sub}."
        >
            <div class="flex items-center gap-4">
                <div
                    class="flex size-12 shrink-0 items-center justify-center rounded-xl border border-border/40 bg-surface/40"
                >
                    <VIcon class="size-6 {verdictIconClass[verdict.tone]}" />
                </div>
                <div>
                    <p
                        class="text-[10px] uppercase tracking-widest text-text-subtle"
                    >
                        System status
                    </p>
                    <p
                        class="text-lg font-bold leading-tight {verdictIconClass[verdict.tone]}"
                    >
                        {verdict.label}
                    </p>
                    <p class="text-xs text-text-muted">{verdict.sub}</p>
                </div>
            </div>
            <dl
                class="grid w-full grid-cols-2 gap-3 text-left sm:w-auto sm:grid-cols-3 sm:gap-8 sm:text-right"
                aria-label="Key operational metrics"
            >
                <div>
                    <dt
                        class="text-[10px] uppercase tracking-widest text-text-subtle"
                    >
                        Down
                    </dt>
                    <dd
                        class="font-mono text-2xl font-bold tabular-nums {downCount > 0 ? 'text-danger' : 'text-text'}"
                    >
                        {downCount}<span
                            class="text-sm text-text-subtle font-medium"
                            >/{monitorTotal}</span
                        >
                    </dd>
                </div>
                <div>
                    <dt
                        class="text-[10px] uppercase tracking-widest text-text-subtle"
                    >
                        Active incidents
                    </dt>
                    <dd
                        class="font-mono text-2xl font-bold tabular-nums {activeIncidents.length > 0 ? 'text-warning' : 'text-text'}"
                    >
                        {activeIncidents.length}
                    </dd>
                </div>
                <div>
                    <dt
                        class="text-[10px] uppercase tracking-widest text-text-subtle"
                    >
                        Avg latency 24h
                    </dt>
                    <dd class="font-mono text-2xl font-bold tabular-nums text-text">
                        {stats.summary?.avg_latency_24h != null
                            ? stats.summary.avg_latency_24h + "ms"
                            : "—"}
                    </dd>
                </div>
            </dl>
        </section>
    {/if}

    {#if loading}
        <div class="stats-grid-auto" style="--d-stats-min: min(100%, 12rem);">
            {#each Array(4) as _, index (index)}
                <div class="card stats-card"><Skeleton class="h-16 w-full" /></div>
            {/each}
        </div>
        <div class="card stats-card"><Skeleton class="h-48 w-full" /></div>
    {:else if error}
        <div class="card p-8 text-center text-danger">
            <TriangleAlert class="size-8 mx-auto mb-2 opacity-60" />
            <p>{error}</p>
        </div>
    {:else if stats}
        <!-- Persistent active-incident banner (visible on every tab when present) -->
        {#if activeIncidents.length > 0 && activeTab !== "incidents"}
            {@const top = activeIncidents[0]}
            {@const topSev = severityMeta[top.severity] ?? severityMeta.minor}
            <button
                type="button"
                class="card flex w-full items-center gap-3 border border-danger/40 bg-danger/10 px-4 py-3 text-left transition-colors hover:bg-danger/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger/60"
                onclick={() => activateTab("incidents")}
                aria-label="View {activeIncidents.length} active incident{activeIncidents.length === 1 ? '' : 's'}"
            >
                <TriangleAlert class="size-5 shrink-0 text-danger" />
                <div class="min-w-0 flex-1">
                    <p class="text-sm font-semibold text-danger">
                        {activeIncidents.length} active incident{activeIncidents.length === 1 ? "" : "s"}
                        {#if criticalIncidents > 0}
                            — {criticalIncidents} critical
                        {/if}
                    </p>
                    <p class="truncate text-xs text-text-muted">
                        Latest: <span class="text-text">{top.title}</span>
                        <span class="text-text-subtle">· {topSev.label} · started {relativeTime(top.started_at)}</span>
                    </p>
                </div>
                <ArrowRight class="size-4 shrink-0 text-text-muted" />
            </button>
        {/if}
        <!-- Tab strip -->
        <div
            class="stats-tabs flex min-h-11 items-center gap-1 overflow-x-auto border-b border-border py-1"
            role="tablist"
            aria-label="Analytics sections"
        >
            {#each tabs as t (t.value)}
                {@const Icon = t.icon}
                {@const isActive = activeTab === t.value}
                {@const badge =
                    t.value === "incidents"
                        ? activeIncidents.length || null
                        : t.value === "monitors"
                          ? stats.monitors?.length || null
                          : null}
                <button
                    type="button"
                    id={tabButtonId(t.value)}
                    role="tab"
                    aria-selected={isActive}
                    aria-controls={isActive ? tabPanelId(t.value) : undefined}
                    tabindex={isActive ? 0 : -1}
                    onclick={() => (activeTab = t.value)}
                    onkeydown={(event) => handleTabKeydown(event, t.value)}
                    class="relative -mb-px inline-flex min-h-10 shrink-0 items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 {isActive
                        ? 'border-primary text-primary'
                        : 'border-transparent text-text-muted hover:text-text'}"
                >
                    <Icon class="size-4" />
                    <span>{t.label}</span>
                    {#if badge}
                        <span
                            class="rounded-full px-1.5 py-0.5 text-[10px] font-semibold {t.value ===
                                'incidents' && badge > 0
                                ? 'bg-danger/15 text-danger'
                                : 'bg-surface-elevated text-text-muted'}"
                        >
                            {badge}
                        </span>
                    {/if}
                </button>
            {/each}
        </div>

        <!-- ─────────── OVERVIEW TAB ─────────── -->
        {#if activeTab === "overview"}
            <div
                id={tabPanelId("overview")}
                role="tabpanel"
                aria-labelledby={tabButtonId("overview")}
                tabindex="0"
                class="flex flex-col gap-[var(--d-section-gap)]"
            >
                <!-- Uptime Rings + Summary Cards -->
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 28rem);">
                <!-- Uptime Donuts -->
                <div
                    class="card stats-card grid gap-3"
                    style="grid-template-columns: repeat(auto-fit, minmax(min(7rem, 100%), 1fr));"
                >
                    {#each stats.global_uptime as u (u.label)}
                        <div class="flex flex-col items-center gap-2">
                            <StatusDonut
                                value={u.percent}
                                size="sm"
                                sublabel={u.label}
                            />
                        </div>
                    {/each}
                </div>

                <!-- Summary stat cards -->
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 9rem);">
                    <Stat
                        label="Checks (24h)"
                        value={stats.summary.total_checks_24h?.toLocaleString() ??
                            "0"}
                        icon={Zap}
                        tone="primary"
                    />
                    <Stat
                        label="Avg Latency"
                        value={stats.summary.avg_latency_24h != null
                            ? stats.summary.avg_latency_24h + "ms"
                            : "—"}
                        icon={Gauge}
                        tone={latencyTone(stats.summary.avg_latency_24h)}
                    />
                    <Stat
                        label="Monitors"
                        value={stats.summary.monitor_count ?? 0}
                        icon={Server}
                        tone="neutral"
                    />
                    <Stat
                        label="Active Incidents"
                        value={stats.summary.active_incidents ?? 0}
                        icon={TriangleAlert}
                        tone={stats.summary.active_incidents > 0
                            ? "danger"
                            : "neutral"}
                    />
                </div>
                </div>

                <!-- Hourly Timeline -->
                {#if stats.hourly_timeline?.length > 0}
                    <div class="card stats-card">
                    <div class="flex items-center justify-between mb-4">
                        <div class="flex items-center gap-2">
                            <Activity class="size-4 text-primary" />
                            <h2 class="text-sm font-semibold text-text">
                                Check Volume (24h)
                            </h2>
                        </div>
                        <div class="flex items-center gap-4 text-[10px]">
                            <span class="flex items-center gap-1.5">
                                <span
                                    class="size-2 rounded-full bg-success/70"
                                    aria-hidden="true"
                                ></span>
                                <span class="text-text-subtle">Up</span>
                            </span>
                            <span class="flex items-center gap-1.5">
                                <span
                                    class="size-2 rounded-sm border border-danger"
                                    style="background-image: {downStripe};"
                                    aria-hidden="true"
                                ></span>
                                <span class="text-text-subtle">Down</span>
                            </span>
                        </div>
                    </div>
                    <div
                        class="flex items-end gap-[3px]"
                        style="height: var(--d-chart-h-hourly);"
                        data-testid="stats-hourly-chart"
                    >
                        {#each stats.hourly_timeline as hour (hour.hour)}
                            {@const total = hour.up + hour.down}
                            {@const h = (total / maxHourly()) * 100}
                            {@const downH =
                                total > 0 ? (hour.down / total) * h : 0}
                            <div
                                class="flex-1 flex flex-col justify-end rounded-t-sm overflow-hidden hover:opacity-80 transition-opacity"
                                style="height: {h}%"
                                title="{formatHour(
                                    hour.hour,
                                )} — {hour.up} up, {hour.down} down"
                                aria-label="{formatHour(hour.hour)}: {hour.up} up, {hour.down} down"
                            >
                                {#if downH > 0}
                                    <div
                                        class="rounded-t-sm"
                                        style="height: {downH}%; background-image: {downStripe};"
                                    ></div>
                                {/if}
                                <div
                                    class="bg-success/60 flex-1 {downH === 0
                                        ? 'rounded-t-sm'
                                        : ''}"
                                ></div>
                            </div>
                        {/each}
                    </div>
                    <div
                        class="flex justify-between text-[9px] text-text-subtle mt-2 font-mono"
                    >
                        <span
                            >{formatHour(
                                stats.hourly_timeline[0]?.hour ?? "",
                            )}</span
                        >
                        {#if stats.hourly_timeline.length > 12}
                            <span
                                >{formatHour(
                                    stats.hourly_timeline[
                                        Math.floor(
                                            stats.hourly_timeline.length / 2,
                                        )
                                    ]?.hour ?? "",
                                )}</span
                            >
                        {/if}
                        <span
                            >{formatHour(
                                stats.hourly_timeline[
                                    stats.hourly_timeline.length - 1
                                ]?.hour ?? "",
                            )}</span
                        >
                    </div>
                    </div>
                {/if}
            </div>
        {/if}

        <!-- ─────────── PERFORMANCE TAB ─────────── -->
        {#if activeTab === "performance"}
            <div
                id={tabPanelId("performance")}
                role="tabpanel"
                aria-labelledby={tabButtonId("performance")}
                tabindex="0"
                class="flex flex-col gap-[var(--d-section-gap)]"
            >
                <!-- Latency Distribution + Type/Code Breakdown -->
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 24rem);">
                <!-- Latency Distribution -->
                {#if stats.latency_distribution?.length > 0}
                    <div class="card stats-card">
                        <div class="flex items-center gap-2 mb-4">
                            <Clock class="size-4 text-primary" />
                            <h2 class="text-sm font-semibold text-text">
                                Latency Distribution (24h)
                            </h2>
                        </div>
                        <div class="space-y-2.5">
                            {#each stats.latency_distribution as bucket (bucket.label)}
                                {@const w =
                                    (bucket.count / maxLatDist()) * 100}
                                {@const bTone = latencyBucketTone(bucket.label)}
                                <div>
                                    <div
                                        class="flex items-center justify-between text-xs mb-1"
                                    >
                                        <span
                                            class="text-text-muted font-mono text-[11px]"
                                            >{bucket.label}</span
                                        >
                                        <span
                                            class="text-text-subtle font-mono tabular-nums text-[11px]"
                                            >{bucket.count.toLocaleString()}</span
                                        >
                                    </div>
                                    <div
                                        class="rounded-full bg-surface-elevated overflow-hidden"
                                        style="height: var(--d-chart-h-bar);"
                                    >
                                        <div
                                            class="h-full rounded-full transition-all duration-500 ease-out"
                                            style="width: {w}%; background-color: {toneBarColor(bTone)}; opacity: 0.8;"
                                        ></div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                <!-- Type + Code Breakdown -->
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 20rem);">
                    <!-- Type breakdown: horizontal stacked bar -->
                    {#if stats.type_breakdown?.length > 0}
                        {@const typeTotal = totalOfArray(stats.type_breakdown, "count")}
                        <div class="card stats-card">
                            <div class="flex items-center justify-between mb-3">
                                <h2
                                    class="text-sm font-semibold text-text flex items-center gap-2"
                                >
                                    <Server class="size-4 text-text-muted" /> Checks by Type
                                </h2>
                                <span
                                    class="text-[10px] uppercase tracking-widest text-text-subtle font-mono tabular-nums"
                                >
                                    {typeTotal.toLocaleString()} total
                                </span>
                            </div>
                            <div
                                class="flex h-6 w-full overflow-hidden rounded-md border border-border bg-surface-elevated"
                                role="img"
                                aria-label="Check type distribution across {stats.type_breakdown.length} types"
                            >
                                {#each stats.type_breakdown as t (t.type)}
                                    {@const w = typeTotal > 0 ? (t.count / typeTotal) * 100 : 0}
                                    <div
                                        class="h-full transition-all duration-500 ease-out"
                                        style="width: {w}%; background: {typeColors[t.type] || 'hsl(215 15% 45%)'};"
                                        title="{formatTypeLabel(t.type)}: {t.count.toLocaleString()} ({w.toFixed(1)}%)"
                                    ></div>
                                {/each}
                            </div>
                            <ul
                                class="mt-3 grid gap-x-4 gap-y-1.5"
                                style="grid-template-columns: repeat(auto-fit, minmax(min(9rem, 100%), 1fr));"
                            >
                                {#each stats.type_breakdown as t (t.type)}
                                    {@const w = typeTotal > 0 ? (t.count / typeTotal) * 100 : 0}
                                    <li
                                        class="flex items-center justify-between gap-2 text-xs"
                                    >
                                        <span
                                            class="flex min-w-0 items-center gap-2 text-text-muted"
                                        >
                                            <span
                                                class="size-2.5 shrink-0 rounded-sm"
                                                style="background: {typeColors[t.type] || 'hsl(215 15% 45%)'};"
                                            ></span>
                                            <span
                                                class="truncate uppercase tracking-wider font-bold text-[10px]"
                                                >{formatTypeLabel(t.type)}</span
                                            >
                                        </span>
                                        <span
                                            class="shrink-0 font-mono tabular-nums text-text"
                                        >
                                            {t.count.toLocaleString()}
                                            <span
                                                class="ml-1 text-text-subtle text-[10px]"
                                                >{w.toFixed(0)}%</span
                                            >
                                        </span>
                                    </li>
                                {/each}
                            </ul>
                        </div>
                    {/if}

                    <!-- Response codes: horizontal stacked bar (red/amber pop pre-attentively) -->
                    {#if stats.response_codes?.length > 0}
                        {@const codeTotal = totalOfArray(stats.response_codes, "count")}
                        <div class="card stats-card">
                            <div class="flex items-center justify-between mb-3">
                                <h2
                                    class="text-sm font-semibold text-text flex items-center gap-2"
                                >
                                    <TrendingUp class="size-4 text-text-muted" />
                                    HTTP Status Codes
                                </h2>
                                <span
                                    class="text-[10px] uppercase tracking-widest text-text-subtle font-mono tabular-nums"
                                >
                                    {codeTotal.toLocaleString()} responses
                                </span>
                            </div>
                            <div
                                class="flex h-6 w-full overflow-hidden rounded-md border border-border bg-surface-elevated"
                                role="img"
                                aria-label="HTTP response code distribution"
                            >
                                {#each stats.response_codes as c (c.code)}
                                    {@const w = codeTotal > 0 ? (c.count / codeTotal) * 100 : 0}
                                    <div
                                        class="h-full transition-all duration-500 ease-out"
                                        style="width: {w}%; background: {codeColors[c.code] || 'hsl(215 15% 45%)'};"
                                        title="{c.code}: {c.count.toLocaleString()} ({w.toFixed(1)}%)"
                                    ></div>
                                {/each}
                            </div>
                            <ul
                                class="mt-3 grid gap-x-4 gap-y-1.5"
                                style="grid-template-columns: repeat(auto-fit, minmax(min(8rem, 100%), 1fr));"
                            >
                                {#each stats.response_codes as c (c.code)}
                                    {@const w = codeTotal > 0 ? (c.count / codeTotal) * 100 : 0}
                                    <li
                                        class="flex items-center justify-between gap-2 text-xs"
                                    >
                                        <span
                                            class="flex items-center gap-2 text-text-muted"
                                        >
                                            <span
                                                class="size-2.5 shrink-0 rounded-sm"
                                                style="background: {codeColors[c.code] || 'hsl(215 15% 45%)'};"
                                            ></span>
                                            <span
                                                class="font-mono font-bold text-[11px]"
                                                >{c.code}</span
                                            >
                                        </span>
                                        <span
                                            class="shrink-0 font-mono tabular-nums text-text"
                                        >
                                            {c.count.toLocaleString()}
                                            <span
                                                class="ml-1 text-text-subtle text-[10px]"
                                                >{w.toFixed(0)}%</span
                                            >
                                        </span>
                                    </li>
                                {/each}
                            </ul>
                        </div>
                    {/if}
                </div>
                </div>

                <!-- Latency leaderboard with fleet-median reference (bullet style) -->
                {#if stats.monitors?.length > 0}
                {@const byP95 = [...stats.monitors]
                    .filter((m: any) => m.p95_latency != null)
                    .sort(
                        (a: any, b: any) =>
                            (b.p95_latency ?? 0) - (a.p95_latency ?? 0),
                    )
                    .slice(0, 5)}
                {@const byAvg = [...stats.monitors]
                    .filter((m: any) => m.avg_latency != null)
                    .sort(
                        (a: any, b: any) =>
                            (a.avg_latency ?? 0) - (b.avg_latency ?? 0),
                    )
                    .slice(0, 5)}
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 26rem);">
                    <div class="card stats-card">
                        <div class="flex items-center justify-between mb-3">
                            <h2
                                class="text-sm font-semibold text-text flex items-center gap-2"
                            >
                                <TriangleAlert class="size-4 text-warning" /> Slowest
                                (P95, 24h)
                            </h2>
                            {#if fleetP95.median != null}
                                <span
                                    class="text-[10px] uppercase tracking-widest text-text-subtle font-mono tabular-nums"
                                    title="Median P95 across the fleet"
                                >
                                    fleet median {fleetP95.median}ms
                                </span>
                            {/if}
                        </div>
                        {#if byP95.length === 0}
                            <p class="text-xs text-text-subtle">
                                No latency data yet.
                            </p>
                        {:else}
                            <ul class="space-y-2.5">
                                {#each byP95 as m (m.id)}
                                    {@const w = (m.p95_latency / fleetP95.max) * 100}
                                    {@const medianPct = fleetP95.median != null
                                        ? (fleetP95.median / fleetP95.max) * 100
                                        : null}
                                    <li class="space-y-1">
                                        <div
                                            class="flex items-center justify-between text-xs gap-3"
                                        >
                                            <button
                                                type="button"
                                                class="truncate text-left text-text hover:text-primary transition-colors"
                                                onclick={() =>
                                                    goto(
                                                        resolve("/monitors/[id]", {
                                                            id: m.id,
                                                        }),
                                                    )}
                                            >
                                                {m.name}
                                            </button>
                                            <span
                                                class="font-mono tabular-nums text-warning shrink-0"
                                                >{m.p95_latency}ms</span
                                            >
                                        </div>
                                        <div
                                            class="relative h-1.5 rounded-full bg-surface-elevated overflow-visible"
                                            aria-hidden="true"
                                        >
                                            <div
                                                class="h-full rounded-full bg-warning/70"
                                                style="width: {w}%"
                                            ></div>
                                            {#if medianPct != null}
                                                <div
                                                    class="absolute -top-0.5 h-2.5 w-px bg-text/70"
                                                    style="left: {medianPct}%"
                                                    title="Fleet median P95: {fleetP95.median}ms"
                                                ></div>
                                            {/if}
                                        </div>
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                    </div>
                    <div class="card stats-card">
                        <div class="flex items-center justify-between mb-3">
                            <h2
                                class="text-sm font-semibold text-text flex items-center gap-2"
                            >
                                <Zap class="size-4 text-success" /> Fastest
                                (Avg, 24h)
                            </h2>
                            {#if fleetAvg.median != null}
                                <span
                                    class="text-[10px] uppercase tracking-widest text-text-subtle font-mono tabular-nums"
                                    title="Median average latency across the fleet"
                                >
                                    fleet median {fleetAvg.median}ms
                                </span>
                            {/if}
                        </div>
                        {#if byAvg.length === 0}
                            <p class="text-xs text-text-subtle">
                                No latency data yet.
                            </p>
                        {:else}
                            <ul class="space-y-2.5">
                                {#each byAvg as m (m.id)}
                                    {@const w = (m.avg_latency / fleetAvg.max) * 100}
                                    {@const medianPct = fleetAvg.median != null
                                        ? (fleetAvg.median / fleetAvg.max) * 100
                                        : null}
                                    <li class="space-y-1">
                                        <div
                                            class="flex items-center justify-between text-xs gap-3"
                                        >
                                            <button
                                                type="button"
                                                class="truncate text-left text-text hover:text-primary transition-colors"
                                                onclick={() =>
                                                    goto(
                                                        resolve("/monitors/[id]", {
                                                            id: m.id,
                                                        }),
                                                    )}
                                            >
                                                {m.name}
                                            </button>
                                            <span
                                                class="font-mono tabular-nums text-success shrink-0"
                                                >{m.avg_latency}ms</span
                                            >
                                        </div>
                                        <div
                                            class="relative h-1.5 rounded-full bg-surface-elevated overflow-visible"
                                            aria-hidden="true"
                                        >
                                            <div
                                                class="h-full rounded-full bg-success/70"
                                                style="width: {w}%"
                                            ></div>
                                            {#if medianPct != null}
                                                <div
                                                    class="absolute -top-0.5 h-2.5 w-px bg-text/70"
                                                    style="left: {medianPct}%"
                                                    title="Fleet median avg: {fleetAvg.median}ms"
                                                ></div>
                                            {/if}
                                        </div>
                                    </li>
                                {/each}
                            </ul>
                        {/if}
                    </div>
                </div>
                {/if}
            </div>
        {/if}

        <!-- ─────────── MONITORS TAB (Sortable Leaderboard) ─────────── -->
        {#if activeTab === "monitors"}
            <div
                id={tabPanelId("monitors")}
                role="tabpanel"
                aria-labelledby={tabButtonId("monitors")}
                tabindex="0"
            >
                {#if !stats.monitors?.length}
                <div class="card p-8 text-center text-text-muted">
                    <Server class="size-8 mx-auto mb-2 opacity-60" />
                    <p>No monitors yet.</p>
                </div>
            {:else}
                <div class="card overflow-hidden" style="padding: 0;">
                    <!-- Toolbar -->
                    <div
                        class="flex flex-col gap-3 border-b border-border bg-surface/30 px-4 py-3.5 sm:flex-row sm:items-center sm:justify-between sm:px-5"
                    >
                        <div class="flex items-center gap-2">
                            <TrendingUp class="size-4 text-primary" />
                            <h2 class="text-sm font-semibold text-text">
                                Monitor Leaderboard
                            </h2>
                            <span
                                class="text-[10px] text-text-subtle uppercase tracking-wider"
                            >
                                24h • {sortedMonitors.length} of {stats.monitors
                                    .length}
                            </span>
                        </div>
                        <div class="flex w-full min-w-0 flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
                            <div class="relative min-w-0 flex-1 sm:flex-none">
                                <Search
                                    class="size-3.5 text-text-subtle absolute left-2.5 top-1/2 -translate-y-1/2 pointer-events-none"
                                />
                                <input
                                    type="text"
                                    bind:value={monitorFilter}
                                    aria-label="Filter monitors"
                                    placeholder="Filter monitors…"
                                    class="w-full rounded-md border border-border bg-surface-elevated py-1.5 pl-8 pr-3 text-xs text-text placeholder:text-text-subtle focus:outline-none focus:border-primary/60 sm:w-56"
                                />
                            </div>
                            <div
                                class="inline-flex w-full items-center gap-0.5 overflow-x-auto rounded-md border border-border bg-surface-elevated p-0.5 sm:w-auto"
                            >
                                {#each ["all", "up", "down", "paused", "pending"] as s (s)}
                                    <button
                                        type="button"
                                        onclick={() =>
                                            (statusFilter = s as any)}
                                        aria-pressed={statusFilter === s}
                                        class="min-h-8 shrink-0 px-2 py-1 text-[10px] uppercase font-bold tracking-wider rounded transition-colors {statusFilter ===
                                        s
                                            ? 'bg-primary/15 text-primary'
                                            : 'text-text-muted hover:text-text'}"
                                    >
                                        {s}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    </div>
                    <div class="overflow-x-auto">
                        <table class="w-full text-xs">
                            <thead>
                                <tr
                                    class="text-[10px] text-text-subtle uppercase tracking-wider border-b border-border bg-surface/40"
                                >
                                    {@render sortableTh(
                                        "name",
                                        "Monitor",
                                        "left",
                                        "px-3 sm:px-5",
                                    )}
                                    {@render sortableTh(
                                        "status",
                                        "Status",
                                        "left",
                                        "px-3",
                                    )}
                                    {@render sortableTh(
                                        "uptime_24h",
                                        "Uptime",
                                        "right",
                                        "px-3",
                                    )}
                                    {@render sortableTh(
                                        "avg_latency",
                                        "Avg",
                                        "right",
                                        "px-3",
                                    )}
                                    {@render sortableTh(
                                        "p95_latency",
                                        "P95",
                                        "right",
                                        "px-3",
                                        "hidden sm:table-cell",
                                    )}
                                    {@render sortableTh(
                                        "min_latency",
                                        "Min",
                                        "right",
                                        "px-3",
                                        "hidden lg:table-cell",
                                    )}
                                    {@render sortableTh(
                                        "max_latency",
                                        "Max",
                                        "right",
                                        "px-3",
                                        "hidden lg:table-cell",
                                    )}
                                    {@render sortableTh(
                                        "total_checks",
                                        "Checks",
                                        "right",
                                        "px-3 sm:px-5",
                                        "hidden sm:table-cell",
                                    )}
                                </tr>
                            </thead>
                            <tbody>
                                {#if sortedMonitors.length === 0}
                                    <tr>
                                        <td
                                            colspan="8"
                                            class="px-5 py-10 text-center text-text-subtle"
                                        >
                                            No monitors match the current
                                            filters.
                                        </td>
                                    </tr>
                                {/if}
                                {#each sortedMonitors as m, idx (m.id)}
                                    {@const SIcon = statusIcon(m.status)}
                                    <tr
                                        class="border-b border-border/50 hover:bg-surface-elevated/50 transition-colors group"
                                    >
                                        <td class="px-3 py-3 sm:px-5 2xl:py-3.5">
                                            <div
                                                class="flex min-w-0 items-center gap-2.5"
                                            >
                                                <span
                                                    class="text-[10px] font-mono text-text-subtle w-5 text-right"
                                                    >#{idx + 1}</span
                                                >
                                                <div class="min-w-0">
                                                    <a
                                                        href={resolve("/monitors/[id]", {
                                                            id: m.id,
                                                        })}
                                                        class="truncate font-medium text-text transition-colors group-hover:text-primary 2xl:text-sm"
                                                    >
                                                        {m.name}
                                                    </a>
                                                    <p
                                                        class="text-[10px] text-text-subtle"
                                                    >
                                                        {formatTypeLabel(
                                                            m.type,
                                                        )}{#if m.group}
                                                            • {m.group}{/if}
                                                    </p>
                                                    <p
                                                        class="font-mono text-[10px] tabular-nums text-text-subtle sm:hidden"
                                                    >
                                                        P95 {m.p95_latency != null
                                                            ? m.p95_latency + "ms"
                                                            : "—"} · {m.total_checks.toLocaleString()} checks
                                                    </p>
                                                </div>
                                            </div>
                                        </td>
                                        <td class="px-3 py-3 2xl:py-3.5">
                                            <span
                                                class="inline-flex items-center gap-1.5 {statusTextClass(m.status)}"
                                                aria-label="Status: {statusLabel(m.status)}"
                                            >
                                                <SIcon
                                                    class="size-3.5"
                                                    aria-hidden="true"
                                                />
                                                <span
                                                    class="uppercase font-bold tracking-wider text-[10px]"
                                                    >{statusLabel(m.status)}</span
                                                >
                                            </span>
                                        </td>
                                        <td class="px-3 py-3 text-right">
                                            <div
                                                class="flex items-center gap-2 justify-end"
                                            >
                                                <div
                                                    class="hidden w-16 h-1.5 rounded-full bg-surface-elevated overflow-hidden md:block"
                                                >
                                                    <div
                                                        class="h-full rounded-full"
                                                        style="width: {pct(
                                                            m.uptime_24h,
                                                        )}%; background: var({uptimeBarVar(
                                                            m.uptime_24h,
                                                        )});"
                                                    ></div>
                                                </div>
                                                <span
                                                    class="font-mono font-bold tabular-nums {uptimeTextClass(m.uptime_24h)}"
                                                >
                                                    {m.uptime_24h.toFixed(4)}%
                                                </span>
                                            </div>
                                        </td>
                                        <td
                                            class="px-3 py-3 text-right font-mono tabular-nums text-text-muted 2xl:py-3.5"
                                        >
                                            {m.avg_latency != null
                                                ? m.avg_latency + "ms"
                                                : "—"}
                                        </td>
                                        <td
                                            class="hidden px-3 py-3 text-right font-mono tabular-nums text-text-muted sm:table-cell 2xl:py-3.5"
                                        >
                                            {m.p95_latency != null
                                                ? m.p95_latency + "ms"
                                                : "—"}
                                        </td>
                                        <td
                                            class="hidden px-3 py-3 text-right font-mono tabular-nums text-text-subtle lg:table-cell 2xl:py-3.5"
                                        >
                                            {m.min_latency != null
                                                ? m.min_latency + "ms"
                                                : "—"}
                                        </td>
                                        <td
                                            class="hidden px-3 py-3 text-right font-mono tabular-nums text-text-subtle lg:table-cell 2xl:py-3.5"
                                        >
                                            {m.max_latency != null
                                                ? m.max_latency + "ms"
                                                : "—"}
                                        </td>
                                        <td
                                            class="hidden px-3 py-3 text-right font-mono tabular-nums text-text-muted sm:table-cell sm:px-5 2xl:py-3.5"
                                        >
                                            {m.total_checks.toLocaleString()}
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </div>
                {/if}
            </div>
        {/if}

        <!-- ─────────── INCIDENTS TAB ─────────── -->
        {#if activeTab === "incidents"}
            <div
                id={tabPanelId("incidents")}
                role="tabpanel"
                aria-labelledby={tabButtonId("incidents")}
                tabindex="0"
                class="flex flex-col gap-[var(--d-section-gap)]"
            >
                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 14rem);">
                <Stat
                    label="Active"
                    value={activeIncidents.length}
                    icon={TriangleAlert}
                    tone={activeIncidents.length > 0 ? "danger" : "neutral"}
                />
                <Stat
                    label="Critical"
                    value={criticalIncidents}
                    icon={AlertCircle}
                    tone={criticalIncidents > 0 ? "danger" : "neutral"}
                />
                <Stat
                    label="Resolved (recent)"
                    value={resolvedIncidents.length}
                    icon={CheckCircle2}
                    tone="neutral"
                />
                </div>

                <div class="stats-grid-auto" style="--d-stats-min: min(100%, 34rem);">
                <!-- Active incidents -->
                <div class="card overflow-hidden" style="padding: 0;">
                    <div
                        class="px-5 py-3.5 border-b border-border bg-surface/30 flex items-center justify-between"
                    >
                        <div class="flex items-center gap-2">
                            <TriangleAlert class="size-4 text-danger" />
                            <h2 class="text-sm font-semibold text-text">
                                Active Incidents
                            </h2>
                        </div>
                        <button
                            type="button"
                            class="text-[10px] text-primary uppercase tracking-wider hover:underline"
                            onclick={() => goto(resolve("/incidents"))}
                        >
                            Manage →
                        </button>
                    </div>
                    {#if activeIncidents.length === 0}
                        <div class="p-8 text-center text-text-muted">
                            <CheckCircle2
                                class="size-8 mx-auto mb-2 text-success/70"
                            />
                            <p class="text-sm">All systems operational.</p>
                        </div>
                    {:else}
                        <ul class="divide-y divide-border/60">
                            {#each activeIncidents as inc (inc.id)}
                                {@const sev =
                                    severityMeta[inc.severity] ??
                                    severityMeta.minor}
                                {@const st =
                                    incidentStatusMeta[inc.status] ?? {
                                        label: inc.status,
                                        tone: "text-text-muted",
                                    }}
                                <li class="px-5 py-3 flex items-start gap-4">
                                    <span
                                        class="mt-0.5 inline-flex items-center justify-center px-2 py-0.5 rounded text-[10px] uppercase font-bold tracking-wider border {sev.ring} {sev.tone}"
                                    >
                                        {sev.label}
                                    </span>
                                    <div class="min-w-0 flex-1">
                                        <p class="text-sm font-medium text-text">
                                            {inc.title}
                                        </p>
                                        {#if inc.description}
                                            <p
                                                class="text-xs text-text-muted mt-0.5 line-clamp-2"
                                            >
                                                {inc.description}
                                            </p>
                                        {/if}
                                        <p
                                            class="text-[10px] text-text-subtle mt-1 flex flex-wrap items-center gap-x-3 gap-y-1"
                                        >
                                            <span class={st.tone}>{st.label}</span>
                                            <span>
                                                Started {relativeTime(
                                                    inc.started_at,
                                                )}
                                            </span>
                                            {#if inc.monitor_ids?.length}
                                                <span
                                                    >• {inc.monitor_ids.length} monitor{inc
                                                        .monitor_ids.length === 1
                                                        ? ""
                                                        : "s"}</span
                                                >
                                            {/if}
                                        </p>
                                    </div>
                                </li>
                            {/each}
                        </ul>
                    {/if}
                </div>

                <!-- Recently resolved -->
                {#if resolvedIncidents.length > 0}
                    <div class="card overflow-hidden" style="padding: 0;">
                        <div
                            class="px-5 py-3.5 border-b border-border bg-surface/30 flex items-center gap-2"
                        >
                            <CheckCircle2 class="size-4 text-success" />
                            <h2 class="text-sm font-semibold text-text">
                                Recently Resolved
                            </h2>
                        </div>
                        <ul class="divide-y divide-border/60">
                            {#each resolvedIncidents as inc (inc.id)}
                                {@const sev =
                                    severityMeta[inc.severity] ??
                                    severityMeta.minor}
                                <li
                                    class="px-5 py-2.5 flex items-center gap-3 text-xs"
                                >
                                    <span
                                        class="inline-flex items-center justify-center px-1.5 py-0.5 rounded text-[10px] uppercase font-bold tracking-wider border border-border bg-surface-elevated text-text-muted"
                                    >
                                        {sev.label}
                                    </span>
                                    <span class="flex-1 truncate text-text"
                                        >{inc.title}</span
                                    >
                                    <span class="text-text-subtle"
                                        >resolved {relativeTime(
                                            inc.resolved_at,
                                        )}</span
                                    >
                                </li>
                            {/each}
                        </ul>
                    </div>
                {/if}
                </div>
            </div>
        {/if}
    {/if}
</div>

{#snippet sortableTh(
    key: SortKey,
    label: string,
    align: "left" | "right",
    pad: string,
    visibility: string = "",
)}
    {@const isActive = sortKey === key}
    <th
        class="{visibility} {pad} py-2.5 font-medium text-{align}"
        aria-sort={isActive
            ? sortDir === "asc"
                ? "ascending"
                : "descending"
            : "none"}
    >
        <button
            type="button"
            onclick={() => toggleSort(key)}
            class="inline-flex items-center gap-1 uppercase tracking-wider transition-colors hover:text-text {isActive
                ? 'text-primary'
                : ''} {align === 'right' ? 'flex-row-reverse' : ''}"
        >
            <span>{label}</span>
            {#if isActive}
                {#if sortDir === "asc"}
                    <ArrowUp class="size-3" />
                {:else}
                    <ArrowDown class="size-3" />
                {/if}
            {:else}
                <ArrowUpDown class="size-3 opacity-40" />
            {/if}
        </button>
    </th>
{/snippet}

<style>
    @keyframes ring-in {
        from {
            transform: scale(0.7) rotate(-90deg);
            opacity: 0;
        }
        to {
            transform: scale(1) rotate(0deg);
            opacity: 1;
        }
    }
</style>
