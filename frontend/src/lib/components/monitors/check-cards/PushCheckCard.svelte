<script lang="ts">
    import { Activity, Webhook } from "lucide-svelte";
    import {
        buildHeartbeatTokenUrl,
        buildPingUrl,
        parseMonitorConfig,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import CopyButton from "./_shared/CopyButton.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const token = $derived(readString(config, "token") ?? "");
    const slugBase = $derived(buildPingUrl(monitor.id));
    const legacyPingUrl = $derived(
        slugBase ? (token ? `${slugBase}?token=${token}` : slugBase) : "",
    );
    const tokenUrl = $derived(buildHeartbeatTokenUrl(token));
    const primaryHeartbeatUrl = $derived(tokenUrl || legacyPingUrl);
    const curlSnippet = $derived(
        tokenUrl
            ? `curl -fsS "${tokenUrl}"`
            : legacyPingUrl
              ? `curl -fsS -X POST "${legacyPingUrl}"`
              : "",
    );
    const markDownUrl = $derived(
        tokenUrl
            ? `${tokenUrl}?status=down`
            : legacyPingUrl
              ? `${legacyPingUrl}${legacyPingUrl.includes("?") ? "&" : "?"}status=down`
              : "",
    );
    const cadence = $derived(monitor.interval_s);
</script>

<CheckCardShell
    typeLabel="Push (Heartbeat)"
    description="Send a request to the URL below from your job, container, or cron — updu marks the monitor down if it doesn't hear from you in time."
    hasDetails
>
    {#snippet hero()}
        <div class="space-y-3">
            <div
                class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-3"
            >
                <div class="flex items-center justify-between gap-2">
                    <div class="flex items-center gap-2">
                        <Webhook class="size-4 text-primary" />
                        <p
                            class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                        >
                            Heartbeat URL
                        </p>
                    </div>
                    {#if primaryHeartbeatUrl}
                        <CopyButton
                            value={primaryHeartbeatUrl}
                            label="Copy heartbeat URL"
                            successMessage="Heartbeat URL copied"
                            testId="monitor-push-copy-url"
                        />
                    {/if}
                </div>
                {#if primaryHeartbeatUrl}
                    <code
                        data-testid="monitor-push-url"
                        class="block break-all rounded-lg bg-background/70 px-3 py-2 font-mono text-xs text-primary"
                    >
                        {primaryHeartbeatUrl}
                    </code>
                    <p class="text-[11px] text-text-muted">
                        Hit this token endpoint from your job, cron, or
                        container. GET, POST, and PUT all work.
                    </p>
                {:else}
                    <p class="text-sm text-text-muted italic">
                        Save the monitor to generate a heartbeat URL.
                    </p>
                {/if}
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
                <FieldTile
                    label="Token"
                    value={token || undefined}
                    monospace
                    copyable={Boolean(token)}
                    copyLabel="Copy token"
                    testId="monitor-push-token"
                />
                <FieldTile
                    label="Cadence"
                    value={cadence ? `Every ${cadence}s` : undefined}
                />
            </div>

            {#if curlSnippet}
                <div class="grid gap-3 sm:grid-cols-2">
                    <FieldTile
                        label="curl"
                        value={curlSnippet}
                        monospace
                        multiline
                        copyable
                        copyLabel="Copy curl snippet"
                        testId="monitor-push-curl"
                    />
                    {#if legacyPingUrl}
                        <FieldTile
                            label="Legacy POST URL"
                            value={legacyPingUrl}
                            monospace
                            copyable
                            copyLabel="Copy legacy POST URL"
                        />
                    {/if}
                </div>
            {/if}
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Heartbeat Routes">
            <FieldTile
                label="Slug Endpoint (POST only)"
                value={legacyPingUrl || undefined}
                monospace
                copyable={Boolean(legacyPingUrl)}
                copyLabel="Copy slug endpoint"
            />
            <FieldTile
                label="Token Endpoint"
                value={tokenUrl || undefined}
                monospace
                copyable={Boolean(tokenUrl)}
                copyLabel="Copy token endpoint"
            />
            <FieldTile
                label="Mark Down"
                value={markDownUrl || undefined}
                monospace
                multiline
                copyable={Boolean(markDownUrl)}
                copyLabel="Copy 'mark down' URL"
            />
            <FieldTile
                label="Slug Route Methods"
                value="POST only"
            />
            <FieldTile
                label="Token Route Methods"
                value="GET, POST, PUT"
            />
        </DetailSection>

        <DetailSection title="Usage Hints">
            <div
                class="sm:col-span-2 space-y-2 rounded-xl border border-border/70 bg-background/60 p-3"
            >
                <div class="flex items-center gap-2 text-text-subtle">
                    <Activity class="size-3.5" />
                    <p
                        class="text-[10px] font-semibold uppercase tracking-[0.16em]"
                    >
                        Cron example
                    </p>
                </div>
                <code
                    class="block whitespace-pre-wrap break-all rounded-lg bg-surface/40 px-3 py-2 font-mono text-xs text-text"
                    >{`*/5 * * * * ${curlSnippet || "curl -fsS <heartbeat-url>"} >/dev/null`}</code
                >
            </div>
        </DetailSection>
    {/snippet}
</CheckCardShell>
