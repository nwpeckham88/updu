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
    const port = $derived(readNumber(config, "port"));
    const cadence = $derived(monitor.interval_s);

    const endpoint = $derived(host && port ? `${host}:${port}` : host);
</script>

<CheckCardShell
    typeLabel="TCP"
    description="updu opens a TCP connection to the host and port to confirm the service is reachable."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Host" value={host} monospace />
        <FieldTile label="Port" value={port} />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <EndpointHero {endpoint} headline="TCP Endpoint" />
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile
                label="Host"
                value={host}
                monospace
                copyable={Boolean(host)}
            />
            <FieldTile label="Port" value={port} />
            <FieldTile
                label="Endpoint"
                value={endpoint}
                monospace
                copyable={Boolean(endpoint)}
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
