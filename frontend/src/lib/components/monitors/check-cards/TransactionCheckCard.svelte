<script lang="ts">
    import {
        parseMonitorConfig,
        readBoolean,
        readNumber,
        readRecord,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import { buildCurlCommand } from "./_shared/curl.ts";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    interface TxStep {
        method: string;
        url?: string;
        expected_status?: number;
        expected_body?: string;
        headers: Record<string, string>;
        body?: string;
        extract: Record<string, string>;
    }

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));
    const skipTLSVerify = $derived(readBoolean(config, "skip_tls_verify"));
    const cadence = $derived(monitor.interval_s);

    const steps = $derived.by((): TxStep[] => {
        const raw = Array.isArray(config.steps) ? config.steps : [];
        return raw.flatMap((entry: unknown) => {
            if (typeof entry !== "object" || entry === null || Array.isArray(entry)) {
                return [];
            }
            const record = entry as Record<string, any>;
            const headersRecord = readRecord(record, "headers");
            const headers: Record<string, string> = {};
            for (const [key, value] of Object.entries(headersRecord)) {
                if (value === undefined || value === null) continue;
                headers[key] = `${value}`;
            }
            const extractRecord = readRecord(record, "extract");
            const extract: Record<string, string> = {};
            for (const [key, value] of Object.entries(extractRecord)) {
                if (value === undefined || value === null) continue;
                extract[key] = `${value}`;
            }
            return [
                {
                    method: readString(record, "method") ?? "GET",
                    url: readString(record, "url"),
                    expected_status: readNumber(record, "expected_status"),
                    expected_body: readString(record, "expected_body"),
                    headers,
                    body: readString(record, "body"),
                    extract,
                },
            ];
        });
    });

    const stepCount = $derived(steps.length);
    const firstStep = $derived(steps[0]);
    const headline = $derived(
        firstStep?.url
            ? `${firstStep.method.toUpperCase()} ${firstStep.url}`
            : firstStep?.method,
    );
</script>

<CheckCardShell
    typeLabel="Transaction"
    description="updu walks through a series of HTTP requests, asserting each step before moving on."
    hasDetails
>
    {#snippet basics()}
        <FieldTile
            label="Steps"
            value={stepCount > 0 ? `${stepCount} step${stepCount === 1 ? "" : "s"}` : undefined}
        />
        <FieldTile
            label="Starts With"
            value={firstStep?.method.toUpperCase()}
        />
        <FieldTile
            label="TLS"
            value={skipTLSVerify ? "Skipped" : "Verified"}
            tone={skipTLSVerify ? "warning" : "default"}
        />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div class="space-y-3">
            <div
                class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-2"
            >
                <p
                    class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                >
                    Flow Summary
                </p>
                <p class="text-sm text-text">
                    {stepCount} step{stepCount === 1 ? "" : "s"} starting with
                    <code class="font-mono text-xs">{headline ?? "—"}</code>
                </p>
            </div>

            <div class="space-y-2">
                {#each steps as step, index (`tx-summary-${index}`)}
                    <div
                        class="flex items-center gap-3 rounded-xl border border-border/70 bg-background/60 p-3"
                    >
                        <div
                            class="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary"
                        >
                            {index + 1}
                        </div>
                        <div class="min-w-0 flex-1">
                            <p class="font-mono text-xs text-text break-all">
                                {step.method.toUpperCase()} {step.url ?? "(no URL)"}
                            </p>
                            <p class="text-[11px] text-text-muted">
                                Expected HTTP {step.expected_status ?? 200}{step.expected_body ? `, body contains "${step.expected_body}"` : ""}
                            </p>
                        </div>
                        {#if step.url}
                            <a
                                href={step.url}
                                target="_blank"
                                rel="noopener noreferrer"
                                class="text-[11px] text-primary hover:underline"
                            >
                                Open
                            </a>
                        {/if}
                    </div>
                {/each}
            </div>
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Configuration">
            <FieldTile label="Step Count" value={stepCount} />
            <FieldTile
                label="Skip TLS Verification"
                value={skipTLSVerify === undefined
                    ? undefined
                    : skipTLSVerify
                      ? "Yes"
                      : "No"}
            />
        </DetailSection>

        {#each steps as step, index (`monitor-detail-transaction-step-${index + 1}`)}
            {@const stepId = `monitor-detail-transaction-step-${index + 1}`}
            {@const stepCurl = buildCurlCommand({
                method: step.method,
                url: step.url,
                headers: step.headers,
                body: step.body,
                insecure: skipTLSVerify,
            })}
            <section class="space-y-3">
                <h3
                    class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle"
                >
                    Step {index + 1}
                </h3>
                <div
                    data-testid={stepId}
                    class="rounded-xl border border-border/70 bg-background/60 p-4 space-y-3"
                >
                    <div class="flex items-start justify-between gap-2">
                        <p class="font-mono text-xs text-text break-all">
                            {step.method.toUpperCase()} {step.url ?? ""}
                        </p>
                    </div>
                    <div class="grid gap-3 sm:grid-cols-2">
                        <FieldTile
                            label="URL"
                            value={step.url}
                            href={step.url}
                            monospace
                            copyable={Boolean(step.url)}
                        />
                        <FieldTile
                            label="Expected Status"
                            value={step.expected_status}
                        />
                        <FieldTile
                            label="Expected Body"
                            value={step.expected_body}
                            multiline
                        />
                        {#if Object.keys(step.headers).length > 0}
                            <FieldTile
                                label="Headers"
                                value={Object.entries(step.headers)
                                    .map(([k, v]) => `${k}: ${v}`)
                                    .join("\n")}
                                monospace
                                multiline
                                copyable
                            />
                        {/if}
                        {#if step.body}
                            <FieldTile
                                label="Request Body"
                                value={step.body}
                                monospace
                                multiline
                                copyable
                            />
                        {/if}
                        {#if Object.keys(step.extract).length > 0}
                            <FieldTile
                                label="Extract"
                                value={Object.entries(step.extract)
                                    .map(([k, v]) => `${k} <- ${v}`)
                                    .join("\n")}
                                monospace
                                multiline
                            />
                        {/if}
                        {#if stepCurl}
                            <FieldTile
                                label="curl"
                                value={stepCurl}
                                monospace
                                multiline
                                copyable
                                copyLabel={`Copy curl for step ${index + 1}`}
                            />
                        {/if}
                    </div>
                </div>
            </section>
        {/each}
    {/snippet}
</CheckCardShell>
