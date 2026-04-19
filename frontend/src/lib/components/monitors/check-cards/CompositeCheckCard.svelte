<script lang="ts">
    import { Layers } from "lucide-svelte";
    import {
        parseCheckMetadata,
        parseMonitorConfig,
        readNumber,
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
    const monitorIDs = $derived(readStringArray(config, "monitor_ids"));
    const mode = $derived(readString(config, "mode") ?? "all_up");
    const quorum = $derived(readNumber(config, "quorum"));
    const cadence = $derived(monitor.interval_s);

    const upCount = $derived(readNumber(metadata, "up_count"));
    const total = $derived(readNumber(metadata, "total"));
    const membersUp = $derived(
        upCount !== undefined && total !== undefined ? `${upCount}/${total}` : undefined,
    );

    const modeLabel = $derived(
        mode === "any_up"
            ? "Any child must be up"
            : mode === "quorum"
              ? `At least ${quorum ?? 1} child${(quorum ?? 1) === 1 ? "" : "ren"} must be up`
              : "All children must be up",
    );

    const tone = $derived.by((): "success" | "warning" | "danger" | "default" => {
        if (upCount === undefined || total === undefined) return "default";
        if (upCount === total) return "success";
        if (upCount === 0) return "danger";
        return "warning";
    });
</script>

<CheckCardShell
    typeLabel="Composite"
    description="updu evaluates the state of multiple child monitors and aggregates them into a single status."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Members Up"
            value={membersUp}
            tone={tone}
        />
        <FieldTile label="Members" value={`${monitorIDs.length}`} />
        <FieldTile label="Mode" value={modeLabel} />
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
                <div class="flex items-center gap-2">
                    <Layers class="size-4 text-primary" />
                    <p
                        class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                    >
                        Aggregation
                    </p>
                </div>
                <p class="text-sm text-text">{modeLabel}</p>
            </div>

            <div class="space-y-2">
                <p
                    class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle"
                >
                    Member Monitors ({monitorIDs.length})
                </p>
                {#if monitorIDs.length === 0}
                    <p class="text-sm text-text-subtle italic">
                        No child monitors configured.
                    </p>
                {:else}
                    <ul
                        data-testid="monitor-composite-members"
                        class="space-y-1.5"
                    >
                        {#each monitorIDs as childId (childId)}
                            <li
                                class="flex items-center justify-between gap-2 rounded-xl border border-border/70 bg-background/60 px-3 py-2"
                            >
                                <a
                                    href={`/monitors/${childId}`}
                                    class="font-mono text-xs text-primary hover:underline break-all"
                                >
                                    {childId}
                                </a>
                                <CopyButton
                                    value={childId}
                                    label={`Copy id ${childId}`}
                                    successMessage="Monitor id copied"
                                    size="xs"
                                />
                            </li>
                        {/each}
                    </ul>
                {/if}
            </div>
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile label="Mode" value={modeLabel} />
            {#if mode === "quorum"}
                <FieldTile label="Quorum" value={quorum} />
            {/if}
            <FieldTile label="Member Count" value={monitorIDs.length} />
            <FieldTile
                label="Member IDs"
                value={monitorIDs.join("\n")}
                monospace
                multiline
                copyable={monitorIDs.length > 0}
                copyLabel="Copy all member ids"
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
