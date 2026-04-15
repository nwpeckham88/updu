<script lang="ts">
    import { invalidateAll } from "$app/navigation";
    import {
        CheckCircle2,
        AlertTriangle,
        XCircle,
        Activity,
        Globe,
        Lock,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";

    let { data } = $props<{ data: any }>();
    let password = $state("");
    let unlockError = $state("");
    let unlocking = $state(false);

    let sp = $derived(data.page);
    let allMonitors = $derived(data.monitors || []);
    let locked = $derived(Boolean(data.locked));

    function monitorGroups(monitor: any): string[] {
        return monitor.groups && monitor.groups.length > 0
            ? monitor.groups
            : ["Core"];
    }

    // Filter monitors assigned to this page
    let relevantMonitors = $derived(
        allMonitors.filter((m: any) => {
            if (!sp?.groups) return false;
            return sp.groups.some((g: any) => {
                if (g.name && monitorGroups(m).includes(g.name)) return true;
                if (!g.name && g.monitor_ids?.includes(m.id)) return true;
                return false;
            });
        }),
    );

    let monitorBucketKeys = $state(new Map<string, string>());

    $effect(() => {
        const buckets = new Map<string, string>();

        for (const monitor of relevantMonitors) {
            const bucket = sp?.groups?.find(
                (g: any) =>
                    (g.name && monitorGroups(monitor).includes(g.name)) ||
                    (!g.name && g.monitor_ids?.includes(monitor.id)),
            );
            if (!bucket) {
                continue;
            }

            buckets.set(
                monitor.id,
                bucket.name ? `group:${bucket.name}` : "standalone",
            );
        }

        monitorBucketKeys = buckets;
    });

    let isAllUp = $derived(
        relevantMonitors.length > 0 &&
            relevantMonitors.every((m: any) => m.status === "up"),
    );
    let hasDown = $derived(
        relevantMonitors.some((m: any) => m.status === "down"),
    );

    async function unlockPage(event: SubmitEvent) {
        event.preventDefault();
        if (!password.trim()) {
            unlockError = "Password is required";
            return;
        }

        unlocking = true;
        unlockError = "";

        try {
            const response = await fetch(`/api/v1/status-pages/${data.slug}/unlock`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ password }),
            });
            const payload = await response.json().catch(() => null);
            if (!response.ok) {
                unlockError = payload?.error || "Failed to unlock status page";
                return;
            }

            password = "";
            await invalidateAll();
        } catch {
            unlockError = "Failed to unlock status page";
        } finally {
            unlocking = false;
        }
    }
</script>

<svelte:head>
    <title>{sp?.name ? `${sp.name} Status` : locked ? "Protected Status Page" : "Status Page"}</title>
</svelte:head>

