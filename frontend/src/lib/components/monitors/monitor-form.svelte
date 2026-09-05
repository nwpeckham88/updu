<script lang="ts">
    import {
        Globe,
        Network,
        Activity,
        Radar,
        CloudOff,
        Zap,
        Copy,
    } from "lucide-svelte";
    import Modal from "$lib/components/ui/modal.svelte";
    import Field from "$lib/components/ui/field.svelte";
    import Select from "$lib/components/ui/select.svelte";
    import Switch from "$lib/components/ui/switch.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import Button from "$lib/components/ui/button.svelte";
    import TypeSelector, {
        type TypeOption,
        type TypeGroup,
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

    type MonitorType = "http" | "tcp" | "ping" | "dns" | "push";
    let type = $state<MonitorType>("http");

    let host = $state("");
    let intervalS = $state(60);
    let startEnabled = $state(false);

    // HTTP probe state
    let method = $state("GET");
    let expectedStatus = $state(200);
    let expectedBody = $state("");
    let warnDays = $state(14);
    let skipTLSVerify = $state(false);

    // TCP probe state
    let port = $state(80);

    // DNS probe state
    let recordType = $state("A");
    let resolver = $state("");
    let expected = $state("");

    // Push probe state
    let token = $state("");
    let pushGracePeriodS = $state("");

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
            expectedBody,
            warnDays,
            skipTLSVerify,
            port,
            recordType,
            resolver,
            expected,
            token,
            pushGracePeriodS,
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

    const typeGroups: TypeGroup[] = [
        {
            label: "Core Probes",
            options: [
                { value: "http", label: "HTTP", icon: Globe, desc: "Web & TLS expiry" },
                { value: "tcp", label: "TCP", icon: Network, desc: "Port reachability" },
                { value: "ping", label: "Ping", icon: Activity, desc: "ICMP reachability" },
                { value: "dns", label: "DNS", icon: Radar, desc: "Domain resolution" },
                { value: "push", label: "Push", icon: CloudOff, desc: "Heartbeat / Snitch" },
            ],
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
        expectedBody = "";
        warnDays = 14;
        skipTLSVerify = false;
        port = 80;
        recordType = "A";
        resolver = "";
        expected = "";
        token = "";
        pushGracePeriodS = "";
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
        type = (src.type === "https" ? "http" : src.type) || "http";
        intervalS = src.interval_s || 60;

        if (type === "http") {
            host = config.url || "";
            method = config.method || "GET";
            expectedStatus = config.expected_status || 200;
            expectedBody = config.expected_body || "";
            warnDays = config.warn_days ?? 14;
            skipTLSVerify = config.skip_tls_verify || false;
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
        } else if (type === "push") {
            token = config.token || "";
            pushGracePeriodS =
                typeof config.grace_period_s === "number"
                    ? `${config.grace_period_s}`
                    : "";
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
            if (!url.startsWith("http://") && !url.startsWith("https://")) {
                url = "https://" + url;
            }
            config = {
                url,
                method,
                expected_status: expectedStatus,
            };
            if (expectedBody) config.expected_body = expectedBody;
            if (warnDays > 0) config.warn_days = warnDays;
            if (skipTLSVerify) config.skip_tls_verify = true;
        } else if (type === "tcp") {
            config = { host, port };
        } else if (type === "ping") {
            config = { host };
        } else if (type === "dns") {
            config = { host, record_type: recordType };
            if (resolver) config.resolver = resolver;
            if (expected) config.expected = expected;
        } else if (type === "push") {
            config = { token };
            const gracePeriodS = parseOptionalGracePeriod(pushGracePeriodS);
            if (gracePeriodS !== undefined) {
                config.grace_period_s = gracePeriodS;
            }
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
            ? "Create a new core probe for updu to monitor."
            : `Update configuration for ${monitor?.name || "this monitor"}.`;
    });

    const hostLabel = $derived.by(() => {
        if (type === "http") return "URL";
        if (type === "dns") return "Domain Name";
        if (type === "push") return "Check-in Token";
        return "Host / IP";
    });

    const hostPlaceholder = $derived.by(() => {
        if (type === "http") return "https://example.com/health";
        if (type === "dns") return "example.com";
        return "1.1.1.1 or example.com";
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
                    groups={typeGroups}
                    onchange={(v) => (type = v as MonitorType)}
                />
            </div>

            <!-- Host / URL / Token -->
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

            <!-- Push options -->
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
                    class="space-y-3 pl-4 border-l-2 border-primary/20 py-1"
                >
                    <div class="grid grid-cols-2 gap-3">
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

                    <div class="grid grid-cols-2 gap-3">
                        <Field id="{idPrefix}-expected-body" label="Body Contains (optional)">
                            {#snippet children({ id })}
                                <input
                                    {id}
                                    bind:value={expectedBody}
                                    placeholder="e.g. healthy or Example Domain"
                                    class="input-base"
                                />
                            {/snippet}
                        </Field>
                        <Field id="{idPrefix}-warndays" label="TLS Warning Threshold (days)">
                            {#snippet children({ id })}
                                <input
                                    {id}
                                    type="number"
                                    min="1"
                                    bind:value={warnDays}
                                    placeholder="14"
                                    class="input-base"
                                />
                            {/snippet}
                        </Field>
                    </div>

                    <Switch
                        id="{idPrefix}-skip-tls"
                        bind:checked={skipTLSVerify}
                        label="Skip TLS Verification"
                    />
                </div>
            {/if}

            <!-- TCP options -->
            {#if type === "tcp"}
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
                                placeholder="80"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
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
                    <Field id="{idPrefix}-resolver" label="Resolver (optional)">
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={resolver}
                                placeholder="1.1.1.1"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
                    <Field
                        id="{idPrefix}-expected"
                        label="Expected IP / Text (optional)"
                    >
                        {#snippet children({ id })}
                            <input
                                {id}
                                bind:value={expected}
                                placeholder="93.184.216.34"
                                class="input-base"
                            />
                        {/snippet}
                    </Field>
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
                        disabled={Boolean(pushGracePeriodError) || (!host && type !== "push")}
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
                    {#each Array(5) as _, skeletonIndex (skeletonIndex)}
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
