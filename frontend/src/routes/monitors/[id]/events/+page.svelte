<script lang="ts">
    import { onMount } from "svelte";
    import { page } from "$app/stores";
    import { fetchAPI } from "$lib/api/client";
    import { ArrowLeft, Clock, History } from "lucide-svelte";
    import { formatDistanceToNow, format } from "date-fns";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";

    let monitorId = $derived($page.params.id);
    let monitor = $state<any>(null);
    let events = $state<any[]>([]);
    let loading = $state(true);
    let error = $state("");

    onMount(async () => {
        try {
            const [mon, evts] = await Promise.all([
                fetchAPI(`/api/v1/monitors/${monitorId}`),
                fetchAPI(`/api/v1/monitors/${monitorId}/events?limit=50`),
            ]);
            monitor = mon;
            events = evts || [];
        } catch (e: any) {
            error = e.message || "Failed to load events";
        } finally {
            loading = false;
        }
    });

    function statusColor(status: string) {
        if (status === "up") return "text-success";
        if (status === "down") return "text-danger";
        return "text-warning";
    }
</script>

<svelte:head>
    <title>{monitor?.name ? monitor.name + " Events" : "Events"} – updu</title>
</svelte:head>

<div class="space-y-5 max-w-5xl">
    <!-- Breadcrumb -->
    <a
        href={`/monitors/${monitorId}`}
        class="inline-flex items-center gap-1.5 text-sm text-text-muted hover:text-text transition-colors"
    >
        <ArrowLeft class="size-4" />
        Back to Monitor
    </a>

    {#if loading}
        <div class="space-y-5">
            <div class="flex items-center gap-3">
                <Skeleton height="h-8" width="w-48" />
                <Skeleton height="h-6" width="w-16" rounded="rounded-full" />
            </div>
            <div class="card p-5">
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
                        <span class="text-text-muted font-normal">Events</span>
                    </h1>
                    <Badge
                        status={!monitor.enabled ? "paused" : monitor.status}
                        size="md"
                    />
                </div>
            </div>
        </div>

        <!-- Events List -->
        <div class="card overflow-hidden" style="padding: 0;">
            <div
                class="px-4 py-3 border-b border-border bg-surface/30 flex items-center gap-2"
            >
                <History class="size-4 text-text-subtle" />
                <h2 class="text-sm font-semibold text-text">Recent Events</h2>
            </div>
            {#if events.length === 0}
                <div class="p-8 text-center text-sm text-text-subtle">
                    No events recorded for this monitor.
                </div>
            {:else}
                <div class="divide-y divide-border/60">
                    {#each events as event}
                        <div
                            class="p-4 hover:bg-surface/30 transition-colors flex flex-col sm:flex-row gap-4 justify-between sm:items-center"
                        >
                            <div class="flex items-start gap-4">
                                <div class="mt-0.5">
                                    <Badge status={event.status} size="sm" />
                                </div>
                                <div>
                                    <p class="text-sm text-text font-medium">
                                        Status changed to {event.status}
                                    </p>
                                    {#if event.message}
                                        <p
                                            class="text-xs text-text-muted mt-0.5 line-clamp-2"
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
    {/if}
</div>
