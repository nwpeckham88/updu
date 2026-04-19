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
    const sendPayload = $derived(readString(config, "send_payload"));
    const expectedResponse = $derived(readString(config, "expected_response"));
    const cadence = $derived(monitor.interval_s);

    const endpoint = $derived(host && port ? `${host}:${port}` : host);
</script>

<CheckCardShell
    typeLabel="UDP"
    description="updu sends a UDP packet and (optionally) waits for a matching response."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Host" value={host} monospace />
        <FieldTile label="Port" value={port} />
        <FieldTile
            label="Expects Response"
            value={expectedResponse ? "Yes" : "No"}
        />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <EndpointHero {endpoint} headline="UDP Endpoint" />

            <div class="grid gap-3 sm:grid-cols-2">
                {#if sendPayload}
                    <FieldTile
                        label="Send Payload"
                        value={sendPayload}
                        monospace
                        multiline
                        copyable
                    />
                {/if}
                {#if expectedResponse}
                    <FieldTile
                        label="Expected Response"
                        value={expectedResponse}
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
                label="Host"
                value={host}
                monospace
                copyable={Boolean(host)}
            />
            <FieldTile label="Port" value={port} />
            <FieldTile
                label="Send Payload"
                value={sendPayload}
                monospace
                multiline
                copyable={Boolean(sendPayload)}
            />
            <FieldTile
                label="Expected Response"
                value={expectedResponse}
                monospace
                multiline
                copyable={Boolean(expectedResponse)}
            />
        </DetailSection>
    {/snippet}
</CheckCardShell>
