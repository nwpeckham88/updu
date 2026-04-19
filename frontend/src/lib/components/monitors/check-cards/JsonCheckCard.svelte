<script lang="ts">
    import {
        parseMonitorConfig,
        readBoolean,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import { buildCurlCommand } from "./_shared/curl.ts";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import HttpHero from "./_shared/HttpHero.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const method = $derived(readString(config, "method") ?? "GET");
    const url = $derived(readString(config, "url"));
    const field = $derived(readString(config, "field"));
    const expectedValue = $derived(readString(config, "expected_value"));
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const curlSnippet = $derived(
        buildCurlCommand({ method, url, insecure: skipTLSVerify }),
    );
    const jqSnippet = $derived(
        url && field ? `${curlSnippet} | jq '${field}'` : "",
    );
    const expectation = $derived(
        field && expectedValue
            ? `${field} equals "${expectedValue}"`
            : field
              ? `Check JSON field ${field}`
              : "Successful response",
    );
</script>

<CheckCardShell
    typeLabel="JSON API"
    description="updu sends an HTTP request and asserts a JSON field in the response."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Method" value={method.toUpperCase()} />
        <FieldTile label="Field" value={field} monospace />
        <FieldTile label="Expected" value={expectedValue} monospace />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <HttpHero {method} {url} />

            <div class="grid gap-3 sm:grid-cols-2">
                <FieldTile label="Expectation" value={expectation} />
                {#if jqSnippet}
                    <FieldTile
                        label="Inspect with jq"
                        value={jqSnippet}
                        monospace
                        multiline
                        copyable
                        copyLabel="Copy jq snippet"
                    />
                {:else if curlSnippet}
                    <FieldTile
                        label="curl"
                        value={curlSnippet}
                        monospace
                        multiline
                        copyable
                        copyLabel="Copy curl snippet"
                    />
                {/if}
            </div>
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Request">
            <FieldTile
                label="URL"
                value={url}
                href={url}
                monospace
                copyable={Boolean(url)}
            />
            <FieldTile label="Method" value={method.toUpperCase()} />
            <FieldTile label="JSON Field" value={field} monospace />
            <FieldTile label="Expected Value" value={expectedValue} monospace />
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
