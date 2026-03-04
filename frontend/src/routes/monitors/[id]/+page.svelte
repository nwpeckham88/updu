<script lang="ts">
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import { fetchAPI } from "$lib/api/client";
    import {
        ArrowLeft,
        CheckCircle2,
        ServerCrash,
        Activity,
        TrendingUp,
        Wifi,
        ExternalLink,
        Clock,
    } from "lucide-svelte";
    import { format, formatDistanceToNow } from "date-fns";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Spinner from "$lib/components/ui/spinner.svelte";
    import Card from "$lib/components/ui/card.svelte";
    import { Tooltip } from "bits-ui";

    let monitorId = $derived($page.params.id);
    let monitor = $state<any>(null);
    let checks = $state<any[]>([]);
    let uptime = $state<{ "24h": number; "7d": number; "30d": number } | null>(
        null,
    );
    let loading = $state(true);
    let error = $state("");

    onMount(async () => {
        try {
            const [mon, recentChecks, uptimeData] = await Promise.all([
                fetchAPI(`/api/v1/monitors/${monitorId}`),
                fetchAPI(`/api/v1/monitors/${monitorId}/checks`),
                fetchAPI(`/api/v1/monitors/${monitorId}/uptime`),
            ]);
            monitor = mon;
            checks = recentChecks || [];
            uptime = uptimeData;
        } catch (e: any) {
            error = e.message || "Failed to load monitor";
        } finally {
            loading = false;
        }
    });

    function uptimePct(n: number | undefined) {
        if (n == null) return "—";
        return n.toFixed(2) + "%";
    }

    function uptimeColor(n: number | undefined) {
        if (n == null) return "text-text-subtle";
        if (n >= 99) return "text-success";
        if (n >= 95) return "text-warning";
        return "text-danger";
    }

    // Build 90-bucket history (newest right)
    const uptimeBuckets = $derived(() => {
        if (checks.length === 0) return Array(90).fill(null);
        const buckets: (any | null)[] = Array(90).fill(null);
        // checks are newest-first from API; place them right-aligned
        const slice = checks.slice(0, 90);
        for (let i = 0; i < slice.length; i++) {
            buckets[89 - i] = slice[i];
        }
        return buckets;
    });

    function statusColor(status: string) {
        if (status === "up") return "text-success";
        if (status === "down") return "text-danger";
        return "text-warning";
    }
</script>

<svelte:head>
    <title>{monitor?.name ?? "Monitor"} – updu</title>
</svelte:head>

