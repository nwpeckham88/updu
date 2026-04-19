<script lang="ts">
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
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Badge from "$lib/components/ui/badge.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import { monitorsStore } from "$lib/stores/monitors.svelte";
    import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
    import { confirmAction } from "$lib/stores/confirm.svelte";
    import { onMount, onDestroy } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import CreateMonitorDialog from "$lib/components/CreateMonitorDialog.svelte";
    import EditMonitorDialog from "$lib/components/EditMonitorDialog.svelte";
    import { DropdownMenu } from "bits-ui";

    let searchQuery = $state("");
    let createDialogOpen = $state(false);
    let editDialogOpen = $state(false);
    let selectedMonitor = $state<any>(null);
    let sortKey = $state<"name" | "status" | "latency" | null>(null);
    let sortDir = $state<"asc" | "desc">("asc");

    onMount(() => {
        monitorsStore.init();
    });
    onDestroy(() => {
        monitorsStore.destroy();
    });

    const filtered = $derived.by(() => {
        let list = monitorsStore.monitors.filter(
            (m) =>
                m.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                m.groups?.some((g) =>
                    g.toLowerCase().includes(searchQuery.toLowerCase()),
                ) ||
                m.type.toLowerCase().includes(searchQuery.toLowerCase()),
        );

        if (sortKey) {
            list = [...list].sort((a, b) => {
                let av: any, bv: any;
                if (sortKey === "name") {
                    av = a.name;
                    bv = b.name;
                } else if (sortKey === "status") {
                    av = a.status;
                    bv = b.status;
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

    function toggleSort(key: typeof sortKey) {
        if (sortKey === key) {
            sortDir = sortDir === "asc" ? "desc" : "asc";
        } else {
            sortKey = key;
            sortDir = "asc";
        }
    }

    async function togglePause(id: string, currentlyEnabled: boolean) {
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
            console.error(`Failed to toggle monitor`, e);
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
        try {
            await fetchAPI(`/api/v1/monitors/${id}`, { method: "DELETE" });
            monitorsStore.init();
            toastStore.success("Monitor deleted");
        } catch (e) {
            toastFromError(e, "Failed to delete monitor");
            console.error("Failed to delete monitor", e);
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

    function sortIconFor(
        key: typeof sortKey,
    ): typeof ChevronUp | typeof ChevronDown | typeof ChevronsUpDown {
        if (sortKey !== key) return ChevronsUpDown;
        return sortDir === "asc" ? ChevronUp : ChevronDown;
    }

    // Reactive sort icons — used directly in template (can't use @const inside <button>)
    const sortIconStatus = $derived(sortIconFor("status"));
    const sortIconName = $derived(sortIconFor("name"));
    const sortIconLatency = $derived(sortIconFor("latency"));
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
            <p class="text-sm text-text-muted mt-1">
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
            class="px-4 py-3 border-b border-border flex items-center justify-between gap-3 bg-surface/30"
        >
            <div class="relative max-w-xs w-full">
                <Search
                    class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-text-subtle pointer-events-none"
                />
                <input
                    type="text"
                    placeholder="Search monitors..."
                    bind:value={searchQuery}
                    data-testid="search-monitors"
                    class="input-base pl-9 h-9 text-xs"
                />
            </div>
            {#if !monitorsStore.loading}
                <span class="text-xs text-text-subtle shrink-0">
                    {filtered.length} monitor{filtered.length === 1 ? "" : "s"}
                </span>
            {/if}
        </div>

        {#if monitorsStore.loading}
            <!-- Skeleton rows -->
            <div class="divide-y divide-border">
                {#each { length: 5 } as _}
                    <div class="px-4 py-3.5 flex items-center gap-4">
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
                        : "No monitors yet"}
                    description={searchQuery
                        ? "Try a different search term."
                        : "Click \u201CNew Monitor\u201D to create your first check."}
                />
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-left text-sm">
                    <thead>
                        <tr
                            class="text-[11px] font-semibold tracking-wider text-text-subtle uppercase border-b border-border bg-surface/20"
                        >
                            <th class="py-3 px-4 font-medium">
                                <button
                                    data-testid="sort-status"
                                    class="flex items-center gap-1 hover:text-text transition-colors"
                                    onclick={() => toggleSort("status")}
                                >
                                    Status <sortIconStatus class="size-3"
                                    ></sortIconStatus>
                                </button>
                            </th>
                            <th class="py-3 px-4 font-medium">
                                <button
                                    data-testid="sort-name"
                                    class="flex items-center gap-1 hover:text-text transition-colors"
                                    onclick={() => toggleSort("name")}
                                >
                                    Name <sortIconName class="size-3"
                                    ></sortIconName>
                                </button>
                            </th>
                            <th class="py-3 px-4 font-medium">Type</th>
                            <th class="py-3 px-4 font-medium">Groups</th>
                            <th class="py-3 px-4 font-medium">
                                <button
                                    data-testid="sort-latency"
                                    class="flex items-center gap-1 hover:text-text transition-colors"
                                    onclick={() => toggleSort("latency")}
                                >
                                    Latency <sortIconLatency class="size-3"
                                    ></sortIconLatency>
                                </button>
                            </th>
                            <th class="py-3 px-4 font-medium text-right"
                                >Actions</th
                            >
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-border/60">
                        {#each filtered as monitor (monitor.id)}
                            <tr
                                data-testid={`monitor-row-${monitor.id}`}
                                class="hover:bg-surface/40 transition-colors group"
                            >
                                <td class="py-3 px-4">
                                    <Badge
                                        status={!monitor.enabled
                                            ? "paused"
                                            : monitor.status}
                                    />
                                </td>
                                <td class="py-3 px-4">
                                    <a
                                        href="/monitors/{monitor.id}"
                                        class="font-medium text-text hover:text-primary transition-colors"
                                    >
                                        {monitor.name}
                                    </a>
                                </td>
                                <td class="py-3 px-4">
                                    <span
                                        class="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-bold uppercase tracking-wider bg-surface-elevated border border-border text-text-muted"
                                    >
                                        {monitor.type}
                                    </span>
                                </td>
                                <td class="py-3 px-4">
                                    <div
                                        class="flex flex-wrap gap-1 items-center"
                                    >
                                        {#if monitor.groups && monitor.groups.length > 0}
                                            {#each monitor.groups as group}
                                                <span
                                                    class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-primary/10 text-primary border border-primary/20"
                                                >
                                                    {group}
                                                </span>
                                            {/each}
                                        {:else}
                                            <span class="text-text-subtle"
                                                >—</span
                                            >
                                        {/if}
                                    </div>
                                </td>
                                <td class="py-3 px-4 font-mono text-xs">
                                    {#if !monitor.enabled}
                                        <span class="text-text-subtle">—</span>
                                    {:else if monitor.last_latency_ms != null}
                                        <span
                                            class={monitor.last_latency_ms >
                                            1000
                                                ? "text-warning"
                                                : "text-text-muted"}
                                        >
                                            {monitor.last_latency_ms}ms
                                        </span>
                                    {:else}
                                        <span class="text-text-subtle">—</span>
                                    {/if}
                                </td>
                                <td class="py-3 px-4 text-right">
                                    <DropdownMenu.Root>
                                        <DropdownMenu.Trigger
                                            data-testid={`monitor-actions-${monitor.id}`}
                                            class="inline-flex items-center justify-center size-8 rounded-lg hover:bg-surface-elevated text-text-subtle hover:text-text transition-colors"
                                            aria-label={`Actions for ${monitor.name}`}
                                        >
                                            <svg
                                                class="size-4"
                                                viewBox="0 0 24 24"
                                                fill="currentColor"
                                            >
                                                <circle
                                                    cx="12"
                                                    cy="5"
                                                    r="1.5"
                                                /><circle
                                                    cx="12"
                                                    cy="12"
                                                    r="1.5"
                                                /><circle
                                                    cx="12"
                                                    cy="19"
                                                    r="1.5"
                                                />
                                            </svg>
                                        </DropdownMenu.Trigger>
                                        <DropdownMenu.Portal>
                                            <DropdownMenu.Content
                                                class="z-50 min-w-[10rem] rounded-xl border border-border bg-surface shadow-[0_8px_32px_hsl(224_71%_4%/0.5)] backdrop-blur-xl p-1 text-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
                                                sideOffset={4}
                                                align="end"
                                            >
                                                <DropdownMenu.Item
                                                    class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-text-muted hover:text-text hover:bg-surface-elevated cursor-pointer transition-colors outline-none"
                                                    onclick={() => {
                                                        void openEditMonitor(
                                                            monitor.id,
                                                        );
                                                    }}
                                                >
                                                    <Pencil class="size-3.5" /> Edit
                                                </DropdownMenu.Item>
                                                <DropdownMenu.Item
                                                    class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-text-muted hover:text-text hover:bg-surface-elevated cursor-pointer transition-colors outline-none"
                                                    onclick={() =>
                                                        togglePause(
                                                            monitor.id,
                                                            monitor.enabled,
                                                        )}
                                                >
                                                    {#if monitor.enabled}
                                                        <Pause
                                                            class="size-3.5"
                                                        /> Pause
                                                    {:else}
                                                        <Play
                                                            class="size-3.5"
                                                        /> Resume
                                                    {/if}
                                                </DropdownMenu.Item>
                                                <DropdownMenu.Separator
                                                    class="my-1 h-px bg-border"
                                                />
                                                <DropdownMenu.Item
                                                    class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-danger hover:bg-danger/10 cursor-pointer transition-colors outline-none"
                                                    onclick={() =>
                                                        deleteMonitor(
                                                            monitor.id,
                                                        )}
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
