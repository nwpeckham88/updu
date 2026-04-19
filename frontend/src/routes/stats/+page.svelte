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
    } from "lucide-svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Stat from "$lib/components/ui/stat.svelte";
    import StatusDonut from "$lib/components/charts/status-donut.svelte";
    import { goto } from "$app/navigation";
    import { format, parseISO } from "date-fns";

    let stats = $state<any>(null);
    let loading = $state(true);
    let error = $state("");

    onMount(async () => {
        try {
            stats = await fetchAPI("/api/v1/stats");
        } catch (e: any) {
            error = e.message || "Failed to load analytics";
        } finally {
            loading = false;
        }
    });

    function pct(n: number) {
        return Math.min(100, Math.max(0, n));
    }

    function ringGradient(value: number, color: string): string {
        return `conic-gradient(${color} ${value * 3.6}deg, hsl(215 28% 17% / 0.3) ${value * 3.6}deg)`;
    }

    function ringColor(p: number): string {
        if (p >= 99) return "hsl(142 71% 45%)";
        if (p >= 95) return "hsl(38 92% 50%)";
        return "hsl(0 84% 60%)";
    }

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

    function donutGradient(
        items: any[],
        countKey: string,
        colorMap: Record<string, string>,
    ): string {
        const total = totalOfArray(items, countKey);
        if (total === 0) return "hsl(215 28% 17%)";
        let angle = 0;
        const stops: string[] = [];
        for (const item of items) {
            const key = item.type || item.code || "other";
            const color = colorMap[key] || "hsl(215 15% 45%)";
            const size = (item[countKey] / total) * 360;
            stops.push(`${color} ${angle}deg ${angle + size}deg`);
            angle += size;
        }
        return `conic-gradient(${stops.join(", ")})`;
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

<div class="space-y-6 pb-8">
    <!-- Hero Header -->
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
            <div
                class="size-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center"
            >
                <BarChart3 class="size-5 text-primary" />
            </div>
            <div>
                <h1 class="text-xl font-bold text-text">System Analytics</h1>
                <p class="text-xs text-text-muted mt-0.5">
                    Real-time monitoring intelligence
                </p>
            </div>
        </div>
        {#if stats}
            <div class="text-right">
                <p class="text-3xl font-black font-mono tabular-nums text-text">
                    {stats.summary?.total_checks_all?.toLocaleString() ?? "—"}
                </p>
                <p
                    class="text-[10px] text-text-subtle uppercase tracking-widest"
                >
                    lifetime checks
                </p>
            </div>
        {/if}
    </div>

    {#if loading}
        <div class="grid grid-cols-4 gap-4">
            {#each Array(4) as _}
                <div class="card p-5"><Skeleton class="h-16 w-full" /></div>
            {/each}
        </div>
        <div class="card p-6"><Skeleton class="h-48 w-full" /></div>
    {:else if error}
        <div class="card p-8 text-center text-danger">
            <TriangleAlert class="size-8 mx-auto mb-2 opacity-60" />
            <p>{error}</p>
        </div>
    {:else if stats}
        <!-- Uptime Rings + Summary Cards -->
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
            <!-- Uptime Donuts -->
            <div
                class="lg:col-span-5 card p-6 flex items-center justify-around gap-4 flex-wrap"
            >
                {#each stats.global_uptime as u}
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
            <div class="lg:col-span-7 grid grid-cols-2 gap-3">
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
                    tone="success"
                />
                <Stat
                    label="Monitors"
                    value={stats.summary.monitor_count ?? 0}
                    icon={Server}
                    tone="warning"
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
            <div class="card p-5">
                <div class="flex items-center justify-between mb-4">
                    <div class="flex items-center gap-2">
                        <Activity class="size-4 text-primary" />
                        <h2 class="text-sm font-semibold text-text">
                            Check Volume (24h)
                        </h2>
                    </div>
                    <div class="flex items-center gap-4 text-[10px]">
                        <span class="flex items-center gap-1.5">
                            <span class="size-2 rounded-full bg-success/70"
                            ></span>
                            <span class="text-text-subtle">Up</span>
                        </span>
                        <span class="flex items-center gap-1.5">
                            <span class="size-2 rounded-full bg-danger/80"
                            ></span>
                            <span class="text-text-subtle">Down</span>
                        </span>
                    </div>
                </div>
                <div class="flex items-end gap-[3px] h-32">
                    {#each stats.hourly_timeline as hour}
                        {@const total = hour.up + hour.down}
                        {@const h = (total / maxHourly()) * 100}
                        {@const downH = total > 0 ? (hour.down / total) * h : 0}
                        <div
                            class="flex-1 flex flex-col justify-end rounded-t-sm overflow-hidden hover:opacity-80 transition-opacity"
                            style="height: {h}%"
                            title="{formatHour(
                                hour.hour,
                            )} — {hour.up} up, {hour.down} down"
                        >
                            {#if downH > 0}
                                <div
                                    class="bg-danger/80 rounded-t-sm"
                                    style="height: {downH}%"
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
                                    Math.floor(stats.hourly_timeline.length / 2)
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

        <!-- Latency Distribution + Type/Code Breakdown -->
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
            <!-- Latency Distribution -->
            {#if stats.latency_distribution?.length > 0}
                <div class="lg:col-span-5 card p-5">
                    <div class="flex items-center gap-2 mb-4">
                        <Clock class="size-4 text-primary" />
                        <h2 class="text-sm font-semibold text-text">
                            Latency Distribution
                        </h2>
                    </div>
                    <div class="space-y-2.5">
                        {#each stats.latency_distribution as bucket}
                            {@const w = (bucket.count / maxLatDist()) * 100}
                            <div>
                                <div
                                    class="flex items-center justify-between text-xs mb-1"
                                >
                                    <span
                                        class="text-text-muted font-mono text-[11px]"
                                        >{bucket.label}</span
                                    >
                                    <span
                                        class="text-text-subtle font-mono text-[11px]"
                                        >{bucket.count.toLocaleString()}</span
                                    >
                                </div>
                                <div
                                    class="h-3 rounded-full bg-surface-elevated overflow-hidden"
                                >
                                    <div
                                        class="h-full rounded-full bg-primary/70 transition-all duration-500 ease-out"
                                        style="width: {w}%;"
                                    ></div>
                                </div>
                            </div>
                        {/each}
                    </div>
                </div>
            {/if}

            <!-- Type + Code Breakdown -->
            <div class="lg:col-span-7 grid grid-cols-2 gap-4">
                <!-- Type breakdown donut -->
                {#if stats.type_breakdown?.length > 0}
                    <div class="card p-5">
                        <h2
                            class="text-sm font-semibold text-text mb-4 flex items-center gap-2"
                        >
                            <Server class="size-4 text-text-muted" /> By Type
                        </h2>
                        <div class="flex justify-center mb-4">
                            <div
                                class="size-20 rounded-full relative"
                                style="background: {donutGradient(
                                    stats.type_breakdown,
                                    'count',
                                    typeColors,
                                )};"
                            >
                                <div
                                    class="absolute inset-2.5 rounded-full bg-surface"
                                ></div>
                            </div>
                        </div>
                        <div class="space-y-1.5">
                            {#each stats.type_breakdown as t}
                                <div
                                    class="flex items-center justify-between text-xs"
                                >
                                    <span
                                        class="flex items-center gap-2 text-text-muted"
                                    >
                                        <span
                                            class="size-2 rounded-full"
                                            style="background: {typeColors[
                                                t.type
                                            ] || 'hsl(215 15% 45%)'};"
                                        ></span>
                                        <span
                                            class="uppercase font-bold tracking-wider text-[10px]"
                                            >{formatTypeLabel(t.type)}</span
                                        >
                                    </span>
                                    <span class="font-mono text-text"
                                        >{t.count}</span
                                    >
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                <!-- Response codes donut -->
                {#if stats.response_codes?.length > 0}
                    <div class="card p-5">
                        <h2
                            class="text-sm font-semibold text-text mb-4 flex items-center gap-2"
                        >
                            <TrendingUp class="size-4 text-text-muted" /> Status
                            Codes
                        </h2>
                        <div class="flex justify-center mb-4">
                            <div
                                class="size-20 rounded-full relative"
                                style="background: {donutGradient(
                                    stats.response_codes,
                                    'count',
                                    codeColors,
                                )};"
                            >
                                <div
                                    class="absolute inset-2.5 rounded-full bg-surface"
                                ></div>
                            </div>
                        </div>
                        <div class="space-y-1.5">
                            {#each stats.response_codes as c}
                                <div
                                    class="flex items-center justify-between text-xs"
                                >
                                    <span
                                        class="flex items-center gap-2 text-text-muted"
                                    >
                                        <span
                                            class="size-2 rounded-full"
                                            style="background: {codeColors[
                                                c.code
                                            ] || 'hsl(215 15% 45%)'};"
                                        ></span>
                                        <span
                                            class="font-mono font-bold text-[11px]"
                                            >{c.code}</span
                                        >
                                    </span>
                                    <span class="font-mono text-text"
                                        >{c.count.toLocaleString()}</span
                                    >
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Monitor Leaderboard -->
        {#if stats.monitors?.length > 0}
            <div class="card overflow-hidden" style="padding: 0;">
                <div
                    class="px-5 py-3.5 border-b border-border bg-surface/30 flex items-center justify-between"
                >
                    <div class="flex items-center gap-2">
                        <TrendingUp class="size-4 text-primary" />
                        <h2 class="text-sm font-semibold text-text">
                            Monitor Leaderboard
                        </h2>
                    </div>
                    <span
                        class="text-[10px] text-text-subtle uppercase tracking-wider"
                        >24h window • sorted by uptime</span
                    >
                </div>
                <div class="overflow-x-auto">
                    <table class="w-full text-xs">
                        <thead>
                            <tr
                                class="text-[10px] text-text-subtle uppercase tracking-wider border-b border-border"
                            >
                                <th class="text-left px-5 py-2.5 font-medium"
                                    >Monitor</th
                                >
                                <th class="text-left px-3 py-2.5 font-medium"
                                    >Status</th
                                >
                                <th class="text-right px-3 py-2.5 font-medium"
                                    >Uptime</th
                                >
                                <th class="text-right px-3 py-2.5 font-medium"
                                    >Avg</th
                                >
                                <th class="text-right px-3 py-2.5 font-medium"
                                    >P95</th
                                >
                                <th class="text-right px-3 py-2.5 font-medium"
                                    >Min</th
                                >
                                <th class="text-right px-3 py-2.5 font-medium"
                                    >Max</th
                                >
                                <th class="px-5 py-2.5 font-medium text-right"
                                    >Checks</th
                                >
                            </tr>
                        </thead>
                        <tbody>
                            {#each stats.monitors as m, idx}
                                <tr
                                    class="border-b border-border/50 hover:bg-surface-elevated/50 transition-colors cursor-pointer group"
                                    onclick={() => goto(`/monitors/${m.id}`)}
                                >
                                    <td class="px-5 py-3">
                                        <div class="flex items-center gap-2.5">
                                            <span
                                                class="text-[10px] font-mono text-text-subtle w-5 text-right"
                                                >#{idx + 1}</span
                                            >
                                            <div>
                                                <p
                                                    class="font-medium text-text group-hover:text-primary transition-colors"
                                                >
                                                    {m.name}
                                                </p>
                                                <p
                                                    class="text-[10px] text-text-subtle"
                                                >
                                                    {formatTypeLabel(m.type)}{#if m.group}
                                                        • {m.group}{/if}
                                                </p>
                                            </div>
                                        </div>
                                    </td>
                                    <td class="px-3 py-3">
                                        <span
                                            class="inline-flex items-center gap-1.5"
                                        >
                                            <span
                                                class="size-1.5 rounded-full {m.status ===
                                                'up'
                                                    ? 'bg-success'
                                                    : m.status === 'down'
                                                      ? 'bg-danger'
                                                      : 'bg-text-subtle'}"
                                            ></span>
                                            <span
                                                class="uppercase font-bold tracking-wider text-[10px] {m.status ===
                                                'up'
                                                    ? 'text-success'
                                                    : m.status === 'down'
                                                      ? 'text-danger'
                                                      : 'text-text-subtle'}"
                                                >{m.status}</span
                                            >
                                        </span>
                                    </td>
                                    <td class="px-3 py-3 text-right">
                                        <div
                                            class="flex items-center gap-2 justify-end"
                                        >
                                            <div
                                                class="w-16 h-1.5 rounded-full bg-surface-elevated overflow-hidden"
                                            >
                                                <div
                                                    class="h-full rounded-full"
                                                    style="width: {pct(
                                                        m.uptime_24h,
                                                    )}%; background: {ringColor(
                                                        m.uptime_24h,
                                                    )};"
                                                ></div>
                                            </div>
                                            <span
                                                class="font-mono font-bold tabular-nums {m.uptime_24h >=
                                                99
                                                    ? 'text-success'
                                                    : m.uptime_24h >= 95
                                                      ? 'text-warning'
                                                      : 'text-danger'}"
                                            >
                                                {m.uptime_24h.toFixed(4)}%
                                            </span>
                                        </div>
                                    </td>
                                    <td
                                        class="px-3 py-3 text-right font-mono tabular-nums text-text-muted"
                                    >
                                        {m.avg_latency != null
                                            ? m.avg_latency + "ms"
                                            : "—"}
                                    </td>
                                    <td
                                        class="px-3 py-3 text-right font-mono tabular-nums text-text-muted"
                                    >
                                        {m.p95_latency != null
                                            ? m.p95_latency + "ms"
                                            : "—"}
                                    </td>
                                    <td
                                        class="px-3 py-3 text-right font-mono tabular-nums text-text-subtle"
                                    >
                                        {m.min_latency != null
                                            ? m.min_latency + "ms"
                                            : "—"}
                                    </td>
                                    <td
                                        class="px-3 py-3 text-right font-mono tabular-nums text-text-subtle"
                                    >
                                        {m.max_latency != null
                                            ? m.max_latency + "ms"
                                            : "—"}
                                    </td>
                                    <td
                                        class="px-5 py-3 text-right font-mono tabular-nums text-text-muted"
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
    {/if}
</div>

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
