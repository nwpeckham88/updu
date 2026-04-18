<script lang="ts">
    import { Dialog } from "bits-ui";
    import {
        X,
        Globe,
        Network,
        Activity,
        Radar,
        ShieldCheck,
        Terminal,
        Braces,
        CloudOff,
        ArrowRightLeft,
        Mail,
        Radio,
        Database,
        Zap,
        Lock,
        Layers,
        List,
        Search,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import { fetchAPI } from "$lib/api/client";
    import { monitorsStore } from "$lib/stores/monitors.svelte";
    import { afterNextPaint } from "$lib/utils";

    let { open = $bindable(false), monitor = $bindable(null) } = $props<{
        open: boolean;
        monitor: any;
    }>();

    let loading = $state(false);
    let errorMsg = $state("");

    let name = $state("");
    let groups = $state<string[]>([]);
    let newGroup = $state("");
    let allGroups = $state<string[]>([]);
    let type = $state<
        | "http"
        | "tcp"
        | "ping"
        | "dns"
        | "ssl"
        | "ssh"
        | "json"
        | "push"
        | "websocket"
        | "smtp"
        | "udp"
        | "redis"
        | "postgres"
        | "mysql"
        | "mongo"
        | "https"
        | "composite"
        | "transaction"
        | "dns_http"
    >("http");
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
    let sshPort = $state(22);
    let jsonField = $state("");
    let jsonExpectedValue = $state("");
    let token = $state("");
    let sendPayload = $state("");
    let expectedResponse = $state("");
    let dbPassword = $state("");
    let dbIndex = $state(0);
    let connString = $state("");
    let requireTls = $state(false);
    // Compound monitor fields
    let httpsWarnDays = $state(14);
    let compositeMonitorIDs = $state("");
    let compositeMode = $state("all_up");
    let compositeQuorum = $state(1);
    let transactionStepsJSON = $state(
        '[\n  {"url": "https://example.com", "method": "GET"}\n]',
    );
    let transactionSkipTLS = $state(false);
    let dnsHTTPExpectedIPPrefix = $state("");
    let dnsHTTPExpectedCNAME = $state("");
    let dnsHTTPExpectedBody = $state("");
    let dnsHTTPSkipTLS = $state(false);
    let dnsHTTPExpectedStatus = $state(200);
    let formReady = $state(false);
    let cancelDeferredOpen: (() => void) | null = null;
    let groupsWarning = $state("");

    function generateToken() {
        const charset = "abcdef0123456789";
        let res = "";
        for (let i = 0; i < 32; i++) {
            res += charset.charAt(Math.floor(Math.random() * charset.length));
        }
        token = res;
    }

    function parseMonitorConfig(config: unknown): Record<string, any> {
        if (typeof config === "string") {
            try {
                return JSON.parse(config);
            } catch (error) {
                console.warn("Failed to parse monitor config", error);
                return {};
            }
        }
        return typeof config === "object" && config !== null
            ? (config as Record<string, any>)
            : {};
    }

    const pingUrl = $derived(
        monitor
            ? `${window.location.origin}/api/v1/heartbeat/${monitor.id}`
            : "",
    );

    function populateMonitorForm(sourceMonitor: any) {
        const config = parseMonitorConfig(sourceMonitor.config);
        name = sourceMonitor.name;
        if (sourceMonitor.groups && sourceMonitor.groups.length > 0) {
            groups = sourceMonitor.groups;
        } else {
            const legacy = sourceMonitor.group_name ?? sourceMonitor.group;
            groups = legacy ? [legacy] : ["Core"];
        }
        type = sourceMonitor.type;
        intervalS = sourceMonitor.interval_s || 60;

        if (type === "http") {
            host = config.url || "";
            method = config.method || "GET";
            expectedStatus = config.expected_status || 200;
        } else if (type === "tcp") {
            host = config.host || "";
            port = config.port || 80;
        } else if (type === "ping") {
            host = config.host || "";
        } else if (type === "dns") {
            host = config.host || "";
            recordType = config.record_type || "A";
            resolver = config.resolver || "";
            expected = config.expected || "";
        } else if (type === "ssl") {
            host = config.host || "";
            sslPort = config.port || 443;
            daysBeforeExpiry = config.days_before_expiry || 7;
        } else if (type === "ssh") {
            host = config.host || "";
            sshPort = config.port || 22;
        } else if (type === "json") {
            host = config.url || "";
            method = config.method || "GET";
            jsonField = config.field || "";
            jsonExpectedValue = config.expected_value || "";
        } else if (type === "push") {
            token = config.token || "";
        } else if (type === "websocket") {
            host = config.url || "";
        } else if (type === "smtp") {
            host = config.host || "";
            port = config.port || 587;
            requireTls = config.require_tls || false;
        } else if (type === "udp") {
            host = config.host || "";
            port = config.port || 0;
            sendPayload = config.send_payload || "";
            expectedResponse = config.expected_response || "";
        } else if (type === "redis") {
            host = config.host || "";
            port = config.port || 6379;
            dbPassword = config.password || "";
            dbIndex = config.database || 0;
        } else if (
            type === "postgres" ||
            type === "mysql" ||
            type === "mongo"
        ) {
            connString = config.connection_string || "";
        } else if (type === "https") {
            host = config.url || "";
            method = config.method || "GET";
            expectedStatus = config.expected_status || 200;
            httpsWarnDays = config.warn_days || 14;
        } else if (type === "composite") {
            compositeMonitorIDs = (config.monitor_ids || []).join(", ");
            compositeMode = config.mode || "all_up";
            compositeQuorum = config.quorum || 1;
        } else if (type === "transaction") {
            transactionStepsJSON = JSON.stringify(config.steps || [], null, 2);
            transactionSkipTLS = config.skip_tls_verify || false;
        } else if (type === "dns_http") {
            host = config.url || "";
            dnsHTTPExpectedIPPrefix = config.expected_ip_prefix || "";
            dnsHTTPExpectedCNAME = config.expected_cname || "";
            dnsHTTPExpectedBody = config.expected_body || "";
            dnsHTTPSkipTLS = config.skip_tls_verify || false;
            dnsHTTPExpectedStatus = config.expected_status || 200;
        }
        errorMsg = "";
        groupsWarning = "";
    }

    function clearDeferredOpen() {
        cancelDeferredOpen?.();
        cancelDeferredOpen = null;
    }

    $effect(() => {
        clearDeferredOpen();

        if (!open || !monitor) {
            formReady = false;
            return;
        }

        const pendingMonitor = monitor;
        const pendingMonitorID = monitor.id;
        formReady = false;
        errorMsg = "";
        groupsWarning = "";
        cancelDeferredOpen = afterNextPaint(() => {
            if (!open || monitor?.id !== pendingMonitorID) {
                return;
            }
            populateMonitorForm(pendingMonitor);
            formReady = true;
            void fetchGroups();
        });

        return () => {
            clearDeferredOpen();
        };
    });

    async function fetchGroups() {
        try {
            const data = await fetchAPI("/api/v1/groups");
            allGroups = Array.isArray(data) ? data : [];
            groupsWarning = "";
        } catch (err) {
            console.error("Failed to fetch groups", err);
            allGroups = [];
            groupsWarning =
                "Failed to load saved groups. You can still type group names manually.";
        }
    }

    function addGroup() {
        const g = newGroup.trim();
        if (g && !groups.includes(g)) {
            groups = [...groups, g];
            newGroup = "";
        }
    }

    function removeGroup(g: string) {
        groups = groups.filter((item) => item !== g);
    }

    async function handleSubmit(e: Event) {
        e.preventDefault();
        if (!monitor) return;
        loading = true;
        errorMsg = "";

        let config: Record<string, any> = {};
        if (type === "http") {
            let url = host;
            if (!url.startsWith("http")) url = "https://" + url;
            config = { url, method, expected_status: expectedStatus };
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
        } else if (type === "ssh") {
            config = { host, port: sshPort };
        } else if (type === "json") {
            let url = host;
            if (!url.startsWith("http")) url = "https://" + url;
            config = {
                url,
                method,
                field: jsonField,
                expected_value: jsonExpectedValue,
            };
        } else if (type === "push") {
            config = { token };
        } else if (type === "websocket") {
            let url = host;
            if (!url.startsWith("ws")) url = "wss://" + url;
            config = { url };
        } else if (type === "smtp") {
            config = { host, port, require_tls: requireTls };
        } else if (type === "udp") {
            config = { host, port };
            if (sendPayload) config.send_payload = sendPayload;
            if (expectedResponse) config.expected_response = expectedResponse;
        } else if (type === "redis") {
            config = { host, port, database: dbIndex };
            if (dbPassword) config.password = dbPassword;
        } else if (
            type === "postgres" ||
            type === "mysql" ||
            type === "mongo"
        ) {
            config = { connection_string: connString };
        } else if (type === "https") {
            let url = host;
            if (!url.startsWith("http")) url = "https://" + url;
            config = {
                url,
                method,
                expected_status: expectedStatus,
                warn_days: httpsWarnDays,
            };
        } else if (type === "composite") {
            config = {
                monitor_ids: compositeMonitorIDs
                    .split(",")
                    .map((s: string) => s.trim())
                    .filter(Boolean),
                mode: compositeMode,
                quorum: compositeQuorum,
            };
        } else if (type === "transaction") {
            let steps: unknown[] = [];
            try {
                steps = JSON.parse(transactionStepsJSON);
            } catch {}
            config = { steps, skip_tls_verify: transactionSkipTLS };
        } else if (type === "dns_http") {
            let url = host;
            if (!url.startsWith("http")) url = "https://" + url;
            config = {
                url,
                expected_ip_prefix: dnsHTTPExpectedIPPrefix,
                expected_cname: dnsHTTPExpectedCNAME,
                expected_status: dnsHTTPExpectedStatus,
                skip_tls_verify: dnsHTTPSkipTLS,
            };
            if (dnsHTTPExpectedBody) config.expected_body = dnsHTTPExpectedBody;
        }

        try {
            await fetchAPI(`/api/v1/monitors/${monitor.id}`, {
                method: "PUT",
                body: JSON.stringify({
                    name,
                    type,
                    groups: groups,
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
        { value: "ssh", label: "SSH", icon: Terminal, desc: "SSH banner" },
        { value: "json", label: "JSON", icon: Braces, desc: "API fields" },
        { value: "push", label: "Push", icon: CloudOff, desc: "Heartbeat API" },
        {
            value: "websocket",
            label: "WS",
            icon: ArrowRightLeft,
            desc: "WebSocket",
        },
        { value: "smtp", label: "SMTP", icon: Mail, desc: "Mail server" },
        { value: "udp", label: "UDP", icon: Radio, desc: "UDP port" },
        { value: "redis", label: "Redis", icon: Database, desc: "Redis DB" },
        {
            value: "postgres",
            label: "PgSQL",
            icon: Database,
            desc: "Postgres DB",
        },
        { value: "mysql", label: "MySQL", icon: Database, desc: "MySQL DB" },
        { value: "mongo", label: "Mongo", icon: Database, desc: "MongoDB" },
        { value: "https", label: "HTTPS", icon: Lock, desc: "HTTP+TLS" },
        { value: "composite", label: "Comp.", icon: Layers, desc: "K-of-N" },
        {
            value: "transaction",
            label: "Chain",
            icon: List,
            desc: "Multi-step",
        },
        {
            value: "dns_http",
            label: "DNS+HTTP",
            icon: Search,
            desc: "DNS+HTTP",
        },
    ] as const;
</script>

<Dialog.Root
    bind:open
    onOpenChange={(value) => {
        if (!value) {
            clearDeferredOpen();
            formReady = false;
        }
    }}
>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/60 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface p-6 shadow-[0_18px_48px_hsl(224_71%_4%/0.45)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
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

            {#if monitor && formReady}
                <div class="sr-only" aria-live="polite">Edit monitor form ready.</div>
                <form onsubmit={handleSubmit} class="space-y-4">
                    {#if errorMsg}
                        <div
                            class="p-3 text-sm text-danger bg-danger/10 border border-danger/20 rounded-lg"
                        >
                            {errorMsg}
                        </div>
                    {/if}

                    {#if groupsWarning}
                        <div
                            class="p-3 text-sm text-warning bg-warning/10 border border-warning/20 rounded-lg"
                        >
                            {groupsWarning}
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
                        <div class="space-y-1.5 col-span-2">
                            <span
                                class="block text-sm font-medium text-text-muted"
                                >Groups</span
                            >
                            <div class="space-y-2">
                                <div
                                    class="flex flex-wrap gap-1.5 min-h-[36px] p-1.5 bg-surface-elevated/50 border border-border rounded-lg"
                                >
                                    {#each groups as group}
                                        <span
                                            class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-primary/10 text-primary border border-primary/20"
                                        >
                                            {group}
                                            <button
                                                type="button"
                                                onclick={() =>
                                                    removeGroup(group)}
                                                class="hover:text-primary-light transition-colors"
                                            >
                                                <X class="size-3" />
                                            </button>
                                        </span>
                                    {/each}
                                    <input
                                        bind:value={newGroup}
                                        placeholder={groups.length === 0
                                            ? "Add groups..."
                                            : ""}
                                        onkeydown={(e) => {
                                            if (e.key === "Enter") {
                                                e.preventDefault();
                                                addGroup();
                                            }
                                        }}
                                        onblur={addGroup}
                                        class="bg-transparent border-none outline-none text-xs flex-1 min-w-[120px] placeholder:text-text-subtle/50"
                                    />
                                </div>

                                {#if allGroups.length > 0}
                                    <div class="flex flex-wrap gap-1">
                                        {#each allGroups.filter((g) => !groups.includes(g)) as g}
                                            <button
                                                type="button"
                                                onclick={() => {
                                                    groups = [...groups, g];
                                                }}
                                                class="text-[10px] px-2 py-0.5 rounded border border-border bg-surface-elevated hover:bg-surface-elevated/80 text-text-subtle transition-colors"
                                            >
                                                + {g}
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
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

                    <!-- Host / URL / Token / ConnString -->
                    {#if type !== "composite" && type !== "transaction"}
                    <div class="space-y-1.5">
                        <label
                            for="em-host"
                            class="text-sm font-medium text-text-muted"
                        >
                            {type === "http" || type === "json" || type === "https" || type === "dns_http"
                                ? "URL"
                                : type === "dns"
                                  ? "Domain Name"
                                  : type === "ssl"
                                    ? "Hostname"
                                    : type === "push"
                                      ? "Token"
                                      : type === "websocket"
                                        ? "WebSocket URL"
                                        : type === "postgres" ||
                                            type === "mysql" ||
                                            type === "mongo"
                                          ? "Connection String"
                                          : "Host / IP"}
                            <span class="text-danger">*</span>
                        </label>
                        {#if type === "push"}
                            <div class="flex gap-2">
                                <input
                                    id="em-host"
                                    required
                                    bind:value={token}
                                    placeholder="Secret token"
                                    class="input-base font-mono text-xs"
                                />
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onclick={generateToken}
                                    class="shrink-0"
                                >
                                    <Zap class="size-3.5 mr-1.5" />
                                    Regenerate
                                </Button>
                            </div>
                            <div
                                class="mt-3 p-3 rounded-lg bg-surface-elevated/50 border border-border"
                            >
                                <p
                                    class="text-[11px] font-medium text-text-muted mb-1.5 uppercase tracking-wider"
                                >
                                    Ping URL
                                </p>
                                <div class="flex items-center gap-2">
                                    <code
                                        class="text-[10px] text-primary break-all bg-primary/5 px-2 py-1 rounded border border-primary/10 flex-1"
                                    >
                                        {pingUrl}?token={token}
                                    </code>
                                    <button
                                        type="button"
                                        class="p-1.5 hover:bg-surface-elevated rounded-md transition-colors text-text-muted hover:text-text"
                                        onclick={() =>
                                            navigator.clipboard.writeText(
                                                `${pingUrl}?token=${token}`,
                                            )}
                                        title="Copy to clipboard"
                                    >
                                        <ArrowRightLeft class="size-3.5" />
                                    </button>
                                </div>
                                <p
                                    class="text-[10px] text-text-subtle mt-2 italic"
                                >
                                    Send a GET or POST request to this URL to
                                    check in.
                                </p>
                            </div>
                        {:else if type === "postgres" || type === "mysql" || type === "mongo"}
                            <input
                                id="em-host"
                                required
                                bind:value={connString}
                                placeholder="postgres://user:pass@localhost:5432/db"
                                class="input-base"
                            />
                        {:else}
                            <input
                                id="em-host"
                                required
                                bind:value={host}
                                placeholder={type === "http" || type === "json"
                                    ? "https://example.com"
                                    : type === "dns" || type === "ssl"
                                      ? "example.com"
                                      : type === "websocket"
                                        ? "wss://example.com/ws"
                                        : "192.168.1.1"}
                                class="input-base"
                            />
                        {/if}
                    </div>
                    {/if}

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

                    <!-- TCP / UDP / SMTP / Redis port options -->
                    {#if type === "tcp" || type === "udp" || type === "smtp" || type === "redis"}
                        <div
                            class="pl-4 border-l-2 border-primary/20 py-1 space-y-3"
                        >
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
                                    placeholder={type === "smtp"
                                        ? "587"
                                        : type === "redis"
                                          ? "6379"
                                          : "3306"}
                                    class="input-base"
                                />
                            </div>

                            {#if type === "smtp"}
                                <div class="flex items-center gap-2">
                                    <input
                                        id="em-smtp-tls"
                                        type="checkbox"
                                        bind:checked={requireTls}
                                        class="rounded border-border"
                                    />
                                    <label
                                        for="em-smtp-tls"
                                        class="text-sm text-text-muted"
                                        >Require TLS</label
                                    >
                                </div>
                            {/if}

                            {#if type === "udp"}
                                <div class="grid grid-cols-2 gap-3">
                                    <div class="space-y-1.5">
                                        <label
                                            for="em-udp-send"
                                            class="text-sm font-medium text-text-muted"
                                            >Send Payload</label
                                        >
                                        <input
                                            id="em-udp-send"
                                            bind:value={sendPayload}
                                            placeholder="ping"
                                            class="input-base"
                                        />
                                    </div>
                                    <div class="space-y-1.5">
                                        <label
                                            for="em-udp-expect"
                                            class="text-sm font-medium text-text-muted"
                                            >Expected Response</label
                                        >
                                        <input
                                            id="em-udp-expect"
                                            bind:value={expectedResponse}
                                            placeholder="pong"
                                            class="input-base"
                                        />
                                    </div>
                                </div>
                            {/if}

                            {#if type === "redis"}
                                <div class="grid grid-cols-2 gap-3">
                                    <div class="space-y-1.5">
                                        <label
                                            for="em-redis-pass"
                                            class="text-sm font-medium text-text-muted"
                                            >Password</label
                                        >
                                        <input
                                            id="em-redis-pass"
                                            type="password"
                                            bind:value={dbPassword}
                                            class="input-base"
                                        />
                                    </div>
                                    <div class="space-y-1.5">
                                        <label
                                            for="em-redis-db"
                                            class="text-sm font-medium text-text-muted"
                                            >Database Index</label
                                        >
                                        <input
                                            id="em-redis-db"
                                            type="number"
                                            bind:value={dbIndex}
                                            placeholder="0"
                                            class="input-base"
                                        />
                                    </div>
                                </div>
                            {/if}
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
                                    >Expected Result (IP/String)</label
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

                    <!-- SSH options -->
                    {#if type === "ssh"}
                        <div class="pl-4 border-l-2 border-primary/20 py-1">
                            <div class="space-y-1.5">
                                <label
                                    for="em-ssh-port"
                                    class="text-sm font-medium text-text-muted"
                                    >Port</label
                                >
                                <input
                                    id="em-ssh-port"
                                    type="number"
                                    bind:value={sshPort}
                                    placeholder="22"
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- JSON API options -->
                    {#if type === "json"}
                        <div
                            class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-json-field"
                                    class="text-sm font-medium text-text-muted"
                                    >JSON Field <span class="text-danger"
                                        >*</span
                                    ></label
                                >
                                <input
                                    id="em-json-field"
                                    required
                                    bind:value={jsonField}
                                    placeholder="status or data.health"
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-json-expected"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected Value <span class="text-danger"
                                        >*</span
                                    ></label
                                >
                                <input
                                    id="em-json-expected"
                                    required
                                    bind:value={jsonExpectedValue}
                                    placeholder="ok"
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- HTTPS options -->
                    {#if type === "https"}
                        <div
                            class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-https-method"
                                    class="text-sm font-medium text-text-muted"
                                    >HTTP Method</label
                                >
                                <select
                                    id="em-https-method"
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
                                    for="em-https-status"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected Status</label
                                >
                                <input
                                    id="em-https-status"
                                    type="number"
                                    bind:value={expectedStatus}
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-https-warndays"
                                    class="text-sm font-medium text-text-muted"
                                    >TLS Warn Days</label
                                >
                                <input
                                    id="em-https-warndays"
                                    type="number"
                                    bind:value={httpsWarnDays}
                                    class="input-base"
                                />
                            </div>
                        </div>
                    {/if}

                    <!-- Composite options -->
                    {#if type === "composite"}
                        <div
                            class="space-y-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-comp-ids"
                                    class="text-sm font-medium text-text-muted"
                                    >Monitor IDs (comma-separated) <span
                                        class="text-danger">*</span
                                    ></label
                                >
                                <input
                                    id="em-comp-ids"
                                    required
                                    bind:value={compositeMonitorIDs}
                                    placeholder="id1, id2, id3"
                                    class="input-base font-mono text-xs"
                                />
                            </div>
                            <div class="grid grid-cols-2 gap-3">
                                <div class="space-y-1.5">
                                    <label
                                        for="em-comp-mode"
                                        class="text-sm font-medium text-text-muted"
                                        >Mode</label
                                    >
                                    <select
                                        id="em-comp-mode"
                                        bind:value={compositeMode}
                                        class="input-base bg-background/50"
                                    >
                                        <option value="all_up">All Up</option>
                                        <option value="any_up">Any Up</option>
                                        <option value="quorum">Quorum</option>
                                    </select>
                                </div>
                                {#if compositeMode === "quorum"}
                                    <div class="space-y-1.5">
                                        <label
                                            for="em-comp-quorum"
                                            class="text-sm font-medium text-text-muted"
                                            >Quorum Count</label
                                        >
                                        <input
                                            id="em-comp-quorum"
                                            type="number"
                                            bind:value={compositeQuorum}
                                            placeholder="2"
                                            class="input-base"
                                        />
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {/if}

                    <!-- Transaction options -->
                    {#if type === "transaction"}
                        <div
                            class="space-y-3 pl-4 border-l-2 border-primary/20 py-1"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-txn-steps"
                                    class="text-sm font-medium text-text-muted"
                                    >Steps (JSON array) <span
                                        class="text-danger">*</span
                                    ></label
                                >
                                <textarea
                                    id="em-txn-steps"
                                    required
                                    bind:value={transactionStepsJSON}
                                    rows="5"
                                    class="input-base font-mono text-xs resize-y"
                                ></textarea>
                                <p class="text-[10px] text-text-subtle">
                                    Each step: url, method, headers, body,
                                    expected_status, expected_body, extract
                                </p>
                            </div>
                            <div class="flex items-center gap-2">
                                <input
                                    id="em-txn-tls"
                                    type="checkbox"
                                    bind:checked={transactionSkipTLS}
                                    class="rounded border-border"
                                />
                                <label
                                    for="em-txn-tls"
                                    class="text-sm text-text-muted"
                                    >Skip TLS Verify</label
                                >
                            </div>
                        </div>
                    {/if}

                    <!-- DNS+HTTP options -->
                    {#if type === "dns_http"}
                        <div
                            class="grid grid-cols-1 gap-3 pl-4 border-l-2 border-primary/20 py-1 md:grid-cols-2"
                        >
                            <div class="space-y-1.5">
                                <label
                                    for="em-dh-prefix"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected IP Prefix</label
                                >
                                <input
                                    id="em-dh-prefix"
                                    bind:value={dnsHTTPExpectedIPPrefix}
                                    placeholder="104.18."
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-dh-cname"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected CNAME</label
                                >
                                <input
                                    id="em-dh-cname"
                                    bind:value={dnsHTTPExpectedCNAME}
                                    placeholder="edge.example.net"
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-dh-body"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected Body Contains</label
                                >
                                <input
                                    id="em-dh-body"
                                    bind:value={dnsHTTPExpectedBody}
                                    placeholder="healthy"
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    for="em-dh-status"
                                    class="text-sm font-medium text-text-muted"
                                    >Expected HTTP Status</label
                                >
                                <input
                                    id="em-dh-status"
                                    type="number"
                                    bind:value={dnsHTTPExpectedStatus}
                                    class="input-base"
                                />
                            </div>
                            <div class="flex items-center gap-2 md:col-span-2">
                                <input
                                    id="em-dh-tls"
                                    type="checkbox"
                                    bind:checked={dnsHTTPSkipTLS}
                                    class="rounded border-border"
                                />
                                <label
                                    for="em-dh-tls"
                                    class="text-sm text-text-muted"
                                    >Skip TLS Verify</label
                                >
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
            {:else if open}
                <div class="space-y-4 min-h-[28rem]" aria-live="polite">
                    <div class="space-y-2">
                        <Skeleton height="h-4" width="w-24" />
                        <Skeleton height="h-10" />
                    </div>
                    <div class="space-y-2">
                        <Skeleton height="h-4" width="w-20" />
                        <div class="grid grid-cols-5 gap-2">
                            {#each Array(10) as _}
                                <Skeleton height="h-20" rounded="rounded-xl" />
                            {/each}
                        </div>
                    </div>
                    <div class="space-y-2">
                        <Skeleton height="h-4" width="w-28" />
                        <Skeleton height="h-10" />
                        <Skeleton height="h-24" />
                    </div>
                    <div class="flex justify-end gap-2 pt-2">
                        <Skeleton height="h-10" width="w-24" rounded="rounded-lg" />
                        <Skeleton height="h-10" width="w-32" rounded="rounded-lg" />
                    </div>
                </div>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
