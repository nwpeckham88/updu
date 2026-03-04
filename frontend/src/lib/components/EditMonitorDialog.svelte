<script lang="ts">
    import { Dialog } from "bits-ui";
    import {
        X,
        Globe,
        Network,
        Activity,
        Radar,
        ShieldCheck,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import { fetchAPI } from "$lib/api/client";
    import { monitorsStore } from "$lib/stores/monitors.svelte";

    let { open = $bindable(false), monitor = $bindable(null) } = $props<{
        open: boolean;
        monitor: any;
    }>();

    let loading = $state(false);
    let errorMsg = $state("");

    let name = $state("");
    let groupName = $state("Core");
    let type = $state<"http" | "tcp" | "ping" | "dns" | "ssl">("http");
    let host = $state("");
    let intervalS = $state(60);
    let method = $state("GET");
    let expectedStatus = $state(200);
    let port = $state(80);
    let recordType = $state("A");
    let resolver = $state("");
    let expected = $state("");
    let sslPort = $state(443);
    let daysBeforeExpiry = $state(7);

    $effect(() => {
        if (monitor && open) {
            name = monitor.name;
            groupName = monitor.group_name ?? monitor.group ?? "Core";
            type = monitor.type;
            intervalS = monitor.interval_s || 60;

            if (type === "http") {
                host = monitor.config?.url || "";
                method = monitor.config?.method || "GET";
                expectedStatus = monitor.config?.expected_status || 200;
            } else if (type === "tcp") {
                host = monitor.config?.host || "";
                port = monitor.config?.port || 80;
            } else if (type === "ping") {
                host = monitor.config?.host || "";
            } else if (type === "dns") {
                host = monitor.config?.host || "";
                recordType = monitor.config?.record_type || "A";
                resolver = monitor.config?.resolver || "";
                expected = monitor.config?.expected || "";
            } else if (type === "ssl") {
                host = monitor.config?.host || "";
                sslPort = monitor.config?.port || 443;
                daysBeforeExpiry = monitor.config?.days_before_expiry || 7;
            }
            errorMsg = "";
        }
    });

    async function handleSubmit(e: Event) {
        e.preventDefault();
        if (!monitor) return;
        loading = true;
        errorMsg = "";

        let config: Record<string, any> = {};
        if (type === "http") {
            if (!host.startsWith("http")) host = "https://" + host;
            config = { url: host, method, expected_status: expectedStatus };
        } else if (type === "tcp") {
            config = { host, port };
        } else if (type === "ping") {
            config = { host };
        } else if (type === "dns") {
            config = { host, record_type: recordType };
            if (resolver) config.resolver = resolver;
            if (expected) config.expected = expected;
        } else if (type === "ssl") {
            config = {
                host,
                port: sslPort,
                days_before_expiry: daysBeforeExpiry,
            };
        }

        try {
            await fetchAPI(`/api/v1/monitors/${monitor.id}`, {
                method: "PUT",
                body: JSON.stringify({
                    name,
                    type,
                    group_name: groupName,
                    interval_s: intervalS,
                    enabled: monitor.enabled,
                    config,
                }),
            });
            open = false;
            monitorsStore.init();
        } catch (err: any) {
            errorMsg = err.message || "Failed to update monitor";
        } finally {
            loading = false;
        }
    }

    const typeOptions = [
        { value: "http", label: "HTTP", icon: Globe, desc: "Web endpoints" },
        { value: "tcp", label: "TCP", icon: Network, desc: "Port checks" },
        { value: "ping", label: "Ping", icon: Activity, desc: "ICMP ping" },
        { value: "dns", label: "DNS", icon: Radar, desc: "DNS records" },
        { value: "ssl", label: "SSL", icon: ShieldCheck, desc: "Cert expiry" },
    ] as const;
</script>

<Dialog.Root bind:open>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-center justify-between mb-5">
                <div>
                    <Dialog.Title class="text-base font-semibold text-text"
                        >Edit Monitor</Dialog.Title
                    >
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        Update configuration for {monitor?.name ||
                            "this monitor"}.
                    </Dialog.Description>
                </div>
                <Dialog.Close
                    class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                >
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if monitor}
                <form onsubmit={handleSubmit} class="space-y-4">
                    {#if errorMsg}
                        <div
                            class="p-3 text-sm text-danger bg-danger/10 border border-danger/20 rounded-lg"
                        >
                            {errorMsg}
                        </div>
                    {/if}

                    <div class="grid grid-cols-2 gap-3">
                        <div class="space-y-1.5">
                            <label
                                for="em-name"
                                class="text-sm font-medium text-text-muted"
                                >Name <span class="text-danger">*</span></label
                            >
                            <input
                                id="em-name"
                                required
                                bind:value={name}
                                placeholder="e.g. Nextcloud UI"
                                class="input-base"
                            />
                        </div>
                        <div class="space-y-1.5">
                            <label
                                for="em-group"
                                class="text-sm font-medium text-text-muted"
                                >Group</label
                            >
                            <input
                                id="em-group"
                                bind:value={groupName}
                                placeholder="e.g. Core"
                                class="input-base"
                            />
                        </div>
                    </div>

                    <!-- Type selector -->
                    <div class="space-y-1.5">
                        <p class="text-sm font-medium text-text-muted">
                            Monitor Type
                        </p>
                        <div class="grid grid-cols-5 gap-2">
                            {#each typeOptions as opt}
                                <button
                                    type="button"
                                    onclick={() => (type = opt.value)}
                                    class="flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 transition-all duration-150 {type ===
                                    opt.value
                                        ? 'border-primary/50 bg-primary/10 text-primary shadow-[0_0_16px_hsl(217_91%_60%/0.1)]'
                                        : 'border-border bg-surface-elevated/50 text-text-muted hover:text-text hover:border-border'}"
                                >
                                    <opt.icon class="size-5" />
                                    <span
                                        class="text-[11px] font-bold uppercase tracking-wider"
                                        >{opt.label}</span
                                    >
                                    <span class="text-[10px] opacity-60"
                                        >{opt.desc}</span
                                    >
                                </button>
                            {/each}
                        </div>
                    </div>

                    <!-- Host / URL -->
                    <div class="space-y-1.5">
                        <label
                            for="em-host"
                            class="text-sm font-medium text-text-muted"
                        >
                            {type === "http"
                                ? "URL"
                                : type === "ssl"
                                  ? "Hostname"
                                  : "Host / IP"}
                            <span class="text-danger">*</span>
                        </label>
                        <input
                            id="em-host"
                            required
                            bind:value={host}
                            placeholder={type === "http"
                                ? "https://example.com"
                                : type === "dns" || type === "ssl"
                                  ? "example.com"
                                  : "192.168.1.1"}
                            class="input-base"
                        />
                    </div>

                    <!-- HTTP options -->
                    {#if type === "http"}
                        <div
                            class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-method"
                                    class="text-sm font-medium text-text-muted"
                                    >HTTP Method</label
                                >
                                <select
                                    id="em-method"
                                    bind:value={method}
                                    class="input-base bg-background/50"
                                >
                                    {#each ["GET", "POST", "PUT", "HEAD"] as m}
                                        <option value={m}>{m}</option>
                                    {/each}
                                </select>
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-status"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected Status</label
                                >
                                <input
                                    id="em-status"
                                    type="number"
                                    bind:value={expectedStatus}
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- TCP options -->
                    {#if type === "tcp"}
                        <div class="pl-4 border-l-2 border-primary/20 py-1">
                            <div class="space-y-1.5">
                                <label
                                    for="em-port"
                                    class="text-sm font-medium text-text-muted"
                                    >Port <span class="text-danger">*</span
                                    ></label
                                >
                                <input
                                    id="em-port"
                                    type="number"
                                    required
                                    bind:value={port}
                                    placeholder="3306"
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- DNS options -->
                    {#if type === "dns"}
                        <div
                            class="grid grid-cols-3 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-record"
                                    class="text-sm font-medium text-text-muted"
                                    >Record Type</label
                                >
                                <select
                                    id="em-record"
                                    bind:value={recordType}
                                    class="input-base bg-background/50"
                                >
                                    {#each ["A", "AAAA", "CNAME", "MX", "TXT", "NS"] as rt}
                                        <option value={rt}>{rt}</option>
                                    {/each}
                                </select>
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-resolver"
                                    class="text-sm font-medium text-text-muted"
                                    >Resolver</label
                                >
                                <input
                                    id="em-resolver"
                                    bind:value={resolver}
                                    placeholder="8.8.8.8"
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-expected"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected</label
                                >
                                <input
                                    id="em-expected"
                                    bind:value={expected}
                                    placeholder="1.2.3.4"
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- SSL options -->
                    {#if type === "ssl"}
                        <div
                            class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-ssl-port"
                                    class="text-sm font-medium text-text-muted"
                                    >Port</label
                                >
                                <input
                                    id="em-ssl-port"
                                    type="number"
                                    bind:value={sslPort}
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-ssl-days"
                                    class="text-sm font-medium text-text-muted"
                                    >Warn before (days)</label
                                >
                                <input
                                    id="em-ssl-days"
                                    type="number"
                                    bind:value={daysBeforeExpiry}
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- Interval -->
                    <div class="space-y-1.5">
                        <div class="flex items-center justify-between">
                            <label
                                for="em-interval"
                                class="text-sm font-medium text-text-muted"
                                >Check Interval</label
                            >
                            <span
                                class="text-xs font-mono bg-surface-elevated px-2 py-0.5 rounded-md border border-border text-text"
                                >{intervalS}s</span
                            >
                        </div>
                        <input
                            id="em-interval"
                            type="range"
                            min="10"
                            max="3600"
                            step="10"
                            bind:value={intervalS}
                            class="w-full appearance-none h-1.5 rounded-full bg-border accent-primary cursor-pointer"
                        />
                        <div
                            class="flex justify-between text-[10px] text-text-subtle"
                        >
                            <span>10s</span><span>1h</span>
                        </div>
                    </div>

                    <div class="flex gap-2 justify-end pt-2">
                        <Button
                            type="button"
                            variant="outline"
                            onclick={() => (open = false)}>Cancel</Button
                        >
                        <Button type="submit" {loading}>
                            {loading ? "Saving..." : "Update Monitor"}
                        </Button>
                    </div>
                </form>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
