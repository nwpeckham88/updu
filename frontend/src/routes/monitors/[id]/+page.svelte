<script lang="ts">
    import { resolve } from "$app/paths";
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import { fetchAPI } from "$lib/api/client";
    import MonitorCheckDetails from "$lib/components/MonitorCheckDetails.svelte";
    import EventRow from "$lib/components/monitors/event-row.svelte";
    import {
        CheckCircle2,
        ServerCrash,
        Activity,
        TrendingUp,
        ExternalLink,
        Clock,
        History,
        Loader2,
        UserCheck,
        Waves,
    } from "lucide-svelte";
    import { format, formatDistanceToNow } from "date-fns";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Stat from "$lib/components/ui/stat.svelte";
    import BulletBar from "$lib/components/charts/bullet-bar.svelte";
    import StatusDonut from "$lib/components/charts/status-donut.svelte";
    import UptimeRibbon from "$lib/components/charts/uptime-ribbon.svelte";
    import Breadcrumbs from "$lib/components/ui/breadcrumbs.svelte";
    import Tooltip from "$lib/components/ui/tooltip.svelte";
    import { authStore } from "$lib/stores/auth.svelte";
    import { toastFromError } from "$lib/stores/toast.svelte";
    import { formatMonitorTypeLabel } from "$lib/monitor-config";
    import {
        uptimeTone,
        statusTextClass,
        latencyTextClass,
        isFlapping,
    } from "$lib/monitor-tones";

    let monitorId = $derived($page.params.id);
    let monitor = $state<any>(null);
    let checks = $state<any[]>([]);
    let events = $state<any[]>([]);
    let uptime = $state<{ "24h": number; "7d": number; "30d": number } | null>(
        null,
    );
    let loading = $state(true);
    let error = $state("");
    let investigationPending = $state(false);
    const latestCheck = $derived(checks[0] ?? null);
    const monitorFlapping = $derived(isFlapping(checks));
    const investigationActive = $derived(Boolean(monitor?.investigation?.active));
    const canInvestigate = $derived(authStore.user?.role === "admin");
    const investigationAge = $derived(
        monitor?.investigation?.updated_at
            ? formatDistanceToNow(new Date(monitor.investigation.updated_at), {
                  addSuffix: true,
              })
            : "",
    );
    const apdexValues = $derived.by(() => {
        const values: number[] = [];
        for (const check of checks) {
            if (check.latency_ms != null) values.push(check.latency_ms);
        }
        return values;
    });

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

    onMount(() => {
        const eventSource = new EventSource("/api/v1/events");
        eventSource.addEventListener("monitor:investigation", (event: MessageEvent) => {
            try {
                const payload = JSON.parse(event.data);
                if (payload.monitor_id !== monitorId) return;
                applyInvestigation(payload);
            } catch {
                // Ignore malformed realtime events; the next page load rehydrates state.
            }
        });

        return () => eventSource.close();
    });

    function applyInvestigation(payload: any) {
        if (!monitor) return;
        if (payload.active) {
            monitor = { ...monitor, investigation: payload };
            return;
        }

        const { investigation: removedInvestigation, ...nextMonitor } = monitor;
        void removedInvestigation;
        monitor = nextMonitor;
    }

    async function toggleInvestigation() {
        if (!monitor || investigationPending) return;

        investigationPending = true;
        try {
            const payload = await fetchAPI(`/api/v1/monitors/${monitor.id}/investigate`, {
                method: "POST",
                body: JSON.stringify({ investigating: !investigationActive }),
            });
            applyInvestigation(payload);
        } catch (err) {
            toastFromError(err, "Failed to update investigation marker");
        } finally {
            investigationPending = false;
        }
    }

    function uptimePct(n: number | undefined | null) {
        if (n == null) return "—";
        return n.toFixed(4) + "%";
    }

    // Build 90-bucket history (newest right)
    const uptimeBuckets = $derived.by(() => {
        if (checks.length === 0) return Array(90).fill(null);
        const buckets: (any | null)[] = Array(90).fill(null);
        const slice = checks.slice(0, 90);
        for (let i = 0; i < slice.length; i++) {
            buckets[89 - i] = slice[i];
        }
        return buckets;
    });

    const sectionLinks = $derived([
        { id: "health", label: "Health" },
        { id: "config", label: "Config" },
        { id: "history", label: "History" },
        { id: "events", label: "Events" },
        { id: "samples", label: "Samples" },
    ]);
</script>

<svelte:head>
    <title>{monitor?.name ?? "Monitor"} – updu</title>
</svelte:head>

