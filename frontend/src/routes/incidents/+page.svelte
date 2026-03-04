<script lang="ts">
    import { onMount } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import {
        Plus,
        TriangleAlert,
        CheckCircle2,
        Search,
        XCircle,
        RefreshCw,
        Clock,
        X,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import { Dialog } from "bits-ui";
    import { formatDistanceToNow } from "date-fns";

    let incidents = $state<any[]>([]);
    let loading = $state(true);
    let searchQuery = $state("");
    let dialogOpen = $state(false);
    let editTarget = $state<any>(null);

    let formTitle = $state("");
    let formStatus = $state("investigating");
    let formSeverity = $state("minor");
    let formDescription = $state("");
    let formMonitorIds = $state<string[]>([]);
    let formSaving = $state(false);
    let formError = $state("");
    let monitors = $state<any[]>([]);

    onMount(() => {
        loadIncidents();
        loadMonitors();
    });

    async function loadMonitors() {
        try {
            const data = await fetchAPI("/api/v1/monitors");
            monitors = data || [];
        } catch {
            /* ignore */
        }
    }

    async function loadIncidents() {
        try {
            loading = true;
            const data = await fetchAPI("/api/v1/incidents");
            incidents = data || [];
        } catch (e: any) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function openCreate() {
        editTarget = null;
        formTitle = "";
        formStatus = "investigating";
        formSeverity = "minor";
        formDescription = "";
        formMonitorIds = [];
        formError = "";
        dialogOpen = true;
    }

    function openEdit(inc: any) {
        editTarget = inc;
        formTitle = inc.title;
        formStatus = inc.status;
        formSeverity = inc.severity || "minor";
        formDescription = inc.description || "";
        formMonitorIds = inc.monitor_ids || [];
        formError = "";
        dialogOpen = true;
    }

    async function saveIncident() {
        if (!formTitle.trim()) {
            formError = "Title is required";
            return;
        }
        formSaving = true;
        formError = "";
        try {
            if (editTarget) {
                await fetchAPI(`/api/v1/incidents/${editTarget.id}`, {
                    method: "PUT",
                    body: JSON.stringify({
                        title: formTitle,
                        status: formStatus,
                        severity: formSeverity,
                        description: formDescription,
                        monitor_ids: formMonitorIds,
                    }),
                });
            } else {
                await fetchAPI("/api/v1/incidents", {
                    method: "POST",
                    body: JSON.stringify({
                        title: formTitle,
                        status: formStatus,
                        severity: formSeverity,
                        description: formDescription,
                        monitor_ids: formMonitorIds,
                    }),
                });
            }
            dialogOpen = false;
            loadIncidents();
        } catch (e: any) {
            formError = e.message || "Failed to save incident";
        } finally {
            formSaving = false;
        }
    }

    async function deleteIncident(id: string) {
        if (!confirm("Delete this incident?")) return;
        await fetchAPI(`/api/v1/incidents/${id}`, { method: "DELETE" });
        loadIncidents();
    }

    // Status badge styles
    const statusBadgeStyle: Record<string, string> = {
        investigating: "border-warning/20 bg-warning/10 text-warning",
        identified: "border-orange-400/20 bg-orange-400/10 text-orange-400",
        monitoring: "border-primary/20 bg-primary/10 text-primary",
        resolved: "border-success/20 bg-success/10 text-success",
    };

    const statusLabel: Record<string, string> = {
        investigating: "Investigating",
        identified: "Identified",
        monitoring: "Monitoring",
        resolved: "Resolved",
    };

    const filtered = $derived(
        incidents.filter(
            (i) =>
                i.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                i.status?.toLowerCase().includes(searchQuery.toLowerCase()),
        ),
    );
</script>

<svelte:head>
    <title>Incidents – updu</title>
</svelte:head>

<div class="space-y-5 max-w-4xl">
    <div
        class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
    >
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Incidents
            </h1>
            <p class="text-sm text-text-muted mt-1">
                Track and resolve service disruptions
            </p>
        </div>
        <Button onclick={openCreate}>
            <Plus class="size-4" />
            Report Incident
        </Button>
    </div>

    <!-- Search -->
    <div class="relative max-w-xs">
        <Search
            class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-text-subtle pointer-events-none"
        />
        <input
            type="text"
            placeholder="Search incidents..."
            bind:value={searchQuery}
            class="input-base pl-9 h-9 text-xs"
        />
    </div>

    <!-- List card -->
    <div class="card overflow-hidden" style="padding: 0;">
        {#if loading}
            <div class="divide-y divide-border">
                {#each { length: 4 } as _}
                    <div class="p-5 flex gap-4">
                        <Skeleton
                            height="h-6"
                            width="w-28"
                            rounded="rounded-full"
                        />
                        <div class="flex-1 space-y-2">
                            <Skeleton height="h-4" width="w-2/3" />
                            <Skeleton height="h-3" width="w-1/2" />
                        </div>
                    </div>
                {/each}
            </div>
        {:else if filtered.length === 0}
            <div
                class="flex flex-col items-center justify-center text-center py-16 px-8 gap-4"
            >
                <div
                    class="size-16 rounded-2xl bg-surface flex items-center justify-center border border-border text-text-subtle"
                >
                    <TriangleAlert class="size-8" />
                </div>
                <div class="space-y-1.5">
                    <h3 class="text-base font-semibold text-text">
                        {searchQuery
                            ? `No incidents matching "${searchQuery}"`
                            : "No incidents"}
                    </h3>
                    <p class="text-sm text-text-muted max-w-xs">
                        {searchQuery
                            ? "Try a different search term."
                            : "All systems are operational 🎉"}
                    </p>
                </div>
                {#if !searchQuery}
                    <Button onclick={openCreate} variant="outline" size="sm"
                        >Report incident</Button
                    >
                {/if}
            </div>
        {:else}
            <div class="divide-y divide-border/60">
                {#each filtered as inc (inc.id)}
                    <div
                        class="p-5 hover:bg-surface/30 transition-colors flex items-start gap-4 group"
                    >
                        <!-- Status badge — rendered inline to avoid dynamic component typing issues -->
                        <div class="shrink-0 mt-0.5">
                            <span
                                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-[11px] font-semibold uppercase tracking-wider {statusBadgeStyle[
                                    inc.status
                                ] ??
                                    'border-border bg-surface text-text-muted'}"
                            >
                                {#if inc.status === "investigating" || inc.status === "identified"}
                                    <TriangleAlert class="size-3" />
                                {:else if inc.status === "monitoring"}
                                    <RefreshCw class="size-3" />
                                {:else if inc.status === "resolved"}
                                    <CheckCircle2 class="size-3" />
                                {:else}
                                    <Clock class="size-3" />
                                {/if}
                                {statusLabel[inc.status] ?? inc.status}
                            </span>
                        </div>

                        <div class="flex-1 min-w-0">
                            <h3 class="font-semibold text-text text-sm">
                                {inc.title}
                            </h3>
                            {#if inc.description}
                                <p
                                    class="text-xs text-text-muted mt-1 line-clamp-2"
                                >
                                    {inc.description}
                                </p>
                            {/if}
                            <p class="text-[11px] text-text-subtle mt-2">
                                {formatDistanceToNow(new Date(inc.created_at), {
                                    addSuffix: true,
                                })}
                            </p>
                        </div>

                        <div
                            class="shrink-0 flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                            <button
                                onclick={() => openEdit(inc)}
                                class="px-3 py-1.5 text-xs rounded-lg border border-border text-text-muted hover:text-text hover:border-border/80 transition-colors hover:bg-surface-elevated"
                                >Update</button
                            >
                            <button
                                onclick={() => deleteIncident(inc.id)}
                                class="size-7 flex items-center justify-center text-text-subtle hover:text-danger rounded-lg hover:bg-danger/10 transition-colors"
                            >
                                <XCircle class="size-4" />
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<!-- Create/Edit Dialog -->
<Dialog.Root
    bind:open={dialogOpen}
    onOpenChange={(v) => {
        if (!v) formError = "";
    }}
>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-center justify-between mb-5">
                <div>
                    <Dialog.Title class="text-base font-semibold text-text">
                        {editTarget ? "Update Incident" : "Report Incident"}
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        {editTarget
                            ? "Update the status and details."
                            : "Document a new service disruption."}
                    </Dialog.Description>
                </div>
                <Dialog.Close
                    class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                >
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if formError}
                <div
                    class="mb-4 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm"
                >
                    {formError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="inc-title"
                        >Title <span class="text-danger">*</span></label
                    >
                    <input
                        id="inc-title"
                        type="text"
                        bind:value={formTitle}
                        placeholder="Brief description of the incident"
                        class="input-base"
                    />
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="inc-status">Status</label
                    >
                    <select
                        id="inc-status"
                        bind:value={formStatus}
                        class="input-base bg-background/50"
                    >
                        <option value="investigating">Investigating</option>
                        <option value="identified">Identified</option>
                        <option value="monitoring">Monitoring</option>
                        <option value="resolved">Resolved</option>
                    </select>
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="inc-severity">Severity</label
                    >
                    <select
                        id="inc-severity"
                        bind:value={formSeverity}
                        class="input-base bg-background/50"
                    >
                        <option value="minor">Minor</option>
                        <option value="major">Major</option>
                        <option value="critical">Critical</option>
                    </select>
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="inc-desc">Description</label
                    >
                    <textarea
                        id="inc-desc"
                        bind:value={formDescription}
                        rows={4}
                        placeholder="Additional details, impact, affected systems..."
                        class="input-base h-auto py-2.5 resize-none"
                    ></textarea>
                </div>
                {#if monitors.length > 0}
                    <div class="space-y-1.5">
                        <p class="text-sm font-medium text-text-muted">
                            Affected Monitors
                        </p>
                        <div
                            class="max-h-28 overflow-y-auto rounded-lg border border-border bg-background/50 px-3 py-2 space-y-1"
                        >
                            {#each monitors as m}
                                <label
                                    class="flex items-center gap-2 text-sm cursor-pointer py-0.5"
                                >
                                    <input
                                        type="checkbox"
                                        value={m.id}
                                        checked={formMonitorIds.includes(m.id)}
                                        onchange={(e: Event) => {
                                            const target =
                                                e.target as HTMLInputElement;
                                            if (target.checked) {
                                                formMonitorIds = [
                                                    ...formMonitorIds,
                                                    m.id,
                                                ];
                                            } else {
                                                formMonitorIds =
                                                    formMonitorIds.filter(
                                                        (id: string) =>
                                                            id !== m.id,
                                                    );
                                            }
                                        }}
                                        class="rounded border-border"
                                    />
                                    <span class="text-text-muted">{m.name}</span
                                    >
                                </label>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button variant="outline" onclick={() => (dialogOpen = false)}
                    >Cancel</Button
                >
                <Button loading={formSaving} onclick={saveIncident}>
                    {formSaving
                        ? "Saving..."
                        : editTarget
                          ? "Update"
                          : "Report"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
