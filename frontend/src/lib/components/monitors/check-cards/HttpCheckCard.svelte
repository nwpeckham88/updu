<script lang="ts">
    import {
        parseMonitorConfig,
        readBoolean,
        readNumber,
        readString,
        readStringRecord,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import { buildCurlCommand } from "./_shared/curl.ts";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import HttpHero from "./_shared/HttpHero.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor, latestCheck }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const method = $derived(readString(config, "method") ?? "GET");
    const url = $derived(readString(config, "url"));
    const expectedStatus = $derived(readNumber(config, "expected_status") ?? 200);
    const expectedBody = $derived(readString(config, "expected_body"));
    const headers = $derived(readStringRecord(config, "headers"));
    const body = $derived(readString(config, "body"));
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const curlSnippet = $derived(
        buildCurlCommand({ method, url, headers, body, insecure: skipTLSVerify }),
    );
    const expectation = $derived(
        [
            `HTTP ${expectedStatus}`,
            expectedBody ? `body contains "${expectedBody}"` : null,
        ]
            .filter(Boolean)
            .join(", "),
    );
    const headerEntries = $derived(Object.entries(headers));
    const headerText = $derived(
        headerEntries
            .map(([key, value]) => `${key}: ${value}`)
            .join("\n"),
    );
    const lastStatusCode = $derived(latestCheck?.status_code);
</script>

<CheckCardShell
    typeLabel="HTTP"
    description="updu sends an HTTP request and validates the response."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Last Status"
            value={lastStatusCode ?? undefined}
        />
        <FieldTile label="Expected" value={`HTTP ${expectedStatus}`} />
        <FieldTile label="Method" value={method.toUpperCase()} />
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
                {#if curlSnippet}
                    <FieldTile
                        label="curl"
                        value={curlSnippet}
                        monospace
                        multiline
                        copyable
                        copyLabel="Copy curl snippet"
                        testId="monitor-http-curl"
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
                copyLabel="Copy URL"
            />
            <FieldTile label="Method" value={method.toUpperCase()} />
            <FieldTile label="Expected Status" value={expectedStatus} />
            <FieldTile
                label="Expected Body"
                value={expectedBody}
                multiline
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

        {#if headerEntries.length > 0}
            <DetailSection title="Headers">
                <FieldTile
                    label={`Headers (${headerEntries.length})`}
                    value={headerText}
                    monospace
                    multiline
                    copyable
                    copyLabel="Copy headers"
                />
            </DetailSection>
        {/if}

        {#if body}
            <DetailSection title="Request Body">
                <FieldTile
                    label="Body"
                    value={body}
                    monospace
                    multiline
                    copyable
                    copyLabel="Copy body"
                />
            </DetailSection>
        {/if}
    {/snippet}
</CheckCardShell>