{#if locked}
    <div class="max-w-md mx-auto py-16 px-4 sm:px-6 lg:px-8">
        <div class="rounded-3xl border border-border bg-surface/95 p-8 shadow-[0_24px_64px_hsl(224_71%_4%/0.16)] space-y-6">
            <div class="space-y-3 text-center">
                <div class="mx-auto size-14 rounded-2xl bg-warning/10 border border-warning/20 flex items-center justify-center text-warning">
                    <Lock class="size-6" />
                </div>
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-text">Protected Status Page</h1>
                    <p class="text-sm text-text-muted mt-2">
                        Enter the page password to view current service status.
                    </p>
                </div>
            </div>

            <form class="space-y-4" onsubmit={unlockPage}>
                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="status-page-password">
                        Password
                    </label>
                    <input
                        id="status-page-password"
                        type="password"
                        bind:value={password}
                        class="input-base"
                        autocomplete="current-password"
                        placeholder="Enter password"
                    />
                </div>

                {#if unlockError}
                    <div class="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
                        {unlockError}
                    </div>
                {/if}

                <Button class="w-full" loading={unlocking} type="submit">
                    {unlocking ? "Unlocking..." : "Unlock Status Page"}
                </Button>
            </form>
        </div>
    </div>
{:else}
    <div class="max-w-4xl mx-auto py-10 px-4 sm:px-6 lg:px-8 space-y-10">
        <div class="text-center space-y-3 pt-6 pb-4 border-b border-border/50">
            <h1 class="text-3xl font-extrabold tracking-tight text-text">
                {sp.name}
            </h1>
            {#if sp.description}
                <p class="text-sm text-text-muted max-w-2xl mx-auto">
                    {sp.description}
                </p>
            {/if}
        </div>

        {#if relevantMonitors.length === 0}
            <div
                class="p-6 rounded-2xl bg-surface-elevated border border-border flex items-center justify-center gap-3"
            >
                <Activity class="size-6 text-text-muted" />
                <h2 class="text-lg font-medium text-text-muted">
                    No services configured
                </h2>
            </div>
        {:else if isAllUp}
            <div
                class="p-6 rounded-2xl bg-success/10 border border-success/20 flex items-center gap-4 text-success shadow-[0_0_24px_hsl(142_71%_45%/0.1)]"
            >
                <CheckCircle2 class="size-8 shrink-0" />
                <div>
                    <h2 class="text-xl font-bold">All Systems Operational</h2>
                    <p class="text-sm opacity-80 mt-0.5">
                        Everything is functioning normally.
                    </p>
                </div>
            </div>
        {:else if hasDown}
            <div
                class="p-6 rounded-2xl bg-danger/10 border border-danger/20 flex items-center gap-4 text-danger shadow-[0_0_24px_hsl(0_84%_60%/0.1)]"
            >
                <XCircle class="size-8 shrink-0" />
                <div>
                    <h2 class="text-xl font-bold">Partial System Outage</h2>
                    <p class="text-sm opacity-80 mt-0.5">
                        Some services are currently experiencing issues.
                    </p>
                </div>
            </div>
        {:else}
            <div
                class="p-6 rounded-2xl bg-warning/10 border border-warning/20 flex items-center gap-4 text-warning shadow-[0_0_24px_hsl(38_92%_50%/0.1)]"
            >
                <AlertTriangle class="size-8 shrink-0" />
                <div>
                    <h2 class="text-xl font-bold">Degraded Performance</h2>
                    <p class="text-sm opacity-80 mt-0.5">
                        Some services are experiencing delayed response times.
                    </p>
                </div>
            </div>
        {/if}

        <div class="space-y-8">
            {#if sp.groups}
                {#each sp.groups as group}
                    {@const nameStr = group.name}
                    {@const isStandalone = !nameStr}
                    {@const bucketKey = isStandalone ? 'standalone' : `group:${nameStr}`}
                    {@const groupMonitors = relevantMonitors.filter(
                        (m: any) => monitorBucketKeys.get(m.id) === bucketKey,
                    )}

                    {#if groupMonitors.length > 0}
                        <div class="space-y-4">
                            {#if !isStandalone}
                                <h3
                                    class="text-lg font-bold text-text border-b border-border/50 pb-2"
                                >
                                    {nameStr}
                                </h3>
                            {/if}

                            <div class="flex flex-col gap-3">
                                {#each groupMonitors as m}
                                    <div
                                        class="p-4 rounded-xl bg-surface border border-border/50 flex flex-col sm:flex-row sm:items-center justify-between gap-4 transition-colors hover:bg-surface-elevated/50"
                                    >
                                        <div class="flex items-center gap-3">
                                            <div
                                                class="size-8 rounded-lg bg-surface-elevated/80 flex items-center justify-center shrink-0 border border-border"
                                            >
                                                {#if m.type === "http" || m.type === "json" || m.type === "ssl"}
                                                    <Globe
                                                        class="size-3.5 text-text-muted"
                                                    />
                                                {:else}
                                                    <Activity
                                                        class="size-3.5 text-text-muted"
                                                    />
                                                {/if}
                                            </div>
                                            <div>
                                                <p
                                                    class="text-sm font-semibold text-text"
                                                >
                                                    {m.name}
                                                </p>
                                                {#if m.uptime_24h != null}
                                                    <p
                                                        class="text-[11px] text-text-subtle mt-0.5"
                                                    >
                                                        {m.uptime_24h.toFixed(4)}%
                                                        uptime over 24h
                                                    </p>
                                                {/if}
                                            </div>
                                        </div>
                                        <div class="flex items-center gap-2">
                                            {#if m.status === "up"}
                                                <span
                                                    class="text-[11px] font-bold uppercase tracking-wider text-success flex items-center gap-1.5"
                                                >
                                                    <div
                                                        class="size-1.5 bg-success rounded-full shadow-[0_0_6px_currentColor]"
                                                    ></div>
                                                    Operational
                                                </span>
                                            {:else if m.status === "down"}
                                                <span
                                                    class="text-[11px] font-bold uppercase tracking-wider text-danger flex items-center gap-1.5"
                                                >
                                                    <div
                                                        class="size-1.5 bg-danger rounded-full shadow-[0_0_6px_currentColor]"
                                                    ></div>
                                                    Outage
                                                </span>
                                            {:else if m.status === "degraded"}
                                                <span
                                                    class="text-[11px] font-bold uppercase tracking-wider text-warning flex items-center gap-1.5"
                                                >
                                                    <div
                                                        class="size-1.5 bg-warning rounded-full shadow-[0_0_6px_currentColor]"
                                                    ></div>
                                                    Degraded
                                                </span>
                                            {:else}
                                                <span
                                                    class="text-[11px] font-bold uppercase tracking-wider text-text-muted flex items-center gap-1.5"
                                                >
                                                    <div
                                                        class="size-1.5 bg-text-subtle rounded-full"
                                                    ></div>
                                                    Unknown
                                                </span>
                                            {/if}
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}
                {/each}
            {/if}
        </div>
    </div>
{/if}
