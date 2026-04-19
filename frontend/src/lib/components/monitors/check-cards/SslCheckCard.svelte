<script lang="ts">
    import { ShieldCheck } from "lucide-svelte";
    import {
        parseCheckMetadata,
        parseMonitorConfig,
        readNumber,
        readString,
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

        {#if certSubject || certIssuer}
            <DetailSection title="Latest Certificate">
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
    {/snippet}
</CheckCardShell>
