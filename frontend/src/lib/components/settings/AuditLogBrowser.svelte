<script lang="ts">
    import { onMount } from "svelte";
    import { format, formatDistanceToNow } from "date-fns";
    import { fetchAPI } from "$lib/api/client";
    import Button from "$lib/components/ui/button.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import { History, RefreshCcw } from "lucide-svelte";

    type AuditLog = {
        id: number;
        actor_type: string;
        actor_id: string;
        actor_name: string;
        action: string;
        resource_type: string;
        resource_id: string;
        summary?: string;
        created_at: string;
    };

    interface Props {
        refreshVersion?: number;
    }

    let { refreshVersion = 0 }: Props = $props();

    let auditLogs = $state<AuditLog[]>([]);
    let loading = $state(true);
    let error = $state("");
    let actionFilter = $state("");
    let resourceFilter = $state("");
    let limit = $state("25");
    let hasMounted = false;

    async function loadAuditLogs() {
        loading = true;
        error = "";

        try {
            const params = new URLSearchParams();
            if (actionFilter.trim()) {
                params.set("action", actionFilter.trim());
            }
            if (resourceFilter.trim()) {
                params.set("resource_type", resourceFilter.trim());
            }
            params.set("limit", limit);

            const query = params.toString();
            auditLogs = (await fetchAPI(`/api/v1/audit-logs?${query}`)) || [];
        } catch (e: any) {
            error = e.message || "Failed to load audit logs";
            auditLogs = [];
        } finally {
            loading = false;
        }
    }

    function formatRelativeTime(value: string): string {
        return formatDistanceToNow(new Date(value), { addSuffix: true });
    }

    function formatAbsoluteTime(value: string): string {
        return format(new Date(value), "PPpp");
    }

    onMount(async () => {
        hasMounted = true;
        await loadAuditLogs();
    });

    $effect(() => {
        if (!hasMounted || refreshVersion === 0) {
            return;
        }

        refreshVersion;
        void loadAuditLogs();
    });
</script>

<div class="card space-y-5">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <History class="size-4 text-primary" />
            </div>
            <div>
                <h3 class="text-sm font-semibold text-text">Audit Log</h3>
                <p class="text-[11px] text-text-subtle mt-0.5 max-w-2xl">
                    Browse configuration changes, who made them, and which resources were touched.
                </p>
            </div>
        </div>

        <Button size="sm" variant="outline" onclick={loadAuditLogs}>
            <RefreshCcw class="size-4" />
            Refresh
        </Button>
    </div>

    <form
        class="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_8rem_auto]"
        onsubmit={(event) => {
            event.preventDefault();
            void loadAuditLogs();
        }}
    >
        <div class="space-y-1.5">
            <label for="audit-action-filter" class="text-xs font-medium text-text-muted">
                Action
            </label>
            <input
                id="audit-action-filter"
                bind:value={actionFilter}
                class="input-base"
                placeholder="api_token.create"
            />
        </div>
        <div class="space-y-1.5">
            <label for="audit-resource-filter" class="text-xs font-medium text-text-muted">
                Resource Type
            </label>
            <input
                id="audit-resource-filter"
                bind:value={resourceFilter}
                class="input-base"
                placeholder="monitor"
            />
        </div>
        <div class="space-y-1.5">
            <label for="audit-limit-filter" class="text-xs font-medium text-text-muted">
                Limit
            </label>
            <select id="audit-limit-filter" bind:value={limit} class="input-base">
                <option value="10">10</option>
                <option value="25">25</option>
                <option value="50">50</option>
                <option value="100">100</option>
            </select>
        </div>
        <div class="flex items-end">
            <Button type="submit" size="sm" variant="outline" class="w-full md:w-auto">
                Apply Filters
            </Button>
        </div>
    </form>

    {#if error}
        <div class="p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm">
            {error}
        </div>
    {/if}

    <div class="space-y-3" data-testid="audit-log-list">
        {#if loading}
            <div class="space-y-3">
                {#each { length: 4 } as _}
                    <div class="rounded-2xl border border-border/60 p-4 space-y-3">
                        <Skeleton height="h-4" width="w-1/4" />
                        <Skeleton height="h-3" width="w-2/3" />
                        <Skeleton height="h-3" width="w-1/2" />
                    </div>
                {/each}
            </div>
        {:else if auditLogs.length === 0}
            <EmptyState
                icon={History}
                title="No audit entries matched"
                description="Try broader filters or perform an admin action to populate the log."
            />
        {:else}
            {#each auditLogs as entry (entry.id)}
                <article
                    class="rounded-2xl border border-border/60 bg-surface/20 p-4 space-y-3"
                    data-testid="audit-log-entry"
                >
                    <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                        <div class="space-y-1">
                            <p class="font-mono text-xs text-primary">{entry.action}</p>
                            <p class="text-sm font-semibold text-text">
                                {entry.summary || `${entry.actor_name} changed ${entry.resource_type}`}
                            </p>
                        </div>
                        <time
                            class="text-xs text-text-subtle shrink-0"
                            title={formatAbsoluteTime(entry.created_at)}
                        >
                            {formatRelativeTime(entry.created_at)}
                        </time>
                    </div>

                    <div class="grid gap-2 text-xs text-text-muted md:grid-cols-3">
                        <div>
                            <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Actor</span>
                            <span class="text-text">{entry.actor_name}</span>
                        </div>
                        <div>
                            <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Resource</span>
                            <span class="text-text">{entry.resource_type}</span>
                        </div>
                        <div>
                            <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Resource ID</span>
                            <span class="font-mono text-text break-all">{entry.resource_id}</span>
                        </div>
                    </div>
                </article>
            {/each}
        {/if}
    </div>
</div>