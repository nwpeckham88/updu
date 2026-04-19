<script lang="ts">
    import {
        parseCheckMetadata,
        parseMonitorConfig,
        readBoolean,
        readNumber,
        readString,
        readStringArray,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import { buildCurlCommand } from "./_shared/curl.ts";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import HttpHero from "./_shared/HttpHero.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor, latestCheck }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const metadata = $derived(parseCheckMetadata(latestCheck?.metadata));
    const url = $derived(readString(config, "url"));
    const expectedStatus = $derived(readNumber(config, "expected_status") ?? 200);
    const expectedIPPrefix = $derived(readString(config, "expected_ip_prefix"));
    const expectedCNAME = $derived(readString(config, "expected_cname"));
    const expectedBody = $derived(readString(config, "expected_body"));
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const resolvedIPs = $derived(readStringArray(metadata, "resolved_ips"));
    const hostname = $derived(readString(metadata, "hostname"));

    const curlSnippet = $derived(
        buildCurlCommand({ method: "GET", url, insecure: skipTLSVerify }),
    );
    const expectation = $derived(
        [
            `HTTP ${expectedStatus}`,
            expectedIPPrefix ? `IP starts with ${expectedIPPrefix}` : null,
            expectedCNAME ? `CNAME ${expectedCNAME}` : null,
            expectedBody ? `body contains "${expectedBody}"` : null,
        ]
            .filter(Boolean)
            .join(", "),
    );
</script>

<CheckCardShell
    typeLabel="DNS + HTTP"
    description="updu resolves the URL's host, optionally checks the IP/CNAME, then makes the HTTP request."
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Hostname" value={hostname} monospace />
        <FieldTile
            label="Resolved IPs"
            value={resolvedIPs.length > 0 ? resolvedIPs.join(", ") : undefined}
            monospace
        />
        <FieldTile label="Expected Status" value={expectedStatus} />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <HttpHero method="GET" {url} />

            <div class="grid gap-3 sm:grid-cols-2">
                <FieldTile label="Expectation" value={expectation} />
                {#if curlSnippet}
                    <FieldTile
                        label="curl"
                        value={curlSnippet}
                        monospace
                        multiline
                        copyable
                    />
                {/if}
                {#if resolvedIPs.length > 0}
                    <FieldTile
                        label="Latest Resolved IPs"
                        value={resolvedIPs.join("\n")}
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
                label="URL"
                value={url}
                href={url}
                monospace
                copyable={Boolean(url)}
            />
            <FieldTile label="Expected Status" value={expectedStatus} />
            <FieldTile
                label="Expected IP Prefix"
                value={expectedIPPrefix}
                monospace
            />
            <FieldTile
                label="Expected CNAME"
                value={expectedCNAME}
                monospace
            />
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
    {/snippet}
</CheckCardShell>
