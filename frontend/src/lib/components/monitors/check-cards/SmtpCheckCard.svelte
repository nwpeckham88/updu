<script lang="ts">
    import {
        parseMonitorConfig,
        readBoolean,
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
    const requireTLS = $derived(readBoolean(config, "require_tls"));
    const cadence = $derived(monitor.interval_s);

    const endpoint = $derived(host && port ? `${host}:${port}` : host);
</script>

<CheckCardShell
    typeLabel="SMTP"
    description="updu opens an SMTP connection and verifies the server greets with a 220 banner."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Host" value={host} monospace />
        <FieldTile label="Port" value={port} />
        <FieldTile
            label="TLS"
            value={requireTLS === undefined
                ? undefined
                : requireTLS
                  ? "Required"
                  : "Optional"}
            tone={requireTLS ? "success" : "default"}
        />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <EndpointHero
            {endpoint}
            headline="SMTP Endpoint"
            subline={requireTLS ? "STARTTLS required." : undefined}
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
            <FieldTile label="Port" value={port} />
            <FieldTile
                label="Require TLS"
                value={requireTLS === undefined
                    ? undefined
                    : requireTLS
                      ? "Yes"
                      : "No"}
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