<div class="space-y-5 max-w-5xl">
    <Breadcrumbs
        items={[
            { label: "Monitors", href: resolve("/monitors") },
            { label: monitor?.name ?? "Monitor" },
        ]}
    />

    {#if loading}
        <div class="space-y-5" aria-busy="true" aria-label="Loading monitor details">
            <div class="flex items-center gap-3">
                <Skeleton height="h-8" width="w-48" />
                <Skeleton height="h-6" width="w-16" rounded="rounded-full" />
            </div>
            <div class="card grid grid-cols-1 gap-4 p-5 sm:grid-cols-[auto,1fr]">
                <Skeleton height="h-36" width="w-36" rounded="rounded-full" />
                <div class="grid grid-cols-3 gap-3">
                    {#each { length: 3 } as _, i (i)}
                        <Skeleton height="h-16" width="w-full" />
                    {/each}
                </div>
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
            class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"
        >
            <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-3">
                    <h1 class="text-2xl font-bold tracking-tight text-text">
                        {monitor.name}
                    </h1>
                    <Badge
                        status={!monitor.enabled ? "paused" : monitor.status}
                        size="md"
                        calm={monitorFlapping}
                    />
                    {#if monitor.enabled && monitorFlapping}
                        <span
                            class="type-kicker inline-flex items-center gap-1.5 rounded-full border border-warning/25 bg-warning/10 px-2.5 py-1 text-warning"
                            aria-label="Flapping: three or more status changes in ten minutes"
                        >
                            <Waves class="size-3.5" aria-hidden="true" />
                            Flapping
                        </span>
                    {/if}
                </div>
                <div
                    class="type-caption mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-text-muted"
                >
                    <span
                        class="type-kicker rounded-md border border-border bg-surface-elevated px-2 py-0.5"
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
                            class="flex max-w-[18rem] items-center gap-1 truncate text-primary hover:underline"
                            title={monitorPrimaryURL}
                        >
                            <ExternalLink class="size-3 shrink-0" />
                            <span class="truncate">{monitorPrimaryURL.replace(/^https?:\/\//, "")}</span>
                        </a>
                    {/if}
                </div>
            </div>

            <div class="flex shrink-0 flex-col gap-2 sm:items-end">
                {#if canInvestigate}
                    <button
                        type="button"
                        class="type-data inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 {investigationActive
                            ? 'border-warning/40 bg-warning/10 text-warning hover:bg-warning/15'
                            : 'border-border bg-surface/50 text-text-muted hover:bg-surface hover:text-text'}"
                        aria-pressed={investigationActive}
                        disabled={investigationPending}
                        onclick={toggleInvestigation}
                    >
                        {#if investigationPending}
                            <Loader2 class="size-3.5 motion-safe:animate-spin" />
                        {:else}
                            <UserCheck class="size-3.5" />
                        {/if}
                        {investigationActive ? "Investigating" : "I'm on it"}
                    </button>
                {/if}
                {#if investigationActive}
                    <p class="type-micro font-medium text-warning">
                        Marked {investigationAge}
                    </p>
                {/if}
                <a
                    href={resolve("/monitors/[id]/events", { id: monitor.id })}
                    class="type-data inline-flex items-center gap-1.5 rounded-md border border-border bg-surface/50 px-3 py-1.5 text-xs font-medium text-text-muted transition-colors hover:bg-surface hover:text-text"
                >
                    <Activity class="size-3.5" />
                    View Events
                </a>

                {#if monitor.last_check}
                    <p class="type-micro text-text-subtle">
                        Last checked
                        <time
                            datetime={monitor.last_check}
                            title={format(new Date(monitor.last_check), "PPpp")}
                        >
                            {formatDistanceToNow(new Date(monitor.last_check), {
                                addSuffix: true,
                            })}
                        </time>
                    </p>
                {/if}
            </div>
        </div>

        <!-- In-page nav rail -->
        <nav
            aria-label="Section navigation"
                    class="type-caption -mt-1 flex flex-wrap gap-1 text-text-muted"
        >
            {#each sectionLinks as link (link.id)}
                <a
                    href={`#${link.id}`}
                    class="rounded-md border border-transparent px-2 py-1 hover:border-border hover:bg-surface-elevated/40 hover:text-text"
                >
                    {link.label}
                </a>
            {/each}
        </nav>

        <!-- Health hero: donut + tile stack -->
        <section
            id="health"
            aria-labelledby="health-heading"
            class="card p-5"
        >
            <h2 id="health-heading" class="sr-only">Health Overview</h2>
            <div class="grid grid-cols-1 gap-5 sm:grid-cols-[auto,1fr] sm:items-center">
                <div class="flex justify-center sm:justify-start">
                    <StatusDonut
                        value={uptime?.["24h"] ?? 0}
                        size="md"
                        label={uptimePct(uptime?.["24h"])}
                        sublabel="Uptime · 24h"
                        apdexValues={apdexValues}
                    />
                </div>
                <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <Stat
                        label="Uptime 7d"
                        value={uptimePct(uptime?.["7d"])}
                        icon={TrendingUp}
                        tone={uptimeTone(uptime?.["7d"])}
                    />
                    <Stat
                        label="Uptime 30d"
                        value={uptimePct(uptime?.["30d"])}
                        icon={TrendingUp}
                        tone={uptimeTone(uptime?.["30d"])}
                    />
                    <div class="card p-4">
                        <BulletBar
                            label="Last latency"
                            value={monitor.last_latency_ms ?? null}
                            target={500}
                            warning={1000}
                            danger={3000}
                        />
                    </div>
                </div>
            </div>
        </section>

        <section id="config" aria-labelledby="config-heading">
            <h2 id="config-heading" class="sr-only">Configuration</h2>
            <MonitorCheckDetails monitor={monitor} latestCheck={latestCheck} />
        </section>

        <!-- Uptime ribbon -->
        <section
            id="history"
            aria-labelledby="history-heading"
            class="card p-5"
        >
            <div class="mb-3 flex items-center justify-between">
                <h2
                    id="history-heading"
                    class="type-section-title text-text"
                >
                    Check History
                </h2>
                <span class="type-numeric text-xs text-text-subtle">
                    {checks.length} checks
                </span>
            </div>
            <UptimeRibbon
                buckets={uptimeBuckets}
                leftLabel="90 checks ago"
                rightLabel="Now"
            />
        </section>

        <!-- Events -->
        <section id="events" aria-labelledby="events-heading">
            <div class="card overflow-hidden" style="padding: 0;">
                <div
                    class="flex items-center justify-between border-b border-border bg-surface/30 px-4 py-3"
                >
                    <div class="flex items-center gap-2">
                        <History class="size-4 text-text-subtle" />
                        <h2
                            id="events-heading"
                            class="type-section-title text-text"
                        >
                            Recent Events
                        </h2>
                    </div>
                    <a
                        href={resolve("/monitors/[id]/events", {
                            id: monitor.id,
                        })}
                        class="type-caption text-primary hover:underline"
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
                            <EventRow {event} />
                        {/each}
                    </div>
                {/if}
            </div>
        </section>

        <!-- Raw Samples (native disclosure) -->
        <section id="samples" aria-labelledby="samples-heading">
            <details
                class="card overflow-hidden transition-all"
                style="padding: 0;"
            >
                <summary
                    class="flex w-full cursor-pointer list-none items-center justify-between bg-surface/30 px-4 py-3 transition-colors hover:bg-surface/50"
                >
                    <div>
                        <h2
                            id="samples-heading"
                            class="type-section-title flex items-center gap-2 text-left text-text"
                        >
                            Raw Monitor Samples
                            <Badge status="unknown" size="sm">
                                {checks.length} recent
                            </Badge>
                        </h2>
                        <p class="mt-1 type-caption text-left text-text-muted">
                            Individual check results and latency measurements.
                        </p>
                    </div>
                    <span
                        class="text-text-subtle transition-transform group-open:rotate-180"
                        aria-hidden="true"
                    >
                        ▾
                    </span>
                </summary>

                {#if checks.length === 0}
                    <div class="p-8 text-center text-sm text-text-subtle">
                        No checks recorded yet.
                    </div>
                {:else}
                    <div class="overflow-x-auto border-t border-border">
                        <table class="w-full text-left text-sm">
                            <thead>
                                <tr
                                    class="type-kicker border-b border-border bg-surface/20 text-text-subtle"
                                >
                                    <th scope="col" class="px-4 py-3 font-medium">
                                        Status
                                    </th>
                                    <th scope="col" class="px-4 py-3 font-medium">
                                        Latency
                                    </th>
                                    <th scope="col" class="px-4 py-3 font-medium">
                                        Message
                                    </th>
                                    <th scope="col" class="px-4 py-3 font-medium">
                                        Time
                                    </th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-border/60">
                                {#each checks.slice(0, 50) as check (check.id ?? check.checked_at)}
                                    <tr
                                        class="transition-colors hover:bg-surface/30 {check.status ===
                                        'down'
                                            ? 'bg-danger/3'
                                            : ''}"
                                    >
                                        <td class="px-4 py-2.5">
                                            <span
                                                class="type-kicker inline-flex items-center gap-1.5 {statusTextClass(
                                                    check.status,
                                                )}"
                                            >
                                                {#if check.status === "up"}
                                                    <CheckCircle2 class="size-3.5" />
                                                {:else if check.status === "down"}
                                                    <ServerCrash class="size-3.5" />
                                                {:else}
                                                    <Activity class="size-3.5" />
                                                {/if}
                                                {check.status}
                                            </span>
                                        </td>
                                        <td
                                            class="type-numeric px-4 py-2.5 text-xs {latencyTextClass(
                                                check.latency_ms,
                                            )}"
                                        >
                                            {check.latency_ms != null
                                                ? check.latency_ms + "ms"
                                                : "—"}
                                        </td>
                                        <td
                                            class="max-w-xs truncate px-4 py-2.5 text-xs text-text-muted"
                                        >
                                            {#if check.message}
                                                <Tooltip content={check.message}>
                                                    <span class="truncate">
                                                        {check.message}
                                                    </span>
                                                </Tooltip>
                                            {:else}
                                                —
                                            {/if}
                                        </td>
                                        <td
                                            class="type-numeric whitespace-nowrap px-4 py-2.5 text-xs text-text-subtle"
                                        >
                                            <time
                                                datetime={check.checked_at}
                                                title={format(
                                                    new Date(check.checked_at),
                                                    "PPpp",
                                                )}
                                            >
                                                {format(
                                                    new Date(check.checked_at),
                                                    "MMM d, HH:mm:ss",
                                                )}
                                            </time>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </details>
        </section>
    {/if}
</div>
