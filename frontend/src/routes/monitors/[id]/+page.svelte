<script lang="ts">
    import { resolve } from "$app/paths";
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import { fetchAPI } from "$lib/api/client";
    import MonitorCheckDetails from "$lib/components/MonitorCheckDetails.svelte";
    import {
        ArrowLeft,
        CheckCircle2,
        ServerCrash,
        Activity,
        TrendingUp,
        Wifi,
        ExternalLink,
        Clock,
        History,
        ChevronDown,
        ChevronUp,
    } from "lucide-svelte";
    import { format, formatDistanceToNow } from "date-fns";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Stat from "$lib/components/ui/stat.svelte";
    import UptimeRibbon from "$lib/components/charts/uptime-ribbon.svelte";
    import { formatMonitorTypeLabel } from "$lib/monitor-config";

    let monitorId = $derived($page.params.id);
    let monitor = $state<any>(null);
    let checks = $state<any[]>([]);
    let events = $state<any[]>([]);
    let uptime = $state<{ "24h": number; "7d": number; "30d": number } | null>(
        null,
    );
    let loading = $state(true);
    let error = $state("");
    let showSamples = $state(false);
    const latestCheck = $derived(checks[0] ?? null);

    const monitorPrimaryGroup = $derived(
        monitor?.groups?.[0] ?? monitor?.group_name ?? monitor?.group ?? null,
    );
    const monitorPrimaryURL = $derived(
        typeof monitor?.config?.url === "string" ? monitor.config.url : null,
    );

    onMount(async () => {
        try {
            const [mon, recentChecks, uptimeData, recentEvents] =
                await Promise.all([
                    fetchAPI(`/api/v1/monitors/${monitorId}`),
                    fetchAPI(`/api/v1/monitors/${monitorId}/checks`),
                    fetchAPI(`/api/v1/monitors/${monitorId}/uptime`),
                    fetchAPI(`/api/v1/monitors/${monitorId}/events?limit=5`),
                ]);
            monitor = mon;
            checks = recentChecks || [];
            uptime = uptimeData;
            events = recentEvents || [];
        } catch (e: any) {
            error = e.message || "Failed to load monitor";
        } finally {
            loading = false;
        }
    });

    function uptimePct(n: number | undefined) {
        if (n == null) return "—";
        return n.toFixed(4) + "%";
    }

    // Build 90-bucket history (newest right)
    const uptimeBuckets = $derived.by(() => {
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
        href={resolve("/monitors")}
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
                {#each { length: 4 } as _, i (i)}
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
                        {formatMonitorTypeLabel(monitor.type)}
                    </span>
                    {#if monitorPrimaryGroup}
                        <span class="flex items-center gap-1">
                            <span class="size-1 rounded-full bg-border"></span>
                            {monitorPrimaryGroup}
                        </span>
                    {/if}
                    <span class="flex items-center gap-1">
                        <Clock class="size-3" />
                        Every {monitor.interval_s}s
                    </span>
                    {#if monitorPrimaryURL}
                        <a
                            href={monitorPrimaryURL}
                            target="_blank"
                            rel="noopener noreferrer"
                            class="flex items-center gap-1 text-primary hover:underline"
                        >
                            <ExternalLink class="size-3" />
                            {monitorPrimaryURL.replace(/^https?:\/\//, "")}
                        </a>
                    {/if}
                </div>
            </div>

            <div class="flex flex-col sm:items-end gap-2 shrink-0">
                <a
                    href={resolve("/monitors/[id]/events", { id: monitor.id })}
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-text-muted bg-surface/50 border border-border rounded-md hover:text-text hover:bg-surface transition-colors"
                >
                    <Activity class="size-3.5" />
                    View Events
                </a>

                {#if monitor.last_check}
                    <p class="text-[11px] text-text-subtle">
                        Last checked {formatDistanceToNow(
                            new Date(monitor.last_check),
                            { addSuffix: true },
                        )}
                    </p>
                {/if}
            </div>
        </div>

        <!-- Stat cards -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <Stat
                label="Uptime 24h"
                value={uptimePct(uptime?.["24h"])}
                icon={TrendingUp}
                tone={uptime?.["24h"] == null
                    ? "neutral"
                    : uptime["24h"] >= 99
                      ? "success"
                      : uptime["24h"] >= 95
                        ? "warning"
                        : "danger"}
            />
            <Stat
                label="Uptime 7d"
                value={uptimePct(uptime?.["7d"])}
                icon={TrendingUp}
                tone={uptime?.["7d"] == null
                    ? "neutral"
                    : uptime["7d"] >= 99
                      ? "success"
                      : uptime["7d"] >= 95
                        ? "warning"
                        : "danger"}
            />
            <Stat
                label="Uptime 30d"
                value={uptimePct(uptime?.["30d"])}
                icon={TrendingUp}
                tone={uptime?.["30d"] == null
                    ? "neutral"
                    : uptime["30d"] >= 99
                      ? "success"
                      : uptime["30d"] >= 95
                        ? "warning"
                        : "danger"}
            />
            <Stat
                label="Last Latency"
                value={monitor.last_latency_ms != null
                    ? monitor.last_latency_ms + "ms"
                    : "—"}
                icon={Wifi}
                tone="primary"
            />
        </div>

        <MonitorCheckDetails monitor={monitor} latestCheck={latestCheck} />

        <!-- Uptime bar -->
        <div class="card p-5">
            <div class="flex items-center justify-between mb-3">
                <h2 class="text-sm font-semibold text-text">Check History</h2>
                <span class="text-xs text-text-subtle"
                    >{checks.length} checks</span
                >
            </div>
            <UptimeRibbon
                buckets={uptimeBuckets}
                leftLabel="90 checks ago"
                rightLabel="Now"
            />
        </div>

        <!-- Events -->
        <div class="card overflow-hidden" style="padding: 0;">
            <div
                class="px-4 py-3 border-b border-border bg-surface/30 flex items-center justify-between"
            >
                <div class="flex items-center gap-2">
                    <History class="size-4 text-text-subtle" />
                    <h2 class="text-sm font-semibold text-text">
                        Recent Events
                    </h2>
                </div>
                <a
                    href={resolve("/monitors/[id]/events", { id: monitor.id })}
                    class="text-xs text-primary hover:underline"
                >
                    View all events
                </a>
            </div>
            {#if events.length === 0}
                <div class="p-8 text-center text-sm text-text-subtle">
                    No status changes recorded yet.
                </div>
            {:else}
                <div class="divide-y divide-border/60">
                    {#each events as event (event.id ?? event.created_at)}
                        <div
                            class="p-4 hover:bg-surface/30 transition-colors flex flex-col sm:flex-row gap-4 justify-between sm:items-center"
                        >
                            <div class="flex items-start gap-4">
                                <div class="mt-0.5">
                                    <Badge status={event.status} size="sm" />
                                </div>
                                <div>
                                    <p class="text-sm text-text font-medium">
                                        Status changed to <span
                                            class={statusColor(event.status)}
                                            >{event.status}</span
                                        >
                                    </p>
                                    {#if event.message}
                                        <p
                                            class="text-xs text-text-muted mt-0.5"
                                        >
                                            {event.message}
                                        </p>
                                    {/if}
                                </div>
                            </div>
                            <div
                                class="flex items-center justify-end gap-2 text-xs text-text-subtle shrink-0"
                            >
                                <Clock class="size-3" />
                                <span
                                    title={format(
                                        new Date(event.created_at),
                                        "PPpp",
                                    )}
                                >
                                    {formatDistanceToNow(
                                        new Date(event.created_at),
                                        { addSuffix: true },
                                    )}
                                </span>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>

        <!-- Raw Samples (Hidden by default) -->
        <div class="card overflow-hidden transition-all" style="padding: 0;">
            <button
                class="w-full px-4 py-3 bg-surface/30 flex items-center justify-between hover:bg-surface/50 transition-colors cursor-pointer {showSamples
                    ? 'border-b border-border'
                    : ''}"
                onclick={() => (showSamples = !showSamples)}
            >
                <div>
                    <h2
                        class="text-sm font-semibold text-text flex items-center gap-2 text-left"
                    >
                        Raw Monitor Samples
                        <Badge status="unknown" size="sm"
                            >{checks.length} recent</Badge
                        >
                    </h2>
                    <p class="text-xs text-text-muted mt-1 text-left">
                        Individual check results and latency measurements.
                    </p>
                </div>
                {#if showSamples}
                    <ChevronUp class="size-4 text-text-subtle" />
                {:else}
                    <ChevronDown class="size-4 text-text-subtle" />
                {/if}
            </button>

            {#if showSamples}
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
                                    <th class="py-3 px-4 font-medium">Status</th
                                    >
                                    <th class="py-3 px-4 font-medium"
                                        >Latency</th
                                    >
                                    <th class="py-3 px-4 font-medium"
                                        >Message</th
                                    >
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
                                                    <ServerCrash
                                                        class="size-3.5"
                                                    />
                                                {:else}
                                                    <Activity
                                                        class="size-3.5"
                                                    />
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
            {/if}
        </div>
    {/if}
</div>
