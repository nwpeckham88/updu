<script lang="ts">
    import { resolve } from "$app/paths";
    import {
        Plus,
        Search,
        Play,
        Pause,
        Pencil,
        Trash2,
        Activity,
        ChevronUp,
        ChevronDown,
        ChevronsUpDown,
        Loader2,
        Waves,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import LatencySparkline from "$lib/components/charts/latency-sparkline.svelte";
    import GroupPills from "$lib/components/monitors/group-pills.svelte";
    import { monitorsStore } from "$lib/stores/monitors.svelte";
    import type { Monitor } from "$lib/stores/monitors.svelte";
    import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
    import { confirmAction } from "$lib/stores/confirm.svelte";
    import { isFlapping, latencyTextClass } from "$lib/monitor-tones";
    import { onMount, onDestroy } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import CreateMonitorDialog from "$lib/components/CreateMonitorDialog.svelte";
    import EditMonitorDialog from "$lib/components/EditMonitorDialog.svelte";
    import { DropdownMenu } from "bits-ui";
    import { cn } from "$lib/utils";

    type SortKey = "name" | "status" | "latency";
    type StatusFilter = "all" | "up" | "down" | "paused";

    let searchQuery = $state("");
    let statusFilter = $state<StatusFilter>("all");
    let createDialogOpen = $state(false);
    let editDialogOpen = $state(false);
    let selectedMonitor = $state<any>(null);
    let sortKey = $state<SortKey | null>("status");
    let sortDir = $state<"asc" | "desc">("asc");
    // Per-row inflight state for pause/delete actions
    let inflight = $state<Record<string, boolean>>({});

    onMount(() => {
        monitorsStore.init();
    });
    onDestroy(() => {
        monitorsStore.destroy();
    });

    const counts = $derived.by(() => {
        const c = { all: 0, up: 0, down: 0, paused: 0 };
        for (const m of monitorsStore.monitors) {
            c.all += 1;
            if (!m.enabled) c.paused += 1;
            else if (m.status === "up") c.up += 1;
            else if (m.status === "down") c.down += 1;
        }
        return c;
    });

    const filtered = $derived.by(() => {
        const q = searchQuery.toLowerCase();
        let list = monitorsStore.monitors.filter((m) => {
            if (statusFilter === "paused" && m.enabled) return false;
            if (statusFilter === "up" && (!m.enabled || m.status !== "up"))
                return false;
            if (statusFilter === "down" && (!m.enabled || m.status !== "down"))
                return false;
            if (!q) return true;
            return (
                m.name.toLowerCase().includes(q) ||
                m.groups?.some((g: string) => g.toLowerCase().includes(q)) ||
                m.type.toLowerCase().includes(q)
            );
        });

        if (sortKey) {
            list = [...list].sort((a, b) => {
                let av: any, bv: any;
                if (sortKey === "name") {
                    av = a.name;
                    bv = b.name;
                } else if (sortKey === "status") {
                    av = monitorPriority(a);
                    bv = monitorPriority(b);
                } else if (sortKey === "latency") {
                    av = a.last_latency_ms ?? Infinity;
                    bv = b.last_latency_ms ?? Infinity;
                }
                if (av < bv) return sortDir === "asc" ? -1 : 1;
                if (av > bv) return sortDir === "asc" ? 1 : -1;
                return 0;
            });
        }
        return list;
    });

    function toggleSort(key: SortKey) {
        if (sortKey === key) {
            sortDir = sortDir === "asc" ? "desc" : "asc";
        } else {
            sortKey = key;
            sortDir = "asc";
        }
    }

    function monitorPriority(monitor: Monitor): number {
        if (!monitor.enabled || monitor.status === "paused") return 4;
        if (monitor.status === "down") return 0;
        if (monitorFlapping(monitor)) return 1;
        if (monitor.status === "degraded") return 2;
        if (monitor.status === "pending") return 3;
        return 5;
    }

    function monitorFlapping(monitor: Monitor): boolean {
        return monitor.enabled && isFlapping(monitor.recent_checks);
    }

    function rowToneClass(monitor: Monitor): string {
        if (!monitor.enabled || monitor.status === "paused") {
            return "border-l-2 border-text-subtle/30 bg-surface/20";
        }
        if (monitor.status === "down") {
            return "border-l-2 border-danger bg-danger/5 hover:bg-danger/10";
        }
        if (monitor.status === "degraded") {
            return "border-l-2 border-warning bg-warning/5 hover:bg-warning/10";
        }
        return "border-l-2 border-transparent";
    }

    function ariaSortFor(key: SortKey): "ascending" | "descending" | "none" {
        if (sortKey !== key) return "none";
        return sortDir === "asc" ? "ascending" : "descending";
    }

    function sortIconFor(key: SortKey) {
        if (sortKey !== key) return ChevronsUpDown;
        return sortDir === "asc" ? ChevronUp : ChevronDown;
    }

    async function togglePause(id: string, currentlyEnabled: boolean) {
        if (inflight[id]) return;
        inflight = { ...inflight, [id]: true };
        try {
            const mon = await fetchAPI(`/api/v1/monitors/${id}`);
            await fetchAPI(`/api/v1/monitors/${id}`, {
                method: "PUT",
                body: JSON.stringify({ ...mon, enabled: !currentlyEnabled }),
            });
            monitorsStore.init();
            toastStore.success(
                currentlyEnabled ? "Monitor paused" : "Monitor resumed",
            );
        } catch (e) {
            toastFromError(e, "Failed to update monitor");
            console.error("Failed to toggle monitor", e);
        } finally {
            inflight = { ...inflight, [id]: false };
        }
    }

    async function deleteMonitor(id: string) {
        const ok = await confirmAction({
            title: "Delete monitor?",
            description:
                "This will permanently remove the monitor and all historical check data. This action cannot be undone.",
            confirmLabel: "Delete monitor",
            variant: "destructive",
        });
        if (!ok) return;
        if (inflight[id]) return;
        inflight = { ...inflight, [id]: true };
        try {
            await fetchAPI(`/api/v1/monitors/${id}`, { method: "DELETE" });
            monitorsStore.init();
            toastStore.success("Monitor deleted");
        } catch (e) {
            toastFromError(e, "Failed to delete monitor");
            console.error("Failed to delete monitor", e);
        } finally {
            inflight = { ...inflight, [id]: false };
        }
    }

    async function openEditMonitor(id: string) {
        selectedMonitor = null;
        editDialogOpen = true;
        try {
            selectedMonitor = await fetchAPI(`/api/v1/monitors/${id}`);
        } catch (e) {
            editDialogOpen = false;
            toastFromError(e, "Failed to load monitor");
            console.error("Failed to load monitor", e);
        }
    }

    const SortIconStatus = $derived(sortIconFor("status"));
    const SortIconName = $derived(sortIconFor("name"));
    const SortIconLatency = $derived(sortIconFor("latency"));

    const filterChips: { value: StatusFilter; label: string }[] = [
        { value: "all", label: "All" },
        { value: "up", label: "Up" },
        { value: "down", label: "Down" },
        { value: "paused", label: "Paused" },
    ];
</script>

<svelte:head>
    <title>Monitors – updu</title>
</svelte:head>

<div class="space-y-5 max-w-7xl">
    <!-- Header -->
    <div
        class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
    >
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Monitors
            </h1>
            <p class="mt-1 type-caption text-text-muted">
                Manage infrastructure checks and endpoints
            </p>
        </div>
        <Button onclick={() => (createDialogOpen = true)}>
            <Plus class="size-4" />
            New Monitor
        </Button>
    </div>

    <!-- Table card -->
    <div class="card overflow-hidden" style="padding: 0;">
        <!-- Toolbar -->
        <div
            class="flex flex-col gap-3 border-b border-border bg-surface/30 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
            <div class="flex flex-1 flex-wrap items-center gap-2">
                <div class="relative w-full max-w-xs">
                    <Search
                        class="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-text-subtle"
                    />
                    <input
                        type="search"
                        placeholder="Search monitors..."
                        bind:value={searchQuery}
                        data-testid="search-monitors"
                        aria-label="Search monitors"
                        class="input-base h-9 pl-9 text-xs"
                    />
                </div>
                <div
                    class="flex items-center gap-1 rounded-lg border border-border bg-surface-elevated/40 p-0.5"
                    role="tablist"
                    aria-label="Filter by status"
                >
                    {#each filterChips as chip (chip.value)}
                        {@const active = statusFilter === chip.value}
                        {@const count = counts[chip.value]}
                        <button
                            type="button"
                            role="tab"
                            aria-selected={active}
                            data-testid={`filter-${chip.value}`}
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
                                    "type-numeric rounded px-1 font-semibold",
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
            </div>
            {#if !monitorsStore.loading}
                <span class="type-caption shrink-0 text-text-subtle">
                    {filtered.length} monitor{filtered.length === 1 ? "" : "s"}
                </span>
            {/if}
        </div>

        {#if monitorsStore.loading}
            <div class="divide-y divide-border" aria-busy="true" aria-label="Loading monitors">
                {#each { length: 5 } as _, i (i)}
                    <div class="flex items-center gap-4 px-4 py-3.5">
                        <Skeleton height="h-3" width="w-16" />
                        <Skeleton height="h-3" width="w-40" />
                        <Skeleton
                            height="h-5"
                            width="w-12"
                            rounded="rounded-full"
                        />
                        <Skeleton height="h-3" width="w-20" />
                        <Skeleton height="h-3" width="w-16" />
                        <div class="ml-auto">
                            <Skeleton
                                height="h-7"
                                width="w-20"
                                rounded="rounded-lg"
                            />
                        </div>
                    </div>
                {/each}
            </div>
        {:else if filtered.length === 0}
            <div data-testid="monitors-empty-state">
                <EmptyState
                    icon={Activity}
                    title={searchQuery
                        ? `No monitors matching "${searchQuery}"`
                        : statusFilter !== "all"
                          ? `No ${statusFilter} monitors`
                          : "No monitors yet"}
                    description={searchQuery || statusFilter !== "all"
                        ? "Try a different search or filter."
                        : "Click \u201CNew Monitor\u201D to create your first check."}
                />
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-left text-sm">
                    <thead class="sticky top-0 z-10">
                        <tr
                            class="type-kicker border-b border-border bg-surface/80 text-text-subtle backdrop-blur"
                        >
                            <th
                                scope="col"
                                class="px-4 py-3 font-medium"
                                aria-sort={ariaSortFor("status")}
                            >
                                <button
                                    data-testid="sort-status"
                                    class="flex items-center gap-1 transition-colors hover:text-text"
                                    onclick={() => toggleSort("status")}
                                >
                                    Status <SortIconStatus class="size-3" />
                                </button>
                            </th>
                            <th
                                scope="col"
                                class="px-4 py-3 font-medium"
                                aria-sort={ariaSortFor("name")}
                            >
                                <button
                                    data-testid="sort-name"
                                    class="flex items-center gap-1 transition-colors hover:text-text"
                                    onclick={() => toggleSort("name")}
                                >
                                    Name <SortIconName class="size-3" />
                                </button>
                            </th>
                            <th scope="col" class="px-4 py-3 font-medium">Type</th>
                            <th scope="col" class="px-4 py-3 font-medium">Groups</th>
                            <th
                                scope="col"
                                class="px-4 py-3 font-medium"
                                aria-sort={ariaSortFor("latency")}
                            >
                                <button
                                    data-testid="sort-latency"
                                    class="flex items-center gap-1 transition-colors hover:text-text"
                                    onclick={() => toggleSort("latency")}
                                >
                                    Latency <SortIconLatency class="size-3" />
                                </button>
                            </th>
                            <th
                                scope="col"
                                class="px-4 py-3 text-right font-medium"
                            >
                                Actions
                            </th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-border/60">
                        {#each filtered as monitor (monitor.id)}
                            {@const busy = !!inflight[monitor.id]}
                            {@const flapping = monitorFlapping(monitor)}
                            <tr
                                data-testid={`monitor-row-${monitor.id}`}
                                class={cn(
                                    "group transition-colors hover:bg-surface/40",
                                    rowToneClass(monitor),
                                    busy && "opacity-60",
                                )}
                            >
                                <td class="px-4 py-3">
                                    <div class="flex flex-col items-start gap-1.5">
                                        <Badge
                                            status={!monitor.enabled
                                                ? "paused"
                                                : monitor.status}
                                            calm={flapping}
                                        />
                                        {#if flapping}
                                            <span
                                                class="type-kicker inline-flex items-center gap-1 rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 text-warning"
                                                aria-label="Flapping: three or more status changes in ten minutes"
                                            >
                                                <Waves class="size-3" aria-hidden="true" />
                                                Flapping
                                            </span>
                                        {/if}
                                    </div>
                                </td>
                                <td class="px-4 py-3">
                                    <a
                                        href={resolve("/monitors/[id]", { id: monitor.id })}
                                        class="font-medium text-text transition-colors hover:text-primary"
                                    >
                                        {monitor.name}
                                    </a>
                                </td>
                                <td class="px-4 py-3">
                                    <span
                                        class="type-kicker inline-flex items-center rounded-md border border-border bg-surface-elevated px-2 py-0.5 text-text-muted"
                                    >
                                        {monitor.type}
                                    </span>
                                </td>
                                <td class="px-4 py-3">
                                    <GroupPills groups={monitor.groups} />
                                </td>
                                <td class="px-4 py-3">
                                    <div class="flex items-center gap-3">
                                        <LatencySparkline
                                            checks={monitor.recent_checks}
                                            fallbackLatency={monitor.last_latency_ms}
                                            label={`${monitor.name} latency trend`}
                                        />
                                        <span class="type-numeric text-xs">
                                            {#if !monitor.enabled}
                                                <span class="text-text-subtle">Paused</span>
                                            {:else if monitor.last_latency_ms != null}
                                                <span
                                                    class={latencyTextClass(
                                                        monitor.last_latency_ms,
                                                    )}
                                                >
                                                    {monitor.last_latency_ms}ms
                                                </span>
                                            {:else}
                                                <span class="text-text-subtle">No data</span>
                                            {/if}
                                        </span>
                                    </div>
                                </td>
                                <td class="px-4 py-3 text-right">
                                    <DropdownMenu.Root>
                                        <DropdownMenu.Trigger
                                            data-testid={`monitor-actions-${monitor.id}`}
                                            disabled={busy}
                                            class="inline-flex size-8 items-center justify-center rounded-lg text-text-subtle transition-colors hover:bg-surface-elevated hover:text-text disabled:cursor-not-allowed"
                                            aria-label={`Actions for ${monitor.name}`}
                                        >
                                            {#if busy}
                                                <Loader2
                                                    class="size-4 animate-spin"
                                                />
                                            {:else}
                                                <svg
                                                    class="size-4"
                                                    viewBox="0 0 24 24"
                                                    fill="currentColor"
                                                    aria-hidden="true"
                                                >
                                                    <circle cx="12" cy="5" r="1.5" />
                                                    <circle cx="12" cy="12" r="1.5" />
                                                    <circle cx="12" cy="19" r="1.5" />
                                                </svg>
                                            {/if}
                                        </DropdownMenu.Trigger>
                                        <DropdownMenu.Portal>
                                            <DropdownMenu.Content
                                                class="z-50 min-w-[10rem] rounded-xl border border-border bg-surface p-1 text-sm shadow-[0_8px_32px_hsl(224_71%_4%/0.5)] backdrop-blur-xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
                                                sideOffset={4}
                                                align="end"
                                            >
                                                <DropdownMenu.Item
                                                    class="flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-text-muted outline-none transition-colors hover:bg-surface-elevated hover:text-text"
                                                    onclick={() => {
                                                        void openEditMonitor(
                                                            monitor.id,
                                                        );
                                                    }}
                                                >
                                                    <Pencil class="size-3.5" /> Edit
                                                </DropdownMenu.Item>
                                                <DropdownMenu.Item
                                                    class="flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-text-muted outline-none transition-colors hover:bg-surface-elevated hover:text-text"
                                                    onclick={() =>
                                                        togglePause(
                                                            monitor.id,
                                                            monitor.enabled,
                                                        )}
                                                >
                                                    {#if monitor.enabled}
                                                        <Pause class="size-3.5" /> Pause
                                                    {:else}
                                                        <Play class="size-3.5" /> Resume
                                                    {/if}
                                                </DropdownMenu.Item>
                                                <DropdownMenu.Separator
                                                    class="my-1 h-px bg-border"
                                                />
                                                <DropdownMenu.Item
                                                    class="flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 text-danger outline-none transition-colors hover:bg-danger/10"
                                                    onclick={() =>
                                                        deleteMonitor(monitor.id)}
                                                >
                                                    <Trash2 class="size-3.5" /> Delete
                                                </DropdownMenu.Item>
                                            </DropdownMenu.Content>
                                        </DropdownMenu.Portal>
                                    </DropdownMenu.Root>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
</div>

<CreateMonitorDialog bind:open={createDialogOpen} />
<EditMonitorDialog bind:open={editDialogOpen} monitor={selectedMonitor} />