<div class="space-y-5 max-w-5xl">
    <!-- Breadcrumb -->
    <a
        href="/monitors"
        class="inline-flex items-center gap-1.5 text-sm text-text-muted hover:text-text transition-colors"
    >
        <ArrowLeft class="size-4" />
        Monitors
    </a>

    {#if loading}
        <div class="space-y-5">
            <div class="flex items-center gap-3">
                <Skeleton height="h-8" width="w-48" />
                <Skeleton height="h-6" width="w-16" rounded="rounded-full" />
            </div>
            <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
                {#each { length: 4 } as _}
                    <div class="card p-4 space-y-2">
                        <Skeleton height="h-3" width="w-20" />
                        <Skeleton height="h-8" width="w-24" />
                    </div>
                {/each}
            </div>
            <div class="card p-5">
                <Skeleton height="h-3" width="w-32" class="mb-3" />
                <Skeleton height="h-8" width="w-full" rounded="rounded-md" />
            </div>
        </div>
    {:else if error}
        <div class="card border-danger/30 bg-danger/5 p-5 text-danger text-sm">
            {error}
        </div>
    {:else if monitor}
        <!-- Header -->
        <div
            class="flex flex-col sm:flex-row sm:items-start justify-between gap-4"
        >
            <div>
                <div class="flex items-center gap-3 flex-wrap">
                    <h1 class="text-2xl font-bold tracking-tight text-text">
                        {monitor.name}
                    </h1>
                    <Badge
                        status={!monitor.enabled ? "paused" : monitor.status}
                        size="md"
                    />
                </div>
                <div
                    class="flex items-center gap-2 mt-2 text-xs text-text-muted flex-wrap"
                >
                    <span
                        class="px-2 py-0.5 rounded-md bg-surface-elevated border border-border uppercase tracking-wider font-bold"
                    >
                        {monitor.type}
                    </span>
                    {#if monitor.group_name}
                        <span class="flex items-center gap-1">
                            <span class="size-1 rounded-full bg-border"></span>
                            {monitor.group_name}
                        </span>
                    {/if}
                    <span class="flex items-center gap-1">
                        <Clock class="size-3" />
                        Every {monitor.interval_s}s
                    </span>
                    {#if monitor.config?.url}
                        <a
                            href={monitor.config.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            class="flex items-center gap-1 text-primary hover:underline"
                        >
                            <ExternalLink class="size-3" />
                            {monitor.config.url.replace(/^https?:\/\//, "")}
                        </a>
                    {/if}
                </div>
            </div>
            {#if monitor.last_check}
                <p class="text-xs text-text-subtle shrink-0">
                    Last checked {formatDistanceToNow(
                        new Date(monitor.last_check),
                        { addSuffix: true },
                    )}
                </p>
            {/if}
        </div>

        <!-- Stat cards -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
            {#each [{ label: "Uptime 24h", value: uptime?.["24h"], icon: TrendingUp }, { label: "Uptime 7d", value: uptime?.["7d"], icon: TrendingUp }, { label: "Uptime 30d", value: uptime?.["30d"], icon: TrendingUp }, { label: "Last Latency", value: monitor.last_latency_ms != null ? monitor.last_latency_ms + "ms" : null, icon: Wifi }] as stat}
                <div class="card p-4">
                    <div
                        class="flex items-center gap-1.5 text-[10px] text-text-subtle uppercase tracking-wider font-medium mb-2"
                    >
                        <stat.icon class="size-3" />
                        {stat.label}
                    </div>
                    {#if stat.value != null}
                        <p
                            class="text-2xl font-bold {typeof stat.value ===
                            'number'
                                ? uptimeColor(stat.value)
                                : 'text-text font-mono'}"
                        >
                            {typeof stat.value === "number"
                                ? uptimePct(stat.value)
                                : stat.value}
                        </p>
                    {:else}
                        <p class="text-2xl font-bold text-text-subtle">—</p>
                    {/if}
                </div>
            {/each}
        </div>

        <!-- Uptime bar -->
        <div class="card p-5">
            <div class="flex items-center justify-between mb-3">
                <h2 class="text-sm font-semibold text-text">Check History</h2>
                <span class="text-xs text-text-subtle"
                    >{checks.length} checks</span
                >
            </div>
            <div class="flex gap-[2px] h-9 items-end">
                {#each uptimeBuckets() as check, i}
                    {#if check}
                        <Tooltip.Provider>
                            <Tooltip.Root delayDuration={100}>
                                <Tooltip.Trigger
                                    class="flex-1 rounded-sm h-full cursor-default transition-opacity hover:opacity-80 {check.status ===
                                    'up'
                                        ? 'bg-success/70'
                                        : check.status === 'down'
                                          ? 'bg-danger/80'
                                          : 'bg-warning/60'}"
                                ></Tooltip.Trigger>
                                <Tooltip.Portal>
                                    <Tooltip.Content
                                        class="z-50 rounded-lg border border-border bg-surface/95 backdrop-blur-sm px-3 py-2 text-xs shadow-[0_8px_32px_hsl(224_71%_4%/0.5)] text-text"
                                        sideOffset={4}
                                    >
                                        <p
                                            class="font-medium {statusColor(
                                                check.status,
                                            )}"
                                        >
                                            {check.status}
                                        </p>
                                        {#if check.latency_ms != null}
                                            <p
                                                class="text-text-muted font-mono"
                                            >
                                                {check.latency_ms}ms
                                            </p>
                                        {/if}
                                        <p class="text-text-subtle mt-1">
                                            {format(
                                                new Date(check.checked_at),
                                                "MMM d, HH:mm:ss",
                                            )}
                                        </p>
                                        <Tooltip.Arrow
                                            class="border-border fill-surface"
                                        />
                                    </Tooltip.Content>
                                </Tooltip.Portal>
                            </Tooltip.Root>
                        </Tooltip.Provider>
                    {:else}
                        <div
                            class="flex-1 rounded-sm h-full bg-border/30"
                        ></div>
                    {/if}
                {/each}
            </div>
            <div
                class="flex justify-between text-[10px] text-text-subtle mt-1.5"
            >
                <span>90 checks ago</span>
                <span>Now</span>
            </div>
        </div>

        <!-- Recent checks table -->
        <div class="card overflow-hidden" style="padding: 0;">
            <div class="px-4 py-3 border-b border-border bg-surface/30">
                <h2 class="text-sm font-semibold text-text">Recent Checks</h2>
            </div>
            {#if checks.length === 0}
                <div class="p-8 text-center text-sm text-text-subtle">
                    No checks recorded yet.
                </div>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full text-left text-sm">
                        <thead>
                            <tr
                                class="text-[11px] text-text-subtle uppercase tracking-wide border-b border-border bg-surface/20"
                            >
                                <th class="py-3 px-4 font-medium">Status</th>
                                <th class="py-3 px-4 font-medium">Latency</th>
                                <th class="py-3 px-4 font-medium">Message</th>
                                <th class="py-3 px-4 font-medium">Time</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border/60">
                            {#each checks.slice(0, 50) as check (check.id ?? check.checked_at)}
                                <tr
                                    class="hover:bg-surface/30 transition-colors {check.status ===
                                    'down'
                                        ? 'bg-danger/3'
                                        : ''}"
                                >
                                    <td class="py-2.5 px-4">
                                        <span
                                            class="inline-flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider {statusColor(
                                                check.status,
                                            )}"
                                        >
                                            {#if check.status === "up"}
                                                <CheckCircle2
                                                    class="size-3.5"
                                                />
                                            {:else if check.status === "down"}
                                                <ServerCrash class="size-3.5" />
                                            {:else}
                                                <Activity class="size-3.5" />
                                            {/if}
                                            {check.status}
                                        </span>
                                    </td>
                                    <td
                                        class="py-2.5 px-4 font-mono text-xs text-text-muted"
                                    >
                                        {check.latency_ms != null
                                            ? check.latency_ms + "ms"
                                            : "—"}
                                    </td>
                                    <td
                                        class="py-2.5 px-4 text-xs text-text-muted truncate max-w-xs"
                                    >
                                        {check.message || "—"}
                                    </td>
                                    <td
                                        class="py-2.5 px-4 text-xs text-text-subtle whitespace-nowrap"
                                    >
                                        {format(
                                            new Date(check.checked_at),
                                            "MMM d, HH:mm:ss",
                                        )}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>
