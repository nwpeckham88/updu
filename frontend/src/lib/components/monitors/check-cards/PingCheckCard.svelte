<script lang="ts">
    import {
        parseMonitorConfig,
        readNumber,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import EndpointHero from "./_shared/EndpointHero.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const host = $derived(readString(config, "host"));
    const count = $derived(readNumber(config, "count"));
    const cadence = $derived(monitor.interval_s);
</script>

<CheckCardShell
    typeLabel="Ping"
    description="updu sends ICMP echo requests to confirm the host is reachable."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Host" value={host} monospace />
        <FieldTile label="Pings" value={count} />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <EndpointHero
            endpoint={host}
            headline="Ping Target"
            subline={count ? `Sending ${count} ICMP packet${count === 1 ? "" : "s"} per check.` : undefined}
        />
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile
                label="Host"
                value={host}
                monospace
                copyable={Boolean(host)}
            />
            <FieldTile label="Ping Count" value={count} />
        </DetailSection>
    {/snippet}
</CheckCardShell>
