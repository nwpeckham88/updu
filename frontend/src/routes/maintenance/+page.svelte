<script lang="ts">
    import { onMount } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import {
        Plus,
        Search,
        Wrench,
        Clock,
        CheckCircle2,
        PlayCircle,
        CalendarClock,
        Pencil,
        Trash2,
        X,
        RefreshCw,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import { Dialog } from "bits-ui";

    let windows = $state<any[]>([]);
    let loading = $state(true);
    let searchQuery = $state("");
    let dialogOpen = $state(false);
    let editTarget = $state<any>(null);

    // Form state
    let formTitle = $state("");
    let formStartsAt = $state("");
    let formEndsAt = $state("");
    let formRecurring = $state("");
    let formMonitorIds = $state<string[]>([]);
    let formSaving = $state(false);
    let formError = $state("");
    let monitors = $state<any[]>([]);

    onMount(() => {
        loadWindows();
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

    async function loadWindows() {
        try {
            loading = true;
            const data = await fetchAPI("/api/v1/maintenance");
            windows = data || [];
        } catch {
            windows = [];
        } finally {
            loading = false;
        }
    }

    function openCreate() {
        editTarget = null;
        formTitle = "";
        formStartsAt = "";
        formEndsAt = "";
        formRecurring = "";
        formMonitorIds = [];
        formError = "";
        dialogOpen = true;
    }

    function openEdit(mw: any) {
        editTarget = mw;
        formTitle = mw.title;
        formStartsAt = toDatetimeLocal(mw.starts_at);
        formEndsAt = toDatetimeLocal(mw.ends_at);
        formRecurring = mw.recurring || "";
        formMonitorIds = mw.monitor_ids || [];
        formError = "";
        dialogOpen = true;
    }

    function toDatetimeLocal(iso: string): string {
        if (!iso) return "";
        const d = new Date(iso);
        const pad = (n: number) => String(n).padStart(2, "0");
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    async function saveWindow() {
        if (!formTitle.trim()) {
            formError = "Title is required";
            return;
        }
        if (!formStartsAt || !formEndsAt) {
            formError = "Start and end times are required";
            return;
        }
        formSaving = true;
        formError = "";
        try {
            const body = {
                title: formTitle,
                starts_at: new Date(formStartsAt).toISOString(),
                ends_at: new Date(formEndsAt).toISOString(),
                recurring: formRecurring || null,
                monitor_ids: formMonitorIds,
            };
            if (editTarget) {
                await fetchAPI(`/api/v1/maintenance/${editTarget.id}`, {
                    method: "PUT",
                    body: JSON.stringify(body),
                });
            } else {
                await fetchAPI("/api/v1/maintenance", {
                    method: "POST",
                    body: JSON.stringify(body),
                });
            }
            dialogOpen = false;
            loadWindows();
        } catch (e: any) {
            formError = e.message || "Failed to save maintenance window";
        } finally {
            formSaving = false;
        }
    }

    async function deleteWindow(id: string) {
        if (!confirm("Delete this maintenance window?")) return;
        await fetchAPI(`/api/v1/maintenance/${id}`, { method: "DELETE" });
        loadWindows();
    }

    function getStatus(mw: any): { label: string; classes: string; icon: any } {
        const now = Date.now();
        const start = new Date(mw.starts_at).getTime();
        const end = new Date(mw.ends_at).getTime();
        if (now < start) {
            return {
                label: "Scheduled",
                classes: "bg-primary/10 text-primary border-primary/20",
                icon: CalendarClock,
            };
        }
        if (now >= start && now <= end) {
            return {
                label: "In Progress",
                classes: "bg-warning/10 text-warning border-warning/20",
                icon: PlayCircle,
            };
        }
        return {
            label: "Completed",
            classes: "bg-success/10 text-success border-success/20",
            icon: CheckCircle2,
        };
    }

    function formatDate(iso: string): string {
        return new Date(iso).toLocaleString(undefined, {
            month: "short",
            day: "numeric",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    const recurringLabels: Record<string, string> = {
        daily: "Daily",
        weekly: "Weekly",
        monthly: "Monthly",
    };

    const filtered = $derived(
        windows.filter((w) =>
            w.title?.toLowerCase().includes(searchQuery.toLowerCase()),
        ),
    );
</script>

<svelte:head>
    <title>Maintenance – updu</title>
</svelte:head>

<div class="space-y-5 max-w-4xl">
    <div
        class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
    >
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Maintenance
            </h1>
            <p class="text-sm text-text-muted mt-1">
                Schedule downtime windows for planned work
            </p>
        </div>
        <Button onclick={openCreate}>
            <Plus class="size-4" />
            New Window
        </Button>
    </div>

    <!-- Search -->
    <div class="relative max-w-xs">
        <Search
            class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-text-subtle pointer-events-none"
        />
        <input
            type="text"
            placeholder="Search maintenance..."
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
                            height="h-10"
                            width="w-10"
                            rounded="rounded-xl"
                        />
                        <div class="flex-1 space-y-2">
                            <Skeleton height="h-4" width="w-2/3" />
                            <Skeleton height="h-3" width="w-1/2" />
                        </div>
                    </div>
                {/each}
            </div>
        {:else if filtered.length === 0}
            <EmptyState
                icon={Wrench}
                title={searchQuery
                    ? `No windows matching "${searchQuery}"`
                    : "No maintenance windows"}
                description={searchQuery
                    ? "Try a different search term."
                    : "Schedule downtime for planned maintenance."}
            >
                {#if !searchQuery}
                    <Button onclick={openCreate} variant="outline" size="sm"
                        >Schedule maintenance</Button
                    >
                {/if}
            </EmptyState>
        {:else}
            <div class="divide-y divide-border/60">
                {#each filtered as mw (mw.id)}
                    {@const s = getStatus(mw)}
                    <div
                        class="p-5 hover:bg-surface/30 transition-colors flex items-start gap-4 group"
                    >
                        <!-- Status badge -->
                        <div class="shrink-0 mt-0.5">
                            <span
                                class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[10px] font-semibold uppercase tracking-wider {s.classes}"
                            >
                                <s.icon class="size-3" />
                                {s.label}
                            </span>
                        </div>

                        <div class="flex-1 min-w-0">
                            <h3 class="font-semibold text-text text-sm">
                                {mw.title}
                            </h3>
                            <p
                                class="text-[11px] text-text-subtle mt-1 flex items-center gap-1.5 flex-wrap"
                            >
                                <Clock class="size-3 shrink-0" />
                                {formatDate(mw.starts_at)} → {formatDate(
                                    mw.ends_at,
                                )}
                                {#if mw.recurring}
                                    <span
                                        class="inline-flex items-center gap-1 ml-2"
                                    >
                                        <RefreshCw class="size-2.5" />
                                        {recurringLabels[mw.recurring] ??
                                            mw.recurring}
                                    </span>
                                {/if}
                            </p>
                        </div>

                        <div
                            class="shrink-0 flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                            <button
                                onclick={() => openEdit(mw)}
                                class="size-7 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-subtle hover:text-text transition-colors"
                                title="Edit"
                            >
                                <Pencil class="size-3.5" />
                            </button>
                            <button
                                onclick={() => deleteWindow(mw.id)}
                                class="size-7 flex items-center justify-center rounded-lg hover:bg-danger/10 text-text-subtle hover:text-danger transition-colors"
                                title="Delete"
                            >
                                <Trash2 class="size-3.5" />
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<!-- Create/Edit Dialog -->
<Dialog.Root bind:open={dialogOpen}>
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
                        {editTarget
                            ? "Edit Maintenance Window"
                            : "New Maintenance Window"}
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        {editTarget
                            ? "Update the schedule for this window."
                            : "Schedule a downtime window for planned work."}
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
                        for="mw-title"
                        >Title <span class="text-danger">*</span></label
                    >
                    <input
                        id="mw-title"
                        type="text"
                        bind:value={formTitle}
                        placeholder="Server upgrade"
                        class="input-base"
                    />
                </div>
                <div class="grid grid-cols-2 gap-3">
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="mw-start"
                            >Starts at <span class="text-danger">*</span></label
                        >
                        <input
                            id="mw-start"
                            type="datetime-local"
                            bind:value={formStartsAt}
                            class="input-base text-xs"
                        />
                    </div>
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="mw-end"
                            >Ends at <span class="text-danger">*</span></label
                        >
                        <input
                            id="mw-end"
                            type="datetime-local"
                            bind:value={formEndsAt}
                            class="input-base text-xs"
                        />
                    </div>
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="mw-recurring">Recurring</label
                    >
                    <select
                        id="mw-recurring"
                        bind:value={formRecurring}
                        class="input-base text-sm"
                    >
                        <option value="">None (one-time)</option>
                        <option value="daily">Daily</option>
                        <option value="weekly">Weekly</option>
                        <option value="monthly">Monthly</option>
                    </select>
                </div>
                {#if monitors.length > 0}
                    <div class="space-y-1.5">
                        <label class="text-sm font-medium text-text-muted"
                            >Affected Monitors</label
                        >
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
                <Button loading={formSaving} onclick={saveWindow}>
                    {formSaving
                        ? "Saving..."
                        : editTarget
                          ? "Save Changes"
                          : "Schedule"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
