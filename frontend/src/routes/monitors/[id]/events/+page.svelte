<script lang="ts">
    import { onMount } from "svelte";
    import { resolve } from "$app/paths";
    import { page } from "$app/stores";
    import { fetchAPI } from "$lib/api/client";
    import { History, AlertCircle, ArrowUp, Activity } from "lucide-svelte";
    import { isToday, isYesterday, isThisWeek, format } from "date-fns";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Stat from "$lib/components/ui/stat.svelte";
    import Breadcrumbs from "$lib/components/ui/breadcrumbs.svelte";
    import EventRow from "$lib/components/monitors/event-row.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import { cn } from "$lib/utils";

    type StatusFilter = "all" | "down" | "up" | "other";

    let monitorId = $derived($page.params.id);
    let monitor = $state<any>(null);
    let events = $state<any[]>([]);
    let loading = $state(true);
    let error = $state("");
    let statusFilter = $state<StatusFilter>("all");
    let limit = $state(50);
    let loadingMore = $state(false);

    async function loadEvents(newLimit: number) {
        const evts = await fetchAPI(
            `/api/v1/monitors/${monitorId}/events?limit=${newLimit}`,
        );
        events = evts || [];
    }

    onMount(async () => {
        try {
            const [mon, evts] = await Promise.all([
                fetchAPI(`/api/v1/monitors/${monitorId}`),
                fetchAPI(`/api/v1/monitors/${monitorId}/events?limit=${limit}`),
            ]);
            monitor = mon;
            events = evts || [];
        } catch (e: any) {
            error = e.message || "Failed to load events";
        } finally {
            loading = false;
        }
    });

    async function loadMore() {
        if (loadingMore) return;
        loadingMore = true;
        try {
            const newLimit = limit + 50;
            await loadEvents(newLimit);
            limit = newLimit;
        } catch (e: any) {
            error = e.message || "Failed to load more events";
        } finally {
            loadingMore = false;
        }
    }

    const counts = $derived.by(() => {
        const c = { all: 0, down: 0, up: 0, other: 0 };
        for (const e of events) {
            c.all += 1;
            if (e.status === "down") c.down += 1;
            else if (e.status === "up") c.up += 1;
            else c.other += 1;
        }
        return c;
    });

    const filtered = $derived.by(() => {
        if (statusFilter === "all") return events;
        if (statusFilter === "other")
            return events.filter(
                (e) => e.status !== "up" && e.status !== "down",
            );
        return events.filter((e) => e.status === statusFilter);
    });

    type Group = { label: string; events: any[] };

    const grouped = $derived.by<Group[]>(() => {
        const buckets: Record<string, Group> = {};
        const order: string[] = [];

        const ensure = (key: string, label: string) => {
            if (!buckets[key]) {
                buckets[key] = { label, events: [] };
                order.push(key);
            }
            return buckets[key];
        };

        for (const e of filtered) {
            const date = new Date(e.created_at);
            let key: string;
            let label: string;
            if (isToday(date)) {
                key = "today";
                label = "Today";
            } else if (isYesterday(date)) {
                key = "yesterday";
                label = "Yesterday";
            } else if (isThisWeek(date, { weekStartsOn: 1 })) {
                key = "this-week";
                label = "Earlier this week";
            } else {
                key = format(date, "yyyy-MM");
                label = format(date, "MMMM yyyy");
            }
            ensure(key, label).events.push(e);
        }

        return order.map((k) => buckets[k]);
    });

    const filterChips: { value: StatusFilter; label: string }[] = [
        { value: "all", label: "All" },
        { value: "down", label: "Down" },
        { value: "up", label: "Up" },
        { value: "other", label: "Other" },
    ];
</script>

<svelte:head>
    <title>{monitor?.name ? monitor.name + " Events" : "Events"} – updu</title>
</svelte:head>

