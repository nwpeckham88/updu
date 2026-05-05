<script lang="ts">
    import {
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
        Copy,
    } from "lucide-svelte";
    import Modal from "$lib/components/ui/modal.svelte";
    import Field from "$lib/components/ui/field.svelte";
    import Select from "$lib/components/ui/select.svelte";
    import Switch from "$lib/components/ui/switch.svelte";
    import Textarea from "$lib/components/ui/textarea.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Button from "$lib/components/ui/button.svelte";
    import TypeSelector, {
        type TypeOption,
    } from "$lib/components/monitors/type-selector.svelte";
    import { fetchAPI } from "$lib/api/client";
    import {
        formatDurationSeconds,
        parseMonitorConfig,
        defaultPushGraceSeconds,
    } from "$lib/monitor-config";
    import { monitorsStore } from "$lib/stores/monitors.svelte";
    import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
    import { confirmAction } from "$lib/stores/confirm.svelte";
    import { afterNextPaint, cn } from "$lib/utils";

    type Mode = "create" | "edit";

    interface Props {
        mode: Mode;
        open: boolean;
        monitor?: any;
    }

    let {
        mode,
        open = $bindable(false),
        monitor = $bindable(null),
    }: Props = $props();

    const idPrefix = $derived(mode === "create" ? "cm" : "em");
    const dialogTitle = $derived(
        mode === "create" ? "Add New Monitor" : "Edit Monitor",
    );

    // ---------------- form state ----------------
    let loading = $state(false);
    let testing = $state(false);
    let testResult = $state<any>(null);
    let errorMsg = $state("");
    let groupsWarning = $state("");
    let formReady = $state(false);
    let cancelDeferredOpen: (() => void) | null = null;

    let name = $state("");
    let groups = $state<string[]>(["Core"]);
    let newGroup = $state("");
    let allGroups = $state<string[]>([]);

    type MonitorType =
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
        | "grpc";
    let type = $state<MonitorType>("http");

    let host = $state("");
    let intervalS = $state(60);
    let startEnabled = $state(false);
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
    let pushGracePeriodS = $state("");
    let sendPayload = $state("");
    let expectedResponse = $state("");
    let dbPassword = $state("");
    let dbIndex = $state(0);
    let connString = $state("");
    let requireTls = $state(false);
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

    // gRPC monitor state
    let grpcService = $state("");
    let grpcTLS = $state(false);
    let grpcSkipVerify = $state(false);

    // Snapshot of initial values (edit mode dirty detection)
    let initialSnapshot: string = "";

    function captureSnapshot() {
        return JSON.stringify({
            name,
            groups,
            type,
            host,
            intervalS,
            method,
            expectedStatus,
            port,
            recordType,
            resolver,
            expected,
            sslPort,
            daysBeforeExpiry,
            sshPort,
            jsonField,
            jsonExpectedValue,
            token,
            pushGracePeriodS,
            sendPayload,
            expectedResponse,
            dbPassword,
            dbIndex,
            connString,
            requireTls,
            httpsWarnDays,
            compositeMonitorIDs,
            compositeMode,
            compositeQuorum,
            transactionStepsJSON,
            transactionSkipTLS,
            dnsHTTPExpectedIPPrefix,
            dnsHTTPExpectedCNAME,
            dnsHTTPExpectedBody,
            dnsHTTPSkipTLS,
            dnsHTTPExpectedStatus,
            grpcService,
            grpcTLS,
            grpcSkipVerify,
        });
    }

    const isDirty = $derived(
        formReady && mode === "edit" && captureSnapshot() !== initialSnapshot,
    );

    // Keep this in sync with internal/models.MaxPushGraceSeconds.
    const MAX_PUSH_GRACE_PERIOD_S = 7 * 24 * 60 * 60;

    function parseOptionalGracePeriod(value: string): number | undefined {
        const trimmed = value.trim();
        if (trimmed.length === 0) {
            return undefined;
        }

        const parsed = Number(trimmed);
        if (!Number.isFinite(parsed) || !Number.isInteger(parsed) || parsed < 0) {
            return undefined;
        }

        return parsed;
    }

    const pushGracePeriodError = $derived.by(() => {
        const trimmed = pushGracePeriodS.trim();
        if (trimmed.length === 0) {
            return "";
        }

        const parsed = Number(trimmed);
        if (!Number.isFinite(parsed) || !Number.isInteger(parsed) || parsed < 0) {
            return "Enter a whole number of seconds.";
        }
        if (parsed > MAX_PUSH_GRACE_PERIOD_S) {
            return "Maximum tolerance is 7 days (604800 seconds).";
        }

        return "";
    });

    const pushCheckInUrl = $derived(
        token && typeof window !== "undefined"
            ? `${window.location.origin}/heartbeat/${token}`
            : "",
    );

    const defaultPushGracePeriodS = $derived(
        intervalS > 0 ? defaultPushGraceSeconds(intervalS) : 0,
    );

    const configuredPushGracePeriodS = $derived(
        parseOptionalGracePeriod(pushGracePeriodS),
    );

    const effectivePushGracePeriodS = $derived(
        configuredPushGracePeriodS ?? defaultPushGracePeriodS,
    );

    const pushGracePeriodLabel = $derived(
        formatDurationSeconds(effectivePushGracePeriodS) ??
            `${effectivePushGracePeriodS}s`,
    );

    const pushDownAfterLabel = $derived(
        formatDurationSeconds(intervalS + effectivePushGracePeriodS) ??
            `${intervalS + effectivePushGracePeriodS}s`,
    );

    const typeOptions: TypeOption[] = [
        { value: "http", label: "HTTP", icon: Globe, desc: "Web endpoints" },
        { value: "tcp", label: "TCP", icon: Network, desc: "Port checks" },
        { value: "ping", label: "Ping", icon: Activity, desc: "ICMP ping" },
        { value: "dns", label: "DNS", icon: Radar, desc: "DNS records" },
        { value: "ssl", label: "SSL", icon: ShieldCheck, desc: "Cert expiry" },
        { value: "ssh", label: "SSH", icon: Terminal, desc: "SSH banner" },
        { value: "json", label: "JSON", icon: Braces, desc: "API fields" },
        { value: "push", label: "Push", icon: CloudOff, desc: "Inbound check-ins" },
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
        {
            value: "grpc",
            label: "gRPC",
            icon: Zap,
            desc: "gRPC health",
        },
    ];

    const httpMethodOptions = [
        { value: "GET", label: "GET" },
        { value: "POST", label: "POST" },
        { value: "PUT", label: "PUT" },
        { value: "HEAD", label: "HEAD" },
    ];

    const dnsRecordOptions = [
        { value: "A", label: "A" },
        { value: "AAAA", label: "AAAA" },
        { value: "CNAME", label: "CNAME" },
        { value: "MX", label: "MX" },
        { value: "TXT", label: "TXT" },
        { value: "NS", label: "NS" },
    ];

    const compositeModeOptions = [
        { value: "all_up", label: "All Up" },
        { value: "any_up", label: "Any Up" },
        { value: "quorum", label: "Quorum" },
    ];

    // ---------------- helpers ----------------
    function generateToken() {
        const bytes = new Uint8Array(16);
        crypto.getRandomValues(bytes);
        token = Array.from(bytes, (byte) =>
            byte.toString(16).padStart(2, "0"),
        ).join("");
    }

    function resetForm() {
        name = "";
        groups = ["Core"];
        newGroup = "";
        type = "http";
        host = "";
        intervalS = 60;
        startEnabled = false;
        method = "GET";
        expectedStatus = 200;
        port = 80;
        recordType = "A";
        resolver = "";
        expected = "";
        sslPort = 443;
        daysBeforeExpiry = 7;
        sshPort = 22;
        jsonField = "";
        jsonExpectedValue = "";
        token = "";
        pushGracePeriodS = "";
        sendPayload = "";
        expectedResponse = "";
        dbPassword = "";
        dbIndex = 0;
        connString = "";
        requireTls = false;
        httpsWarnDays = 14;
        compositeMonitorIDs = "";
        compositeMode = "all_up";
        compositeQuorum = 1;
        transactionStepsJSON =
            '[\n  {"url": "https://example.com", "method": "GET"}\n]';
        transactionSkipTLS = false;
        dnsHTTPExpectedIPPrefix = "";
        dnsHTTPExpectedCNAME = "";
        dnsHTTPExpectedBody = "";
        dnsHTTPSkipTLS = false;
        dnsHTTPExpectedStatus = 200;
        grpcService = "";
        grpcTLS = false;
        grpcSkipVerify = false;
        errorMsg = "";
        testResult = null;
        groupsWarning = "";
    }

    function populateFromMonitor(src: any) {
        const config = parseMonitorConfig(src.config);
        name = src.name;
        if (src.groups && src.groups.length > 0) {
            groups = [...src.groups];
        } else {
            const legacy = src.group_name ?? src.group;
            groups = legacy ? [legacy] : ["Core"];
        }
        type = src.type;
        intervalS = src.interval_s || 60;

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
            pushGracePeriodS =
                typeof config.grace_period_s === "number"
                    ? `${config.grace_period_s}`
                    : "";
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
        } else if (type === "grpc") {
            host = config.host || "";
            port = config.port || 50051;
            grpcService = config.service || "";
            grpcTLS = config.tls || false;
            grpcSkipVerify = config.insecure_skip_verify || false;
        }
        errorMsg = "";
        testResult = null;
        groupsWarning = "";
    }

    function clearDeferredOpen() {
        cancelDeferredOpen?.();
        cancelDeferredOpen = null;
    }

    // Auto-generate push token on type change in create mode
    $effect(() => {
        if (mode === "create" && type === "push" && !token) {
            generateToken();
        }
    });

    // Open lifecycle
    $effect(() => {
        clearDeferredOpen();

        if (!open || (mode === "edit" && !monitor)) {
            formReady = false;
            return;
        }

        const pendingMonitor = monitor;
        const pendingMonitorID = monitor?.id;
        formReady = false;
        errorMsg = "";
        groupsWarning = "";

        cancelDeferredOpen = afterNextPaint(() => {
            if (mode === "edit") {
                if (!open || monitor?.id !== pendingMonitorID) return;
                populateFromMonitor(pendingMonitor);
            } else {
                resetForm();
            }
            initialSnapshot = captureSnapshot();
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

    function buildConfig(): Record<string, any> {
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
            const gracePeriodS = parseOptionalGracePeriod(pushGracePeriodS);
            if (gracePeriodS !== undefined) {
                config.grace_period_s = gracePeriodS;
            }
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
                    .map((s) => s.trim())
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
        } else if (type === "grpc") {
            config = {
                host,
                port,
                tls: grpcTLS,
            };
            if (grpcService) config.service = grpcService;
            if (grpcTLS && grpcSkipVerify) config.insecure_skip_verify = true;
        }
        return config;
    }

    async function handleTest() {
        testing = true;
        testResult = null;
        errorMsg = "";
        try {
            const res = await fetchAPI("/api/v1/monitors/test", {
                method: "POST",
                body: JSON.stringify({
                    type,
                    config: buildConfig(),
                    timeout_s: 10,
                }),
            });
            testResult = res;
        } catch (err) {
            errorMsg = toastFromError(err, "Test failed");
        } finally {
            testing = false;
        }
    }

    async function handleSubmit(e: Event) {
        e.preventDefault();
        loading = true;
        errorMsg = "";
        try {
            if (mode === "create") {
                await fetchAPI("/api/v1/monitors", {
                    method: "POST",
                    body: JSON.stringify({
                        name,
                        type,
                        groups,
                        interval_s: intervalS,
                        enabled: startEnabled,
                        config: buildConfig(),
                    }),
                });
                toastStore.success(`Monitor "${name}" created`);
            } else {
                if (!monitor) return;
                await fetchAPI(`/api/v1/monitors/${monitor.id}`, {
                    method: "PUT",
                    body: JSON.stringify({
                        name,
                        type,
                        groups,
                        interval_s: intervalS,
                        enabled: monitor.enabled,
                        config: buildConfig(),
                    }),
                });
                toastStore.success(`Monitor "${name}" updated`);
            }
            open = false;
            void monitorsStore.init();
        } catch (err) {
            errorMsg = toastFromError(
                err,
                mode === "create"
                    ? "Failed to create monitor"
                    : "Failed to update monitor",
            );
        } finally {
            loading = false;
        }
    }

    async function handleCancel() {
        if (mode === "edit" && isDirty) {
            const ok = await confirmAction({
                title: "Discard changes?",
                description:
                    "You have unsaved changes to this monitor. Closing will discard them.",
                confirmLabel: "Discard",
                variant: "destructive",
            });
            if (!ok) return;
        }
        open = false;
    }

    function copyCheckInUrl() {
        try {
            if (!pushCheckInUrl) return;
            navigator.clipboard.writeText(pushCheckInUrl);
            toastStore.success("Check-in URL copied");
        } catch {
            // ignore
        }
    }

    const description = $derived.by(() => {
        if (type === "push") {
            return mode === "create"
                ? "Create a passive check-in monitor for cron jobs, workers, and backups."
                : `Update how ${monitor?.name || "this monitor"} receives and evaluates inbound check-ins.`;
        }

        return mode === "create"
            ? "Create a new endpoint check for updu to monitor."
            : `Update configuration for ${monitor?.name || "this monitor"}.`;
    });

    const hostLabel = $derived.by(() => {
        if (type === "http" || type === "json" || type === "https" || type === "dns_http")
            return "URL";
        if (type === "dns") return "Domain Name";
        if (type === "ssl") return "Hostname";
        if (type === "push") return "Check-in Token";
        if (type === "websocket") return "WebSocket URL";
        if (type === "postgres" || type === "mysql" || type === "mongo")
            return "Connection String";
        return "Host / IP";
    });

    const hostPlaceholder = $derived.by(() => {
        if (type === "http" || type === "json")
            return "https://example.com/api/health";
        if (type === "dns" || type === "ssl") return "example.com";
        if (type === "websocket") return "wss://example.com/ws";
        return "192.168.1.1";
    });
</script>

<Modal
    bind:open
    title={dialogTitle}
    {description}
    size="lg"
    contentClass="overflow-visible"
>
    {#if formReady}
        <div class="sr-only" aria-live="polite">
            {mode === "create"
                ? "Create monitor form ready."
                : "Edit monitor form ready."}
        </div>
        <form onsubmit={handleSubmit} class="space-y-4">
            {#if errorMsg}
                <div
                    class="p-3 text-sm text-danger bg-danger/10 border border-danger/20 rounded-lg"
                    role="alert"
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

            <!-- Name + Groups -->
            <div class="grid grid-cols-2 gap-3">
                <Field id="{idPrefix}-name" label="Name" required>
                    {#snippet children({ id })}
                        <input
                            {id}
                            required
                            bind:value={name}
                            placeholder="e.g. Nextcloud UI"
                            class="input-base"
                        />
                    {/snippet}
                </Field>
                <div class="space-y-1.5 col-span-2">
                    <span class="text-sm font-medium text-text-muted">Groups</span>
                    <div class="space-y-2">
                        <div
                            class="flex flex-wrap gap-1.5 min-h-[36px] p-1.5 bg-surface-elevated/50 border border-border rounded-lg"
                        >
                            {#each groups as group (group)}
                                <span
                                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-primary/10 text-primary border border-primary/20"
                                >
                                    {group}
                                    <button
                                        type="button"
                                        aria-label={`Remove group ${group}`}
                                        onclick={() => removeGroup(group)}
                                        class="hover:text-primary-light transition-colors"
                                    >
                                        <span aria-hidden="true">×</span>
                                    </button>
                                </span>
                            {/each}
                            <input
                                bind:value={newGroup}
                                placeholder={groups.length === 0
                                    ? "Add groups..."
                                    : ""}
                                aria-label="Add group"
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
                                {#each allGroups.filter((g) => !groups.includes(g)) as g (g)}
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
                <p class="text-sm font-medium text-text-muted">Monitor Type</p>
                <TypeSelector
                    value={type}
                    options={typeOptions}
                    onchange={(v) => (type = v as MonitorType)}
                />
            </div>

            <!-- Host / URL / Token / ConnString -->
            {#if type !== "composite" && type !== "transaction"}
                <Field id="{idPrefix}-host" label={hostLabel} required>
                    {#snippet children({ id })}
                        {#if type === "push"}
                            <div class="flex gap-2">
                                <input
                                    {id}
                                    required
                                    bind:value={token}
                                    placeholder="Secret check-in token"
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
                            {#if pushCheckInUrl}
                                <div
                                    class="mt-3 rounded-lg border border-border bg-surface-elevated/50 p-3"
                                >
                                    <p
                                        class="mb-1.5 text-[11px] font-medium uppercase tracking-wider text-text-muted"
                                    >
                                        {mode === "create"
                                            ? "Check-in URL Preview"
                                            : "Check-in URL"}
                                    </p>
                                    <div class="flex items-center gap-2">
                                        <code
                                            class="flex-1 break-all rounded border border-primary/10 bg-primary/5 px-2 py-1 text-[10px] text-primary"
                                        >
                                            {pushCheckInUrl}
                                        </code>
                                        <button
                                            type="button"
                                            class="rounded-md p-1.5 text-text-muted transition-colors hover:bg-surface-elevated hover:text-text"
                                            onclick={copyCheckInUrl}
                                            title="Copy to clipboard"
                                            aria-label="Copy check-in URL"
                                        >
                                            <Copy class="size-3.5" />
                                        </button>
                                    </div>
                                    <p class="mt-2 text-[10px] italic text-text-subtle">
                                        {mode === "create"
                                            ? "This URL becomes active after you save the monitor."
                                            : "Recommended endpoint for jobs, containers, and cron tasks. GET, POST, and PUT all work."}
                                    </p>
                                </div>
                            {/if}
                        {:else if type === "postgres" || type === "mysql" || type === "mongo"}
                            <input
                                {id}
                                required
                                bind:value={connString}
                                placeholder="postgres://user:pass@localhost:5432/db"
                                class="input-base"
                            />
                        {:else}
                            <input
                                {id}
                                required
                                bind:value={host}
                                placeholder={hostPlaceholder}
                                class="input-base"
                            />
                        {/if}
                    {/snippet}
                </Field>
            {/if}

            {#if type === "push"}
                <div class="space-y-3 pl-4 border-l-2 border-primary/20 py-1">
                    <Field
                        id="{idPrefix}-push-grace"
                        label="Late Check-in Tolerance"
                        hint={pushGracePeriodError
                            ? undefined
                            : `Extra time after the expected cadence before updu marks this monitor down. Leave blank to use the default ${formatDurationSeconds(defaultPushGracePeriodS) ?? `${defaultPushGracePeriodS}s`} buffer.`}
                        error={pushGracePeriodError || undefined}
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                min="0"
                                max={MAX_PUSH_GRACE_PERIOD_S}
                                step="1"
                                inputmode="numeric"
                                value={pushGracePeriodS}
                                oninput={(event) => {
                                    pushGracePeriodS = (
                                        event.currentTarget as HTMLInputElement
                                    ).value;
                                }}
                                placeholder={`${defaultPushGracePeriodS}`}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>

                    <div class="rounded-lg border border-border bg-surface-elevated/40 p-3">
                        <p
                            class="text-[11px] font-medium uppercase tracking-wider text-text-muted"
                        >
                            Passive Behavior
                        </p>
                        <p class="mt-1 text-xs text-text-muted">
                            updu waits for a check-in every
                            {formatDurationSeconds(intervalS) ?? `${intervalS}s`}
                            and currently gives it {pushGracePeriodLabel} of
                            extra time. This monitor goes down after
                            {pushDownAfterLabel} without a request.
                        </p>
                    </div>
                </div>
            {/if}

            <!-- HTTP options -->
            {#if type === "http"}
                <div
                    class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field id="{idPrefix}-method" label="HTTP Method">
                        {#snippet children({ id })}
                            <Select
                                {id}
                                bind:value={method}
                                options={httpMethodOptions}
                            />
                        {/snippet}
                    </Field>
                    <Field id="{idPrefix}-status" label="Expected Status">
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={expectedStatus}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- TCP / UDP / SMTP / Redis / gRPC port options -->
            {#if type === "tcp" || type === "udp" || type === "smtp" || type === "redis" || type === "grpc"}
                <div
                    class="pl-4 border-l-2 border-primary/20 py-1 space-y-3"
                >
                    <Field id="{idPrefix}-port" label="Port" required>
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                required
                                bind:value={port}
                                placeholder={type === "smtp"
                                    ? "587"
                                    : type === "redis"
                                      ? "6379"
                                      : type === "grpc"
                                        ? "50051"
                                        : "3306"}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>

                    {#if type === "smtp"}
                        <Switch
                            id="{idPrefix}-smtp-tls"
                            bind:checked={requireTls}
                            label="Require TLS"
                        />
                    {/if}

                    {#if type === "udp"}
                        <div class="grid grid-cols-2 gap-3">
                            <Field id="{idPrefix}-udp-send" label="Send Payload">
                                {#snippet children({ id })}
                                    <input
                                        {id}
                                        bind:value={sendPayload}
                                        placeholder="ping"
                                        class="input-base"
                                    />
                                {/snippet}
                            </Field>
                            <Field
                                id="{idPrefix}-udp-expect"
                                label="Expected Response"
                            >
                                {#snippet children({ id })}
                                    <input
                                        {id}
                                        bind:value={expectedResponse}
                                        placeholder="pong"
                                        class="input-base"
                                    />
                                {/snippet}
                            </Field>
                        </div>
                    {/if}

                    {#if type === "redis"}
                        <div class="grid grid-cols-2 gap-3">
                            <Field id="{idPrefix}-redis-pass" label="Password">
                                {#snippet children({ id })}
                                    <input
                                        {id}
                                        type="password"
                                        bind:value={dbPassword}
                                        class="input-base"
                                    />
                                {/snippet}
                            </Field>
                            <Field
                                id="{idPrefix}-redis-db"
                                label="Database Index"
                            >
                                {#snippet children({ id })}
                                    <input
                                        {id}
                                        type="number"
                                        bind:value={dbIndex}
                                        placeholder="0"
                                        class="input-base"
                                    />
                                {/snippet}
                            </Field>
                        </div>
                    {/if}

                    {#if type === "grpc"}
                        <Field
                            id="{idPrefix}-grpc-service"
                            label="Service (optional)"
                        >
                            {#snippet children({ id })}
                                <input
                                    {id}
                                    bind:value={grpcService}
                                    placeholder="payments.v1.PaymentService"
                                    class="input-base"
                                />
                            {/snippet}
                        </Field>
                        <Switch
                            id="{idPrefix}-grpc-tls"
                            bind:checked={grpcTLS}
                            label="Use TLS"
                        />
                        {#if grpcTLS}
                            <Switch
                                id="{idPrefix}-grpc-skip"
                                bind:checked={grpcSkipVerify}
                                label="Skip TLS verify (insecure)"
                            />
                        {/if}
                    {/if}
                </div>
            {/if}

            <!-- DNS options -->
            {#if type === "dns"}
                <div
                    class="grid grid-cols-3 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field id="{idPrefix}-record" label="Record Type">
                        {#snippet children({ id })}
                            <Select
                                {id}
                                bind:value={recordType}
                                options={dnsRecordOptions}
                            />
                        {/snippet}
                    </Field>
                    <Field id="{idPrefix}-resolver" label="Resolver">
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={resolver}
                                placeholder="8.8.8.8"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-expected"
                        label="Expected Result (IP/String)"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={expected}
                                placeholder="1.2.3.4"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- SSL options -->
            {#if type === "ssl"}
                <div
                    class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field id="{idPrefix}-ssl-port" label="Port">
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={sslPort}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-ssl-days"
                        label="Warn before (days)"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={daysBeforeExpiry}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- SSH options -->
            {#if type === "ssh"}
                <div class="pl-4 border-l-2 border-primary/20 py-1">
                    <Field id="{idPrefix}-ssh-port" label="Port">
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={sshPort}
                                placeholder="22"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- JSON API options -->
            {#if type === "json"}
                <div
                    class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field
                        id="{idPrefix}-json-field"
                        label="JSON Field"
                        required
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                required
                                bind:value={jsonField}
                                placeholder="status or data.health"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-json-expected"
                        label="Expected Value"
                        required
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                required
                                bind:value={jsonExpectedValue}
                                placeholder="ok"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- HTTPS options -->
            {#if type === "https"}
                <div
                    class="grid grid-cols-2 gap-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field id="{idPrefix}-https-method" label="HTTP Method">
                        {#snippet children({ id })}
                            <Select
                                {id}
                                bind:value={method}
                                options={httpMethodOptions}
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-https-status"
                        label="Expected Status"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={expectedStatus}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-https-warndays"
                        label="TLS Warn Days"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={httpsWarnDays}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                </div>
            {/if}

            <!-- Composite options -->
            {#if type === "composite"}
                <div
                    class="space-y-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field
                        id="{idPrefix}-comp-ids"
                        label="Monitor IDs (comma-separated)"
                        required
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                required
                                bind:value={compositeMonitorIDs}
                                placeholder="id1, id2, id3"
                                class="input-base font-mono text-xs"
                            />
                        {/snippet}
                    </Field>
                    <div class="grid grid-cols-2 gap-3">
                        <Field id="{idPrefix}-comp-mode" label="Mode">
                            {#snippet children({ id })}
                                <Select
                                    {id}
                                    bind:value={compositeMode}
                                    options={compositeModeOptions}
                                />
                            {/snippet}
                        </Field>
                        {#if compositeMode === "quorum"}
                            <Field
                                id="{idPrefix}-comp-quorum"
                                label="Quorum Count"
                            >
                                {#snippet children({ id })}
                                    <input
                                        {id}
                                        type="number"
                                        bind:value={compositeQuorum}
                                        placeholder="2"
                                        class="input-base"
                                    />
                                {/snippet}
                            </Field>
                        {/if}
                    </div>
                </div>
            {/if}

            <!-- Transaction options -->
            {#if type === "transaction"}
                <div
                    class="space-y-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <Field
                        id="{idPrefix}-txn-steps"
                        label="Steps (JSON array)"
                        required
                        hint="Each step: url, method, headers, body, expected_status, expected_body, extract"
                    >
                        {#snippet children({ id })}
                            <Textarea
                                {id}
                                required
                                bind:value={transactionStepsJSON}
                                rows={5}
                                class="font-mono text-xs"
                            />
                        {/snippet}
                    </Field>
                    <Switch
                        id="{idPrefix}-txn-tls"
                        bind:checked={transactionSkipTLS}
                        label="Skip TLS Verify"
                    />
                </div>
            {/if}

            <!-- DNS+HTTP options -->
            {#if type === "dns_http"}
                <div
                    class="grid grid-cols-1 gap-3 pl-4 border-l-2 border-primary/20 py-1 md:grid-cols-2"
                >
                    <Field
                        id="{idPrefix}-dh-prefix"
                        label="Expected IP Prefix"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={dnsHTTPExpectedIPPrefix}
                                placeholder="104.18."
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field id="{idPrefix}-dh-cname" label="Expected CNAME">
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={dnsHTTPExpectedCNAME}
                                placeholder="edge.example.net"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-dh-body"
                        label="Expected Body Contains"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={dnsHTTPExpectedBody}
                                placeholder="healthy"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-dh-status"
                        label="Expected HTTP Status"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                type="number"
                                bind:value={dnsHTTPExpectedStatus}
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <div class="md:col-span-2">
                        <Switch
                            id="{idPrefix}-dh-tls"
                            bind:checked={dnsHTTPSkipTLS}
                            label="Skip TLS Verify"
                        />
                    </div>
                </div>
            {/if}

            <!-- Interval -->
            <div class="space-y-1.5">
                <div class="flex items-center justify-between">
                    <label
                        for="{idPrefix}-interval"
                        class="text-sm font-medium text-text-muted"
                        >Check Interval</label
                    >
                    <span
                        class="text-xs font-mono bg-surface-elevated px-2 py-0.5 rounded-md border border-border text-text"
                        >{intervalS}s</span
                    >
                </div>
                <input
                    id="{idPrefix}-interval"
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

            {#if mode === "create"}
                <div class="rounded-lg border border-border bg-surface-elevated/40 p-3">
                    <Switch
                        id="{idPrefix}-start-enabled"
                        bind:checked={startEnabled}
                        label="Enable checks after creation"
                        description="Leave paused while reviewing recipients, thresholds, and initial history."
                    />
                </div>
            {/if}

            <!-- Test result (create only) -->
            {#if mode === "create" && testResult}
                <div
                    class={cn(
                        "p-3 rounded-lg border text-sm",
                        testResult.status === "up"
                            ? "bg-success/10 border-success/20 text-success"
                            : "bg-danger/10 border-danger/20 text-danger",
                    )}
                    role="status"
                >
                    <div class="flex items-center justify-between">
                        <span class="font-semibold uppercase text-xs"
                            >{testResult.status}</span
                        >
                        {#if testResult.latency_ms != null}
                            <span class="text-xs opacity-70"
                                >{testResult.latency_ms}ms</span
                            >
                        {/if}
                    </div>
                    {#if testResult.message}
                        <p class="text-xs mt-1 opacity-80 break-all">
                            {testResult.message}
                        </p>
                    {/if}
                </div>
            {/if}

            <!-- Actions -->
            <div class="flex gap-2 justify-end pt-2">
                <Button type="button" variant="outline" onclick={handleCancel}
                    >Cancel</Button
                >
                {#if mode === "create"}
                    <Button
                        type="button"
                        variant="outline"
                        loading={testing}
                        disabled={Boolean(pushGracePeriodError) ||
                            (type === "composite"
                                ? !compositeMonitorIDs
                                : type === "transaction"
                                  ? false
                                  : !host && type !== "push")}
                        onclick={handleTest}
                    >
                        <Zap class="size-3.5" />
                        {testing ? "Testing..." : "Test"}
                    </Button>
                    <Button type="submit" {loading} disabled={Boolean(pushGracePeriodError) || loading}>
                        {loading ? "Creating..." : "Create Monitor"}
                    </Button>
                {:else}
                    <Button
                        type="submit"
                        {loading}
                        disabled={Boolean(pushGracePeriodError) || (!isDirty && !loading)}
                    >
                        {loading ? "Saving..." : "Update Monitor"}
                    </Button>
                {/if}
            </div>
        </form>
    {:else}
        <div class="space-y-4 min-h-[28rem]" aria-live="polite">
            <div class="space-y-2">
                <Skeleton height="h-4" width="w-24" />
                <Skeleton height="h-10" />
            </div>
            <div class="space-y-2">
                <Skeleton height="h-4" width="w-20" />
                <div class="grid grid-cols-5 gap-2">
                    {#each Array(10) as _, skeletonIndex (skeletonIndex)}
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
</Modal>
