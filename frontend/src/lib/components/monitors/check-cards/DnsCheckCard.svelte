<script lang="ts">
    import { Radar } from "lucide-svelte";
    import {
        parseCheckMetadata,
        parseMonitorConfig,
        readString,
        readStringArray,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import CopyButton from "./_shared/CopyButton.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor, latestCheck }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const metadata = $derived(parseCheckMetadata(latestCheck?.metadata));
    const host = $derived(readString(config, "host"));
    const recordType = $derived(readString(config, "record_type") ?? "A");
    const resolver = $derived(readString(config, "resolver"));
    const expected = $derived(readString(config, "expected"));
    const cadence = $derived(monitor.interval_s);

    const answers = $derived(readStringArray(metadata, "answers"));

    const digSnippet = $derived(
        host
            ? `dig ${recordType} ${host}${resolver ? ` @${resolver}` : ""} +short`
            : "",
    );
</script>

<CheckCardShell
    typeLabel="DNS"
    description="updu resolves the configured record and (optionally) compares the answer."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Record" value={recordType} />
        <FieldTile label="Host" value={host} monospace />
        <FieldTile label="Expected" value={expected} monospace />
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
                        <Radar class="size-4 text-primary" />
                        <p
                            class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                        >
                            DNS Lookup
                        </p>
                    </div>
                    {#if host}
                        <CopyButton
                            value={host}
                            label="Copy host"
                            size="xs"
                        />
                    {/if}
                </div>
                <p class="text-sm text-text">
                    <span class="font-mono text-xs">{recordType}</span>
                    record for
                    <code class="font-mono text-xs break-all">{host ?? "—"}</code>
                    {#if resolver}
                        via
                        <code class="font-mono text-xs">{resolver}</code>
                    {/if}
                </p>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
                {#if digSnippet}
                    <FieldTile
                        label="dig"
                        value={digSnippet}
                        monospace
                        copyable
                        copyLabel="Copy dig command"
                    />
                {/if}
                {#if answers.length > 0}
                    <FieldTile
                        label="Latest Answers"
                        value={answers.join("\n")}
                        monospace
                        multiline
                        copyable
                    />
                {/if}
            </div>
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile
                label="Domain"
                value={host}
                monospace
                copyable={Boolean(host)}
            />
            <FieldTile label="Record Type" value={recordType} />
            <FieldTile
                label="Resolver"
                value={resolver}
                monospace
                copyable={Boolean(resolver)}
            />
            <FieldTile
                label="Expected Answer"
                value={expected}
                monospace
                copyable={Boolean(expected)}
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