<div class="max-w-5xl space-y-5">
    <Breadcrumbs
        items={[
            { label: "Monitors", href: resolve("/monitors") },
            {
                label: monitor?.name ?? "Monitor",
                href: monitor
                    ? resolve("/monitors/[id]", { id: monitor.id })
                    : undefined,
            },
            { label: "Events" },
        ]}
    />

    {#if loading}
        <div class="space-y-5">
            <div class="flex items-center gap-3">
                <Skeleton height="h-8" width="w-48" />
                <Skeleton height="h-6" width="w-16" rounded="rounded-full" />
            </div>
            <div class="grid grid-cols-3 gap-3">
                {#each { length: 3 } as _, i (i)}
                    <Skeleton height="h-16" width="w-full" />
                {/each}
            </div>
            <div class="card p-5">
                <Skeleton height="h-8" width="w-full" rounded="rounded-md" />
            </div>
        </div>
    {:else if error}
        <div class="card border-danger/30 bg-danger/5 p-5 text-sm text-danger">
            {error}
        </div>
    {:else if monitor}
        <!-- Header -->
        <div
            class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"
        >
            <div class="min-w-0">
                <h1 class="text-2xl font-bold tracking-tight text-text">
                    {monitor.name}
                </h1>
                <p class="mt-1 flex items-center gap-2 text-sm text-text-muted">
                    Event history
                    <Badge
                        status={!monitor.enabled ? "paused" : monitor.status}
                        size="sm"
                    />
                </p>
            </div>
        </div>

        <!-- Summary tiles -->
        <div class="grid grid-cols-3 gap-3">
            <Stat
                label="Down events"
                value={String(counts.down)}
                icon={AlertCircle}
                tone={counts.down > 0 ? "danger" : "neutral"}
            />
            <Stat
                label="Recovery events"
                value={String(counts.up)}
                icon={ArrowUp}
                tone={counts.up > 0 ? "success" : "neutral"}
            />
            <Stat
                label="Other events"
                value={String(counts.other)}
                icon={Activity}
                tone="neutral"
            />
        </div>

        <!-- Filter chips -->
        <div
            class="flex items-center gap-1 rounded-lg border border-border bg-surface-elevated/40 p-0.5 w-fit"
            role="tablist"
            aria-label="Filter events by status"
        >
            {#each filterChips as chip (chip.value)}
                {@const active = statusFilter === chip.value}
                {@const count = counts[chip.value]}
                <button
                    type="button"
                    role="tab"
                    aria-selected={active}
                    onclick={() => (statusFilter = chip.value)}
                    class={cn(
                        "inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
                        active
                            ? "bg-primary text-primary-foreground"
                            : "text-text-muted hover:text-text",
                    )}
                >
                    <span>{chip.label}</span>
                    <span
                        class={cn(
                            "rounded px-1 text-[10px] font-semibold",
                            active
                                ? "bg-primary-foreground/20"
                                : "bg-surface text-text-subtle",
                        )}
                    >
                        {count}
                    </span>
                </button>
            {/each}
        </div>

        <!-- Events list -->
        <div class="card overflow-hidden" style="padding: 0;">
            <div
                class="flex items-center gap-2 border-b border-border bg-surface/30 px-4 py-3"
            >
                <History class="size-4 text-text-subtle" />
                <h2 class="text-sm font-semibold text-text">All Events</h2>
            </div>
            {#if filtered.length === 0}
                <EmptyState
                    icon={History}
                    title={statusFilter === "all"
                        ? "No events recorded for this monitor."
                        : `No ${statusFilter} events.`}
                    description="Status changes appear here as they happen."
                />
            {:else}
                {#each grouped as group (group.label)}
                    <div
                        class="sticky top-0 z-10 border-y border-border bg-surface/80 px-4 py-2 text-[11px] font-semibold uppercase tracking-wider text-text-subtle backdrop-blur"
                    >
                        {group.label}
                        <span class="text-text-subtle/60">
                            · {group.events.length}
                        </span>
                    </div>
                    <div class="divide-y divide-border/60">
                        {#each group.events as event (event.id ?? event.created_at)}
                            <EventRow {event} />
                        {/each}
                    </div>
                {/each}
                {#if events.length >= limit}
                    <div class="border-t border-border bg-surface/20 p-3 text-center">
                        <button
                            type="button"
                            onclick={loadMore}
                            disabled={loadingMore}
                            class="text-xs font-medium text-primary hover:underline disabled:opacity-60"
                        >
                            {loadingMore ? "Loading..." : "Load more"}
                        </button>
                    </div>
                {/if}
            {/if}
        </div>
    {/if}
</div>
