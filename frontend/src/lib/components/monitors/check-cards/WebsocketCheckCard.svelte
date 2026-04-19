<script lang="ts">
    import { ExternalLink, Plug } from "lucide-svelte";
    import {
        parseMonitorConfig,
        readBoolean,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import CopyButton from "./_shared/CopyButton.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const url = $derived(readString(config, "url"));
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const httpUrl = $derived(
        url ? url.replace(/^wss:\/\//i, "https://").replace(/^ws:\/\//i, "http://") : undefined,
    );
    const wscatSnippet = $derived(url ? `wscat -c '${url.replace(/'/g, "'\\''")}'` : "");
</script>

<CheckCardShell
    typeLabel="WebSocket"
    description="updu opens a WebSocket connection and verifies the upgrade succeeds."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Protocol"
            value={url?.startsWith("wss") ? "wss (TLS)" : url?.startsWith("ws") ? "ws (plain)" : undefined}
        />
        <FieldTile
            label="Skip TLS"
            value={skipTLSVerify === undefined
                ? undefined
                : skipTLSVerify
                  ? "Yes"
                  : "No"}
            tone={skipTLSVerify ? "warning" : "default"}
        />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <div
                class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-2"
            >
                <div class="flex items-center justify-between gap-2">
                    <div class="flex items-center gap-2">
                        <Plug class="size-4 text-primary" />
                        <p
                            class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                        >
                            WebSocket URL
                        </p>
                    </div>
                    {#if url}
                        <div class="flex items-center gap-1">
                            {#if httpUrl}
                                <a
                                    href={httpUrl}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class="p-1 hover:bg-surface-elevated rounded-md transition-colors text-text-muted hover:text-text"
                                    title="Open over HTTP(S)"
                                    aria-label="Open over HTTP(S)"
                                >
                                    <ExternalLink class="size-3.5" />
                                </a>
                            {/if}
                            <CopyButton
                                value={url}
                                label="Copy WebSocket URL"
                                successMessage="URL copied"
                                size="xs"
                            />
                        </div>
                    {/if}
                </div>
                {#if url}
                    <code
                        class="block break-all rounded-lg bg-background/70 px-3 py-2 font-mono text-xs text-primary"
                    >
                        {url}
                    </code>
                {:else}
                    <p class="text-sm text-text-muted italic">
                        No URL configured.
                    </p>
                {/if}
            </div>

            {#if wscatSnippet}
                <FieldTile
                    label="wscat"
                    value={wscatSnippet}
                    monospace
                    multiline
                    copyable
                    copyLabel="Copy wscat command"
                />
            {/if}
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile
                label="URL"
                value={url}
                monospace
                copyable={Boolean(url)}
            />
            <FieldTile
                label="Skip TLS Verification"
                value={skipTLSVerify === undefined
                    ? undefined
                    : skipTLSVerify
                      ? "Yes"
                      : "No"}
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
