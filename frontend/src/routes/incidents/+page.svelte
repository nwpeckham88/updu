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
        ChevronDown,
        ChevronRight,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import { Dialog } from "bits-ui";
    import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
    import { confirmAction } from "$lib/stores/confirm.svelte";
    import { formatDistanceToNow } from "date-fns";

    interface Incident {
        id: string;
        title: string;
        description?: string;
        status: string;
        severity?: string;
        monitor_ids?: string[];
        started_at?: string;
        created_at?: string;
        resolved_at?: string | null;
    }

    interface MonitorSummary {
        id: string;
        name: string;
        groups?: string[];
    }

    interface IncidentGroup {
        id: string;
        incidents: Incident[];
        keys: Set<string>;
        monitorIds: string[];
        groupNames: string[];
        anchorMs: number;
        durationLabel: string;
        status: string;
        severity: string;
    }

    let incidents = $state<Incident[]>([]);
    let loading = $state(true);
    let searchQuery = $state("");
    let showUngrouped = $state(false);
    let expandedGroups = $state<Record<string, boolean>>({});
    let dialogOpen = $state(false);
    let editTarget = $state<Incident | null>(null);

    let formTitle = $state("");
    let formStatus = $state("investigating");
    let formSeverity = $state("minor");
    let formDescription = $state("");
    let formMonitorIds = $state<string[]>([]);
    let formSaving = $state(false);
    let formError = $state("");
    let monitors = $state<MonitorSummary[]>([]);

    onMount(() => {
        loadIncidents();
        loadMonitors();

        const eventSource = new EventSource("/api/v1/events");
        eventSource.addEventListener("incident:change", () => {
            void loadIncidents();
        });

        return () => eventSource.close();
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
        const ok = await confirmAction({
            title: "Delete incident?",
            description:
                "This incident and its update history will be permanently removed.",
            confirmLabel: "Delete incident",
            variant: "destructive",
        });
        if (!ok) return;
        try {
            await fetchAPI(`/api/v1/incidents/${id}`, { method: "DELETE" });
            loadIncidents();
            toastStore.success("Incident deleted");
        } catch (e) {
            toastFromError(e, "Failed to delete incident");
        }
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

    const monitorById = $derived.by(() => {
        const byId = new Map<string, MonitorSummary>();
        for (const monitor of monitors) byId.set(monitor.id, monitor);
        return byId;
    });

    const filtered = $derived(
        incidents.filter(
            (i) =>
                i.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                i.status?.toLowerCase().includes(searchQuery.toLowerCase()),
        ),
    );

    function incidentStartedAt(incident: Incident): string | undefined {
        return incident.started_at ?? incident.created_at;
    }

    function incidentTimestamp(incident: Incident): number {
        const raw = incidentStartedAt(incident);
        const timestamp = raw ? new Date(raw).getTime() : 0;
        return Number.isFinite(timestamp) ? timestamp : 0;
    }

    function incidentKeys(incident: Incident): Set<string> {
        const keys = new Set<string>();
        for (const monitorId of incident.monitor_ids ?? []) {
            keys.add(`monitor:${monitorId}`);
            const monitor = monitorById.get(monitorId);
            for (const group of monitor?.groups ?? []) {
                keys.add(`group:${group.toLowerCase()}`);
            }
        }
        return keys;
    }

    function sharesAnyKey(a: Set<string>, b: Set<string>): boolean {
        for (const key of a) {
            if (b.has(key)) return true;
        }
        return false;
    }

    function incidentSeverityRank(severity: string | undefined): number {
        if (severity === "critical") return 0;
        if (severity === "major") return 1;
        return 2;
    }

    function incidentStatusRank(status: string): number {
        if (status === "investigating") return 0;
        if (status === "identified") return 1;
        if (status === "monitoring") return 2;
        return 3;
    }

    function durationLabel(startMs: number, endMs: number): string {
        const minutes = Math.max(1, Math.round((endMs - startMs) / 60000));
        if (minutes < 60) return `${minutes}m`;
        const hours = Math.floor(minutes / 60);
        const remaining = minutes % 60;
        if (hours < 24) return remaining > 0 ? `${hours}h ${remaining}m` : `${hours}h`;
        const days = Math.floor(hours / 24);
        return `${days}d`;
    }

    function makeIncidentGroup(incident: Incident): IncidentGroup {
        const keys = incidentKeys(incident);
        const monitorIds = [...new Set(incident.monitor_ids ?? [])];
        const groupNames = [...keys]
            .filter((key) => key.startsWith("group:"))
            .map((key) => key.slice(6));
        const anchorMs = incidentTimestamp(incident);
        const endMs = incident.resolved_at
            ? new Date(incident.resolved_at).getTime()
            : Date.now();

        return {
            id: incident.id,
            incidents: [incident],
            keys,
            monitorIds,
            groupNames,
            anchorMs,
            durationLabel: durationLabel(anchorMs, endMs),
            status: incident.status,
            severity: incident.severity ?? "minor",
        };
    }

    function refreshIncidentGroup(group: IncidentGroup): IncidentGroup {
        const monitorIds = [
            ...new Set(group.incidents.flatMap((incident) => incident.monitor_ids ?? [])),
        ];
        const groupNames = [...group.keys]
            .filter((key) => key.startsWith("group:"))
            .map((key) => key.slice(6));
        const startMs = Math.min(...group.incidents.map(incidentTimestamp));
        const endMs = Math.max(
            ...group.incidents.map((incident) =>
                incident.resolved_at ? new Date(incident.resolved_at).getTime() : Date.now(),
            ),
        );
        const status = [...group.incidents].sort(
            (a, b) => incidentStatusRank(a.status) - incidentStatusRank(b.status),
        )[0].status;
        const severity = [...group.incidents].sort(
            (a, b) => incidentSeverityRank(a.severity) - incidentSeverityRank(b.severity),
        )[0].severity ?? "minor";

        return {
            ...group,
            monitorIds,
            groupNames,
            anchorMs: startMs,
            durationLabel: durationLabel(startMs, endMs),
            status,
            severity,
        };
    }

    function buildCorrelatedGroups(list: Incident[]): IncidentGroup[] {
        const sorted = [...list].sort((a, b) => incidentTimestamp(b) - incidentTimestamp(a));
        const groups: IncidentGroup[] = [];

        for (const incident of sorted) {
            const keys = incidentKeys(incident);
            const group = groups.find(
                (candidate) =>
                    keys.size > 0 &&
                    Math.abs(incidentTimestamp(incident) - candidate.anchorMs) <= 60000 &&
                    sharesAnyKey(keys, candidate.keys),
            );

            if (!group) {
                groups.push(makeIncidentGroup(incident));
                continue;
            }

            group.incidents = [...group.incidents, incident].sort(
                (a, b) => incidentTimestamp(b) - incidentTimestamp(a),
            );
            for (const key of keys) group.keys.add(key);
        }

        return groups.map(refreshIncidentGroup);
    }

    const correlatedGroups = $derived.by(() => buildCorrelatedGroups(filtered));
    const visibleGroups = $derived.by(() =>
        showUngrouped ? filtered.map(makeIncidentGroup) : correlatedGroups,
    );

    function toggleGroup(id: string) {
        expandedGroups = { ...expandedGroups, [id]: !expandedGroups[id] };
    }

    function monitorNames(group: IncidentGroup): string {
        const names = group.monitorIds
            .map((id) => monitorById.get(id)?.name)
            .filter(Boolean);
        if (names.length === 0) return "No monitors linked";
        return names.join(", ");
    }

    function timelineTone(group: IncidentGroup): string {
        if (group.status === "resolved") return "bg-success/60";
        if (group.severity === "critical") return "bg-danger";
        if (group.severity === "major") return "bg-warning";
        return "bg-primary/70";
    }
</script>

<svelte:head>
    <title>Incidents – updu</title>
</svelte:head>

{#snippet incidentBadge(status: string)}
    <span
        class="type-kicker inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border {statusBadgeStyle[
            status
        ] ?? 'border-border bg-surface text-text-muted'}"
    >
        {#if status === "investigating" || status === "identified"}
            <TriangleAlert class="size-3" />
        {:else if status === "monitoring"}
            <RefreshCw class="size-3" />
        {:else if status === "resolved"}
            <CheckCircle2 class="size-3" />
        {:else}
            <Clock class="size-3" />
        {/if}
        {statusLabel[status] ?? status}
    </span>
{/snippet}

<div class="space-y-5 max-w-4xl">
    <div
        class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
    >
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Incidents
            </h1>
            <p class="type-caption text-text-muted mt-1">
                Track and resolve service disruptions
            </p>
        </div>
        <Button onclick={openCreate}>
            <Plus class="size-4" />
            Report Incident
        </Button>
    </div>

    <!-- Search -->
    <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="relative max-w-xs flex-1">
            <Search
                class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-text-subtle pointer-events-none"
            />
            <input
                type="text"
                placeholder="Search incidents..."
                bind:value={searchQuery}
                aria-label="Search incidents"
                class="input-base pl-9 h-9 text-xs"
            />
        </div>
        <label
            class="type-caption inline-flex items-center gap-2 rounded-lg border border-border bg-surface/40 px-3 py-2 font-medium text-text-muted"
        >
            <input type="checkbox" bind:checked={showUngrouped} class="rounded border-border" />
            Show ungrouped
        </label>
    </div>

    <!-- List card -->
    <div class="card overflow-hidden" style="padding: 0;">
        {#if loading}
            <div class="divide-y divide-border" aria-busy="true" aria-label="Loading incidents">
                {#each { length: 4 } as _, index (index)}
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
                    <h3 class="type-section-title text-text">
                        {searchQuery
                            ? `No incidents matching "${searchQuery}"`
                            : "No incidents"}
                    </h3>
                    <p class="type-caption text-text-muted max-w-xs">
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
                {#each visibleGroups as group (group.id)}
                    {@const primary = group.incidents[0]}
                    {@const isGrouped = group.incidents.length > 1}
                    <div class="group">
                        <div
                            class="p-5 transition-colors hover:bg-surface/30"
                        >
                            <div class="flex items-start gap-4">
                                <div class="shrink-0 mt-0.5 flex items-center gap-2">
                                    {#if isGrouped}
                                        <button
                                            type="button"
                                            onclick={() => toggleGroup(group.id)}
                                            class="inline-flex size-7 items-center justify-center rounded-lg border border-border text-text-muted transition-colors hover:bg-surface-elevated hover:text-text"
                                            aria-expanded={!!expandedGroups[group.id]}
                                            aria-label={`${expandedGroups[group.id] ? "Collapse" : "Expand"} correlated incident group`}
                                        >
                                            {#if expandedGroups[group.id]}
                                                <ChevronDown class="size-4" />
                                            {:else}
                                                <ChevronRight class="size-4" />
                                            {/if}
                                        </button>
                                    {:else}
                                        <span class="size-7" aria-hidden="true"></span>
                                    {/if}
                                    {@render incidentBadge(group.status)}
                                </div>

                                <div class="min-w-0 flex-1">
                                    <div class="flex flex-wrap items-center gap-2">
                                        <h3 class="type-data-title text-text">
                                            {isGrouped ? `${group.incidents.length} correlated incidents` : primary.title}
                                        </h3>
                                        <span class="type-kicker rounded-full border border-border bg-surface px-2 py-0.5 text-text-subtle">
                                            {group.monitorIds.length || "No"} monitor{group.monitorIds.length === 1 ? "" : "s"} · {group.durationLabel}
                                        </span>
                                    </div>
                                    <p class="type-caption mt-1 text-text-muted line-clamp-2">
                                        {isGrouped
                                            ? monitorNames(group)
                                            : primary.description || monitorNames(group)}
                                    </p>
                                    {#if group.groupNames.length > 0}
                                        <p class="type-micro mt-1 text-text-subtle">
                                            Shared group: {group.groupNames.join(", ")}
                                        </p>
                                    {/if}
                                    <div
                                        class="mt-3 flex h-2 max-w-md gap-1"
                                        aria-label={`Timeline for ${monitorNames(group)}`}
                                    >
                                        {#each (group.monitorIds.length > 0 ? group.monitorIds : [group.id]) as monitorId (monitorId)}
                                            <span
                                                class={`h-2 flex-1 rounded-full ${timelineTone(group)}`}
                                                title={monitorById.get(monitorId)?.name ?? "Unlinked incident"}
                                            ></span>
                                        {/each}
                                    </div>
                                    <p class="type-micro mt-2 text-text-subtle">
                                        {formatDistanceToNow(new Date(incidentStartedAt(primary) ?? Date.now()), {
                                            addSuffix: true,
                                        })}
                                    </p>
                                </div>

                                {#if !isGrouped}
                                    <div
                                        class="shrink-0 flex items-center gap-1.5 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
                                    >
                                        <button
                                            onclick={() => openEdit(primary)}
                                            class="px-3 py-1.5 text-xs rounded-lg border border-border text-text-muted hover:text-text hover:border-border/80 transition-colors hover:bg-surface-elevated"
                                            >Update</button
                                        >
                                        <button
                                            onclick={() => deleteIncident(primary.id)}
                                            class="size-7 flex items-center justify-center text-text-subtle hover:text-danger rounded-lg hover:bg-danger/10 transition-colors"
                                            aria-label={`Delete incident ${primary.title}`}
                                        >
                                            <XCircle class="size-4" />
                                        </button>
                                    </div>
                                {/if}
                            </div>
                        </div>

                        {#if isGrouped && expandedGroups[group.id]}
                            <div class="border-t border-border/60 bg-surface/20 px-5 py-3">
                                <div class="space-y-2 pl-11">
                                    {#each group.incidents as inc (inc.id)}
                                        <div class="flex items-center gap-3 rounded-lg border border-border/60 bg-background/40 px-3 py-2">
                                            {@render incidentBadge(inc.status)}
                                            <div class="min-w-0 flex-1">
                                                <p class="type-data-title truncate text-text">{inc.title}</p>
                                                <p class="type-micro text-text-subtle">
                                                    {formatDistanceToNow(new Date(incidentStartedAt(inc) ?? Date.now()), {
                                                        addSuffix: true,
                                                    })}
                                                </p>
                                            </div>
                                            <button
                                                onclick={() => openEdit(inc)}
                                                class="px-3 py-1.5 text-xs rounded-lg border border-border text-text-muted hover:text-text hover:border-border/80 transition-colors hover:bg-surface-elevated"
                                                >Update</button
                                            >
                                            <button
                                                onclick={() => deleteIncident(inc.id)}
                                                class="size-7 flex items-center justify-center text-text-subtle hover:text-danger rounded-lg hover:bg-danger/10 transition-colors"
                                                aria-label={`Delete incident ${inc.title}`}
                                            >
                                                <XCircle class="size-4" />
                                            </button>
                                        </div>
                                    {/each}
                                </div>
                            </div>
                        {/if}
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
                            {#each monitors as m (m.id)}
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
