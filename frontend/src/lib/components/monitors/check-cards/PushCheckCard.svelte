<script lang="ts">
    import { Activity, Webhook } from "lucide-svelte";
    import {
        buildHeartbeatTokenUrl,
        buildPingUrl,
        formatDurationSeconds,
        parseMonitorConfig,
        readString,
        resolvePushGracePeriodSeconds,
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
    const gracePeriodS = $derived(resolvePushGracePeriodSeconds(config, cadence) ?? 0);
    const cadenceLabel = $derived(
        typeof cadence === "number"
            ? `Every ${formatDurationSeconds(cadence) ?? `${cadence}s`}`
            : undefined,
    );
    const gracePeriodLabel = $derived(
        formatDurationSeconds(gracePeriodS) ?? `${gracePeriodS}s`,
    );
    const downAfterLabel = $derived(
        typeof cadence === "number"
            ? `No check-in for ${formatDurationSeconds(cadence + gracePeriodS) ?? `${cadence + gracePeriodS}s`}`
            : undefined,
    );
</script>

<CheckCardShell
    typeLabel="Push Check-In"
    description="Passive monitor. Your job, backup, or cron task checks in here; updu marks it down when the expected window expires without an inbound request."
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
                            Check-In URL
                        </p>
                    </div>
                    {#if primaryHeartbeatUrl}
                        <CopyButton
                            value={primaryHeartbeatUrl}
                            label="Copy check-in URL"
                            successMessage="Check-in URL copied"
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
                        Recommended endpoint for jobs, cron tasks, workers,
                        and backups. GET, POST, and PUT all work.
                    </p>
                {:else}
                    <p class="text-sm text-text-muted italic">
                        Save the monitor to activate this check-in URL.
                    </p>
                {/if}
            </div>

            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <FieldTile
                    label="Token"
                    value={token || undefined}
                    monospace
                    copyable={Boolean(token)}
                    copyLabel="Copy token"
                    testId="monitor-push-token"
                />
                <FieldTile
                    label="Expected Cadence"
                    value={cadenceLabel}
                />
                <FieldTile
                    label="Late Tolerance"
                    value={gracePeriodLabel}
                />
                <FieldTile
                    label="Down After"
                    value={downAfterLabel}
                />
            </div>

            {#if curlSnippet}
                <div class="grid gap-3 sm:grid-cols-2">
                    <FieldTile
                        label="curl check-in"
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
        <DetailSection title="Check-In Endpoints">
            <FieldTile
                label="Recommended Endpoint"
                value={tokenUrl || undefined}
                monospace
                copyable={Boolean(tokenUrl)}
                copyLabel="Copy recommended endpoint"
            />
            <FieldTile
                label="Legacy Slug Endpoint (POST only)"
                value={legacyPingUrl || undefined}
                monospace
                copyable={Boolean(legacyPingUrl)}
                copyLabel="Copy legacy endpoint"
            />
            <FieldTile
                label="Failure URL"
                value={markDownUrl || undefined}
                monospace
                multiline
                copyable={Boolean(markDownUrl)}
                copyLabel="Copy failure URL"
            />
            <FieldTile
                label="Accepted Methods"
                value="GET, POST, PUT"
            />
            <FieldTile
                label="Late Check-in Tolerance"
                value={gracePeriodLabel}
            />
            <FieldTile
                label="Down After"
                value={downAfterLabel}
            />
        </DetailSection>

        <DetailSection title="How It Works">
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
                <p class="text-xs text-text-muted">
                    This is a passive monitor. updu waits for inbound
                    check-ins instead of polling a target directly.
                </p>
                <code
                    class="block whitespace-pre-wrap break-all rounded-lg bg-surface/40 px-3 py-2 font-mono text-xs text-text"
                    >{`*/5 * * * * ${curlSnippet || "curl -fsS <check-in-url>"} >/dev/null`}</code
                >
            </div>
        </DetailSection>
    {/snippet}
</CheckCardShell>
