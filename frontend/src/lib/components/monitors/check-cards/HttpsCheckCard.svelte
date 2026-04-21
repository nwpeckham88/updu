<script lang="ts">
    import { ShieldCheck } from "lucide-svelte";
    import {
        formatPublicKeySummary,
        formatTLSVerification,
        parseCheckMetadata,
        parseMonitorConfig,
        readBoolean,
        readNumber,
        readString,
        readStringArray,
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
    const certSerialNumber = $derived(readString(metadata, "cert_serial_number"));
    const certFingerprint = $derived(readString(metadata, "cert_fingerprint_sha256"));
    const certSignatureAlgorithm = $derived(readString(metadata, "cert_signature_algorithm"));
    const certPublicKeyAlgorithm = $derived(readString(metadata, "cert_public_key_algorithm"));
    const certPublicKeyBits = $derived(readNumber(metadata, "cert_public_key_bits"));
    const certDNSNames = $derived(readStringArray(metadata, "cert_dns_names"));
    const certIPAddresses = $derived(readStringArray(metadata, "cert_ip_addresses"));
    const certVerificationMode = $derived(readString(metadata, "cert_tls_verification_mode"));
    const certVerified = $derived(readBoolean(metadata, "cert_tls_verified"));
    const certChainLength = $derived(readNumber(metadata, "cert_chain_length"));
    const certChainSummary = $derived(readStringArray(metadata, "cert_chain_summary"));
    const certVerification = $derived(
        formatTLSVerification(certVerificationMode, certVerified),
    );
    const certPublicKey = $derived(
        formatPublicKeySummary(certPublicKeyAlgorithm, certPublicKeyBits),
    );

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

    function formatList(values: string[]): string | undefined {
        return values.length > 0 ? values.join("\n") : undefined;
    }

    function formatChainLength(length?: number): string | undefined {
        if (length === undefined) return undefined;
        return `${length} certificate${length === 1 ? "" : "s"}`;
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
            label="Certificate Expires"
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
        <FieldTile label="Verification" value={certVerification} />
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

            {#if certSubject || certIssuer || certVerification}
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
                        {#if certVerification}
                            <p class="text-text">
                                <span class="text-text-subtle">Verification:</span>
                                {certVerification}
                            </p>
                        {/if}
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
                <FieldTile label="Verification" value={certVerification} />
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
                <FieldTile
                    label="Serial Number"
                    value={certSerialNumber}
                    monospace
                    copyable={Boolean(certSerialNumber)}
                />
                <FieldTile
                    label="SHA-256 Fingerprint"
                    value={certFingerprint}
                    monospace
                    multiline
                    copyable={Boolean(certFingerprint)}
                />
                <FieldTile label="Signature Algorithm" value={certSignatureAlgorithm} />
                <FieldTile label="Public Key" value={certPublicKey} />
                <FieldTile
                    label="DNS Names"
                    value={formatList(certDNSNames)}
                    monospace
                    multiline={certDNSNames.length > 1}
                    copyable={certDNSNames.length > 0}
                />
                <FieldTile
                    label="IP Addresses"
                    value={formatList(certIPAddresses)}
                    monospace
                    multiline={certIPAddresses.length > 1}
                    copyable={certIPAddresses.length > 0}
                />
                <FieldTile
                    label="Presented Chain"
                    value={formatChainLength(certChainLength ?? undefined)}
                />
                <FieldTile
                    label="Chain Summary"
                    value={formatList(certChainSummary)}
                    monospace
                    multiline={certChainSummary.length > 1}
                    copyable={certChainSummary.length > 0}
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
