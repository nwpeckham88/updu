<script lang="ts">
    import {
        Plus,
        Search,
        Globe,
        Lock,
        ExternalLink,
        Pencil,
        Trash2,
        LayoutTemplate,
        X,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import { Dialog } from "bits-ui";
    import { fetchAPI } from "$lib/api/client";
    import { onMount } from "svelte";
    import { monitorsStore } from "$lib/stores/monitors.svelte";

    let pages = $state<any[]>([]);
    let loading = $state(true);
    let searchQuery = $state("");
    let dialogOpen = $state(false);
    let editTarget = $state<any>(null);

    // Form state
    let formName = $state("");
    let formSlug = $state("");
    let formDescription = $state("");
    let formIsPublic = $state(true);
    let formGroups = $state<string[]>([]);
    let formStandaloneMonitors = $state<string[]>([]);
    let formSaving = $state(false);
    let formError = $state("");

    onMount(() => {
        loadPages();
        monitorsStore.init();
    });

    const monitors = $derived(monitorsStore.monitors);
    const availableGroups = $derived(
        Array.from(
            new Set(
                monitors
                    .map((m) => m.group_name)
                    .filter((g) => g && g !== "Core" && g !== ""),
            ),
        ).sort(),
    );

    async function loadPages() {
        try {
            loading = true;
            const data = await fetchAPI("/api/v1/status-pages");
            pages = data || [];
        } catch (e) {
            // API may not be ready — use empty list
            pages = [];
        } finally {
            loading = false;
        }
    }

    function openCreate() {
        editTarget = null;
        formName = "";
        formSlug = "";
        formDescription = "";
        formIsPublic = true;
        formGroups = [];
        formStandaloneMonitors = [];
        formError = "";
        dialogOpen = true;
    }

    function openEdit(p: any) {
        editTarget = p;
        formName = p.name;
        formSlug = p.slug;
        formDescription = p.description || "";
        formIsPublic = p.is_public;

        let groups: string[] = [];
        let standalone: string[] = [];
        for (const g of p.groups || []) {
            if (g.name && g.name !== "") {
                groups.push(g.name);
            } else if (!g.name && g.monitor_ids) {
                standalone.push(...g.monitor_ids);
            }
        }
        formGroups = groups;
        formStandaloneMonitors = standalone;

        formError = "";
        dialogOpen = true;
    }

    async function savePage() {
        if (!formName.trim() || !formSlug.trim()) {
            formError = "Name and slug are required";
            return;
        }
        formSaving = true;
        formError = "";
        try {
            const compiledGroups = formGroups.map((name) => ({
                name,
                monitor_ids: [] as string[],
            }));
            if (formStandaloneMonitors.length > 0) {
                compiledGroups.push({
                    name: "",
                    monitor_ids: formStandaloneMonitors,
                });
            }

            if (editTarget) {
                await fetchAPI(`/api/v1/status-pages/${editTarget.id}`, {
                    method: "PUT",
                    body: JSON.stringify({
                        name: formName,
                        slug: formSlug,
                        description: formDescription,
                        is_public: formIsPublic,
                        groups: compiledGroups,
                    }),
                });
            } else {
                await fetchAPI("/api/v1/status-pages", {
                    method: "POST",
                    body: JSON.stringify({
                        name: formName,
                        slug: formSlug,
                        description: formDescription,
                        is_public: formIsPublic,
                        groups: compiledGroups,
                    }),
                });
            }
            dialogOpen = false;
            loadPages();
        } catch (e: any) {
            formError = e.message || "Failed to save status page";
        } finally {
            formSaving = false;
        }
    }

    async function deletePage(id: string) {
        if (!confirm("Delete this status page?")) return;
        try {
            await fetchAPI(`/api/v1/status-pages/${id}`, { method: "DELETE" });
            loadPages();
        } catch (e) {
            console.error("Failed to delete", e);
        }
    }

    const filteredPages = $derived(
        pages.filter(
            (p) =>
                p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                p.slug.toLowerCase().includes(searchQuery.toLowerCase()),
        ),
    );

    // Auto-generate slug from name
    $effect(() => {
        if (!editTarget && formName && !formSlug) {
            formSlug = formName
                .toLowerCase()
                .replace(/\s+/g, "-")
                .replace(/[^a-z0-9-]/g, "");
        }
    });
</script>

<svelte:head>
    <title>Status Pages – updu</title>
</svelte:head>

<div class="space-y-5 max-w-7xl">
    <div
        class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
    >
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Status Pages
            </h1>
            <p class="text-sm text-text-muted mt-1">
                Public dashboards for your services
            </p>
        </div>
        <Button onclick={openCreate}>
            <Plus class="size-4" />
            New Status Page
        </Button>
    </div>

    <!-- Search -->
    <div class="relative max-w-xs">
        <Search
            class="absolute left-3 top-1/2 -translate-y-1/2 size-3.5 text-text-subtle pointer-events-none"
        />
        <input
            type="text"
            placeholder="Search pages..."
            bind:value={searchQuery}
            class="input-base pl-9 h-9 text-xs"
        />
    </div>

    {#if loading}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {#each { length: 3 } as _}
                <div class="card p-5 space-y-3">
                    <div class="flex items-center gap-3">
                        <Skeleton
                            height="h-9"
                            width="w-9"
                            rounded="rounded-xl"
                        />
                        <div class="flex-1 space-y-2">
                            <Skeleton height="h-4" width="w-1/2" />
                            <Skeleton height="h-3" width="w-1/3" />
                        </div>
                    </div>
                    <Skeleton height="h-3" width="w-full" />
                    <Skeleton height="h-3" width="w-3/4" />
                </div>
            {/each}
        </div>
    {:else if filteredPages.length === 0}
        <div class="card">
            <EmptyState
                icon={LayoutTemplate}
                title={searchQuery
                    ? `No pages matching "${searchQuery}"`
                    : "No status pages yet"}
                description={searchQuery
                    ? "Try a different search term."
                    : "Create a page to share your service status publicly."}
            >
                {#if !searchQuery}
                    <Button onclick={openCreate}>Create Status Page</Button>
                {/if}
            </EmptyState>
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {#each filteredPages as p (p.id)}
                <div class="card p-5 flex flex-col gap-3 group">
                    <div class="flex items-start justify-between">
                        <div class="flex items-center gap-3">
                            <div
                                class="size-9 rounded-xl flex items-center justify-center shrink-0 {p.is_public
                                    ? 'bg-primary/10 text-primary'
                                    : 'bg-warning/10 text-warning'}"
                            >
                                {#if p.is_public}
                                    <Globe class="size-4" />
                                {:else}
                                    <Lock class="size-4" />
                                {/if}
                            </div>
                            <div>
                                <h3 class="text-sm font-semibold text-text">
                                    {p.name}
                                </h3>
                                <a
                                    href="/status/{p.slug}"
                                    target="_blank"
                                    class="flex items-center gap-1 text-[11px] text-primary hover:underline mt-0.5 group/link"
                                >
                                    /{p.slug}
                                    <ExternalLink
                                        class="size-2.5 opacity-0 group-hover/link:opacity-100 transition-opacity"
                                    />
                                </a>
                            </div>
                        </div>
                        <div
                            class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                            <button
                                onclick={() => openEdit(p)}
                                class="size-7 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-subtle hover:text-text transition-colors"
                                title="Edit"
                            >
                                <Pencil class="size-3.5" />
                            </button>
                            <button
                                onclick={() => deletePage(p.id)}
                                class="size-7 flex items-center justify-center rounded-lg hover:bg-danger/10 text-text-subtle hover:text-danger transition-colors"
                                title="Delete"
                            >
                                <Trash2 class="size-3.5" />
                            </button>
                        </div>
                    </div>

                    {#if p.description}
                        <p class="text-xs text-text-muted line-clamp-2">
                            {p.description}
                        </p>
                    {/if}

                    {#if p.groups?.length}
                        <div
                            class="flex flex-wrap gap-1.5 pt-2 border-t border-border/50"
                        >
                            {#each p.groups as group}
                                <span
                                    class="px-2 py-0.5 rounded-full bg-surface-elevated border border-border text-[10px] font-semibold uppercase tracking-wider text-text-muted"
                                >
                                    {group}
                                </span>
                            {/each}
                        </div>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
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
                        {editTarget ? "Edit Status Page" : "New Status Page"}
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        {editTarget
                            ? "Update the configuration for this page."
                            : "Create a shareable status page for your services."}
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
                        for="sp-name"
                        >Name <span class="text-danger">*</span></label
                    >
                    <input
                        id="sp-name"
                        type="text"
                        bind:value={formName}
                        placeholder="My Services"
                        class="input-base"
                    />
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="sp-slug"
                        >Slug <span class="text-danger">*</span></label
                    >
                    <div class="relative">
                        <span
                            class="absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle text-sm select-none"
                            >/</span
                        >
                        <input
                            id="sp-slug"
                            type="text"
                            bind:value={formSlug}
                            placeholder="my-services"
                            class="input-base pl-6"
                        />
                    </div>
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="sp-desc">Description</label
                    >
                    <textarea
                        id="sp-desc"
                        bind:value={formDescription}
                        rows={3}
                        placeholder="Brief description of this status page..."
                        class="input-base h-auto py-2.5 resize-none"
                    ></textarea>
                </div>
                <!-- Monitor & Group Selection -->
                <div class="space-y-4 pt-2 border-t border-border/50">
                    <div class="space-y-2">
                        <div>
                            <div class="text-sm font-medium text-text">
                                Include Groups
                            </div>
                            <p class="text-[11px] text-text-subtle mt-0.5">
                                Automatically include all monitors assigned to
                                these groups.
                            </p>
                        </div>
                        <div class="flex flex-wrap gap-2">
                            <!-- Always show explicit 'Core' group -->
                            <label
                                class="px-3 py-1.5 rounded-lg border text-xs font-semibold uppercase tracking-wider cursor-pointer transition-colors {formGroups.includes(
                                    'Core',
                                )
                                    ? 'bg-primary/10 border-primary/30 text-primary'
                                    : 'bg-surface-elevated border-border text-text-muted hover:border-text-subtle'}"
                            >
                                <input
                                    type="checkbox"
                                    value="Core"
                                    bind:group={formGroups}
                                    class="hidden"
                                />
                                CORE
                            </label>
                            {#each availableGroups as ag}
                                <label
                                    class="px-3 py-1.5 rounded-lg border text-xs font-semibold uppercase tracking-wider cursor-pointer transition-colors {formGroups.includes(
                                        ag,
                                    )
                                        ? 'bg-primary/10 border-primary/30 text-primary'
                                        : 'bg-surface-elevated border-border text-text-muted hover:border-text-subtle'}"
                                >
                                    <input
                                        type="checkbox"
                                        value={ag}
                                        bind:group={formGroups}
                                        class="hidden"
                                    />
                                    {ag}
                                </label>
                            {/each}
                        </div>
                    </div>

                    <div class="space-y-2">
                        <div>
                            <div class="text-sm font-medium text-text">
                                Include Standalone Monitors
                            </div>
                            <p class="text-[11px] text-text-subtle mt-0.5">
                                Display specific monitors independently of their
                                group structure.
                            </p>
                        </div>
                        <div class="max-h-40 overflow-y-auto pr-2 space-y-1">
                            {#each monitors as sm}
                                <label
                                    class="flex items-center gap-3 p-2 rounded-lg border cursor-pointer transition-colors {formStandaloneMonitors.includes(
                                        sm.id,
                                    )
                                        ? 'bg-primary/5 border-primary/20'
                                        : 'bg-surface border-transparent hover:bg-surface-elevated'}"
                                >
                                    <input
                                        type="checkbox"
                                        value={sm.id}
                                        bind:group={formStandaloneMonitors}
                                        class="rounded border-border text-primary focus:ring-primary size-3.5 bg-surface-elevated"
                                    />
                                    <div
                                        class="flex flex-1 items-center justify-between"
                                    >
                                        <span
                                            class="text-xs font-medium text-text"
                                            >{sm.name}</span
                                        >
                                        <span
                                            class="text-[10px] text-text-subtle uppercase tracking-wider"
                                            >{sm.group_name || "Core"}</span
                                        >
                                    </div>
                                </label>
                            {/each}
                        </div>
                    </div>
                </div>
                <label
                    class="flex items-center gap-3 cursor-pointer select-none"
                >
                    <div class="relative">
                        <input
                            type="checkbox"
                            bind:checked={formIsPublic}
                            class="sr-only peer"
                            id="sp-public"
                        />
                        <div
                            class="w-9 h-5 rounded-full border border-border bg-surface-elevated peer-checked:bg-primary peer-checked:border-primary transition-colors"
                        ></div>
                        <div
                            class="absolute top-0.5 left-0.5 size-4 rounded-full bg-white shadow transition-transform peer-checked:translate-x-4"
                        ></div>
                    </div>
                    <div>
                        <p class="text-sm font-medium text-text">
                            Public access
                        </p>
                        <p class="text-[11px] text-text-subtle">
                            Anyone can view this status page without login
                        </p>
                    </div>
                </label>
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button variant="outline" onclick={() => (dialogOpen = false)}
                    >Cancel</Button
                >
                <Button loading={formSaving} onclick={savePage}>
                    {formSaving
                        ? "Saving..."
                        : editTarget
                          ? "Save Changes"
                          : "Create Page"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
