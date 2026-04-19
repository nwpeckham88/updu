<script lang="ts">
    import { ShieldCheck } from "lucide-svelte";
    import {
        parseCheckMetadata,
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
    const metadata = $derived(parseCheckMetadata(latestCheck?.metadata));
    const method = $derived(readString(config, "method") ?? "GET");
    const url = $derived(readString(config, "url"));
    const expectedStatus = $derived(readNumber(config, "expected_status") ?? 200);
    const expectedBody = $derived(readString(config, "expected_body"));
    const headers = $derived(readStringRecord(config, "headers"));
    const body = $derived(readString(config, "body"));
    const warnDays = $derived(readNumber(config, "warn_days") ?? 14);
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const curlSnippet = $derived(
        buildCurlCommand({ method, url, headers, body, insecure: skipTLSVerify }),
    );

    const certNotAfter = $derived(readString(metadata, "cert_not_after"));
    const certDaysRemaining = $derived(readNumber(metadata, "cert_days_remaining"));
    const certSubject = $derived(readString(metadata, "cert_subject"));
    const certIssuer = $derived(readString(metadata, "cert_issuer"));
    const certNotBefore = $derived(readString(metadata, "cert_not_before"));

    function formatDate(iso?: string): string | undefined {
        if (!iso) return undefined;
        const parsed = new Date(iso);
        if (Number.isNaN(parsed.getTime())) return iso;
        return parsed.toISOString().slice(0, 10);
    }

    function formatTimestamp(iso?: string): string | undefined {
        if (!iso) return undefined;
        const parsed = new Date(iso);
        if (Number.isNaN(parsed.getTime())) return iso;
        return `${parsed.toISOString().slice(0, 16).replace("T", " ")} UTC`;
    }

    const certTone = $derived.by((): "success" | "warning" | "danger" | "default" => {
        if (certDaysRemaining === undefined) return "default";
        if (certDaysRemaining < 0) return "danger";
        if (certDaysRemaining < warnDays) return "warning";
        return "success";
    });

    const headerEntries = $derived(Object.entries(headers));
    const headerText = $derived(
        headerEntries.map(([k, v]) => `${k}: ${v}`).join("\n"),
    );
    const lastStatusCode = $derived(latestCheck?.status_code);
</script>

<CheckCardShell
    typeLabel="HTTPS"
    description="updu sends an HTTPS request and validates the response and the TLS certificate."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Last Status"
            value={lastStatusCode ?? undefined}
        />
        <FieldTile
            label="Cert Expires"
            value={formatDate(certNotAfter)}
            tone={certTone}
            testId="monitor-basic-certificate-expires"
        />
        <FieldTile
            label="Days Left"
            value={certDaysRemaining ?? undefined}
            tone={certTone}
            testId="monitor-basic-days-left"
        />
        <FieldTile
            label="Warn Threshold"
            value={`${warnDays}d`}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <HttpHero {method} {url} />

            <div class="grid gap-3 sm:grid-cols-2">
                <FieldTile
                    label="Expectation"
                    value={`HTTP ${expectedStatus}, TLS valid > ${warnDays}d${expectedBody ? `, body contains "${expectedBody}"` : ""}`}
                />
                {#if curlSnippet}
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

            {#if certSubject || certIssuer}
                <div
                    class="rounded-xl border border-border/70 bg-background/60 p-3 space-y-2"
                >
                    <div class="flex items-center gap-2 text-text-subtle">
                        <ShieldCheck class="size-3.5" />
                        <p
                            class="text-[10px] font-semibold uppercase tracking-[0.16em]"
                        >
                            Latest Certificate
                        </p>
                    </div>
                    <div class="grid gap-2 text-xs sm:grid-cols-2">
                        {#if certSubject}
                            <p class="font-mono break-all text-text">
                                <span class="text-text-subtle">Subject:</span>
                                {certSubject}
                            </p>
                        {/if}
                        {#if certIssuer}
                            <p class="font-mono break-all text-text">
                                <span class="text-text-subtle">Issuer:</span>
                                {certIssuer}
                            </p>
                        {/if}
                    </div>
                </div>
            {/if}
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
            <FieldTile label="Expected Status" value={expectedStatus} />
            <FieldTile label="Expected Body" value={expectedBody} multiline />
            <FieldTile label="TLS Warn Threshold" value={`${warnDays} days`} />
            <FieldTile
                label="Skip TLS Verification"
                value={skipTLSVerify === undefined
                    ? undefined
                    : skipTLSVerify
                      ? "Yes"
                      : "No"}
            />
            <FieldTile
                label="Cadence"
                value={cadence ? `Every ${cadence}s` : undefined}
            />
        </DetailSection>

        {#if certNotAfter || certNotBefore || certSubject || certIssuer}
            <DetailSection title="Latest Certificate">
                <FieldTile label="Valid From" value={formatTimestamp(certNotBefore)} />
                <FieldTile label="Valid Until" value={formatTimestamp(certNotAfter)} />
                <FieldTile
                    label="Days Left"
                    value={certDaysRemaining ?? undefined}
                />
                <FieldTile
                    label="Subject"
                    value={certSubject}
                    monospace
                    copyable={Boolean(certSubject)}
                />
                <FieldTile
                    label="Issuer"
                    value={certIssuer}
                    monospace
                    copyable={Boolean(certIssuer)}
                />
            </DetailSection>
        {/if}

        {#if headerEntries.length > 0}
            <DetailSection title="Headers">
                <FieldTile
                    label={`Headers (${headerEntries.length})`}
                    value={headerText}
                    monospace
                    multiline
                    copyable
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
                />
            </DetailSection>
        {/if}
    {/snippet}
</CheckCardShell>
