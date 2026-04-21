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
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor, latestCheck }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const metadata = $derived(parseCheckMetadata(latestCheck?.metadata));

    const host = $derived(readString(config, "host"));
    const port = $derived(readNumber(config, "port") ?? 443);
    const warnDays = $derived(readNumber(config, "days_before_expiry") ?? 7);
    const cadence = $derived(monitor.interval_s);
    const endpoint = $derived(host ? `${host}:${port}` : undefined);

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

    const certTone = $derived.by((): "success" | "warning" | "danger" | "default" => {
        if (certDaysRemaining === undefined) return "default";
        if (certDaysRemaining < 0) return "danger";
        if (certDaysRemaining < warnDays) return "warning";
        return "success";
    });

    const daysLabel = $derived(
        certDaysRemaining === undefined
            ? undefined
            : certDaysRemaining < 0
              ? `Expired ${Math.abs(certDaysRemaining)}d ago`
              : `${certDaysRemaining} days`,
    );

    const expiresFormatted = $derived(
        certNotAfter ? new Date(certNotAfter).toLocaleString() : undefined,
    );

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
</script>

<CheckCardShell
    typeLabel="SSL / TLS"
    description="updu opens a TLS handshake and inspects the certificate expiry."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Certificate Expires"
            value={expiresFormatted}
            testId="monitor-basic-certificate-expires"
        />
        <FieldTile
            label="Days Left"
            value={daysLabel}
            tone={certTone}
            testId="monitor-basic-days-left"
        />
        <FieldTile label="Warn Threshold" value={`${warnDays} days`} />
        <FieldTile label="Verification" value={certVerification} />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div
            class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-3"
        >
            <div class="flex items-center gap-2">
                <ShieldCheck class="size-4 text-primary" />
                <p
                    class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                >
                    Certificate Endpoint
                </p>
            </div>
            <p
                data-testid="monitor-ssl-endpoint"
                class="font-mono text-base sm:text-lg break-all text-text"
            >
                {endpoint ?? "—"}
            </p>
            <p class="text-sm text-text-muted">
                Alerts fire when fewer than {warnDays} days remain on the certificate.
            </p>
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Endpoint">
            <FieldTile label="Host" value={host} monospace copyable={Boolean(host)} />
            <FieldTile label="Port" value={port} />
            <FieldTile label="Warn Threshold" value={`${warnDays} days`} />
        </DetailSection>

        {#if certSubject || certIssuer || certVerification}
            <DetailSection title="Latest Certificate">
                <FieldTile label="Valid From" value={formatTimestamp(certNotBefore)} />
                <FieldTile label="Valid Until" value={formatTimestamp(certNotAfter)} />
                <FieldTile label="Days Left" value={daysLabel} />
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
    {/snippet}
</CheckCardShell>
