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
    const port = $derived(readNumber(config, "port") ?? 22);
    const cadence = $derived(monitor.interval_s);

    const endpoint = $derived(host ? `${host}:${port}` : undefined);
    const sshSnippet = $derived(host ? `ssh -p ${port} ${host}` : "");
</script>

<CheckCardShell
    typeLabel="SSH"
    description="updu opens an SSH connection and verifies the server responds with a valid banner."
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
        <div class="space-y-3">
            <EndpointHero {endpoint} headline="SSH Endpoint" />
            {#if sshSnippet}
                <FieldTile
                    label="ssh"
                    value={sshSnippet}
                    monospace
                    copyable
                    copyLabel="Copy ssh command"
                />
            {/if}
        </div>
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
        </DetailSection>
    {/snippet}
</CheckCardShell>
