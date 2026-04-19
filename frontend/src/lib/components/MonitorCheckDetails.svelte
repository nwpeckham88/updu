<script lang="ts">
    import { ChevronDown, ChevronUp } from "lucide-svelte";
    import {
        describeMonitorCheck,
        type MonitorCheckResult,
        type MonitorConfigTarget,
    } from "$lib/monitor-config";

    let {
        monitor,
        latestCheck = null,
    } = $props<{
        monitor: MonitorConfigTarget;
        latestCheck?: MonitorCheckResult | null;
    }>();

    let showDetails = $state(false);

    const description = $derived(describeMonitorCheck(monitor, latestCheck));
    const monitorTypeName = $derived(description.typeLabel.toLowerCase());
    const expandedSections = $derived([
        ...description.runtimeSections,
        ...description.detailSections,
    ]);
</script>

<div class="card overflow-hidden" style="padding: 0;">
    <div class="px-4 py-4 sm:px-5 sm:py-5 space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3">
            <div class="space-y-1">
                <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-text-subtle">
                    What This Monitor Checks
                </p>
                <h2 class="text-lg font-semibold text-text">
                    {description.typeLabel}
                </h2>
                <p class="text-sm text-text-muted">
                    Target, expectations, and key configuration for this {monitorTypeName} monitor.
                </p>
            </div>

            <button
                type="button"
                class="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-text-muted bg-surface/50 border border-border rounded-md hover:text-text hover:bg-surface transition-colors"
                aria-expanded={showDetails}
                aria-controls="monitor-check-details"
                onclick={() => (showDetails = !showDetails)}
            >
                {#if showDetails}
                    <ChevronUp class="size-3.5" />
                    Hide Detailed Config
                {:else}
                    <ChevronDown class="size-3.5" />
                    Show Detailed Config
                {/if}
            </button>
        </div>

        {#if description.basicItems.length > 0}
            <div class="space-y-2">
                <p class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                    Current Basics
                </p>
                <div
                    data-testid="monitor-current-basics"
                    class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
                >
                    {#each description.basicItems as item (item.label)}
                        <div
                            data-testid={item.testId}
                            class="rounded-xl border border-border/70 bg-background/60 p-3"
                        >
                            <p class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                                {item.label}
                            </p>
                            <p
                                class="mt-1 text-sm font-medium text-text break-all {item.monospace
                                    ? 'font-mono text-xs'
                                    : ''}"
                            >
                                {item.value}
                            </p>
                        </div>
                    {/each}
                </div>
            </div>
        {/if}

        <div
            data-testid="monitor-check-summary"
            class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
        >
            {#each description.summaryItems as item (item.label)}
                <div class="rounded-xl border border-border/70 bg-background/60 p-3">
                    <p class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                        {item.label}
                    </p>
                    {#if item.href}
                        <a
                            href={item.href}
                            target="_blank"
                            rel="noopener noreferrer"
                            class="mt-1 block text-sm font-medium text-primary break-all hover:underline {item.monospace
                                ? 'font-mono text-xs'
                                : ''}"
                        >
                            {item.value}
                        </a>
                    {:else}
                        <p
                            class="mt-1 text-sm font-medium text-text break-all {item.monospace
                                ? 'font-mono text-xs'
                                : ''}"
                        >
                            {item.value}
                        </p>
                    {/if}
                </div>
            {/each}
        </div>
    </div>

    {#if showDetails}
        <div
            id="monitor-check-details"
            data-testid="monitor-check-details"
            class="border-t border-border bg-surface/20 px-4 py-4 sm:px-5 sm:py-5 space-y-4"
        >
            {#each expandedSections as section, index (`${section.title}-${index}`)}
                <section class="space-y-3">
                    <h3 class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                        {section.title}
                    </h3>
                    <div class="grid gap-3 sm:grid-cols-2">
                        {#each section.rows as row (`${section.title}-${row.label}`)}
                            <div
                                class="rounded-xl border border-border/70 bg-background/60 p-3 {row.multiline
                                    ? 'sm:col-span-2'
                                    : ''}"
                            >
                                <p class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                                    {row.label}
                                </p>
                                {#if row.href}
                                    <a
                                        href={row.href}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="mt-1 block break-all text-primary hover:underline {row.monospace
                                            ? 'font-mono text-xs'
                                            : 'text-sm font-medium'} {row.multiline
                                            ? 'whitespace-pre-wrap'
                                            : ''}"
                                    >
                                        {row.value}
                                    </a>
                                {:else if row.multiline}
                                    <code
                                        class="mt-1 block whitespace-pre-wrap break-all text-text {row.monospace
                                            ? 'font-mono text-xs'
                                            : 'text-sm font-medium'}"
                                        >{row.value}</code
                                    >
                                {:else}
                                    <p
                                        class="mt-1 break-all text-text {row.monospace
                                            ? 'font-mono text-xs'
                                            : 'text-sm font-medium'}"
                                    >
                                        {row.value}
                                    </p>
                                {/if}
                            </div>
                        {/each}
                    </div>
                </section>
            {/each}

            {#if description.steps.length > 0}
                <section class="space-y-3">
                    <h3 class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                        Transaction Steps
                    </h3>
                    <div class="space-y-3">
                        {#each description.steps as step (step.id)}
                            <div
                                data-testid={step.id}
                                class="rounded-xl border border-border/70 bg-background/60 p-4 space-y-3"
                            >
                                <div class="space-y-1">
                                    <p class="text-sm font-semibold text-text">
                                        {step.title}
                                    </p>
                                    <p class="text-sm text-text-muted break-all">
                                        {step.summary}
                                    </p>
                                </div>

                                {#if step.rows.length > 0}
                                    <div class="grid gap-3 sm:grid-cols-2">
                                        {#each step.rows as row (`${step.id}-${row.label}`)}
                                            <div
                                                class="rounded-lg border border-border/60 bg-surface/40 p-3 {row.multiline
                                                    ? 'sm:col-span-2'
                                                    : ''}"
                                            >
                                                <p class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle">
                                                    {row.label}
                                                </p>
                                                {#if row.href}
                                                    <a
                                                        href={row.href}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        class="mt-1 block break-all text-primary hover:underline {row.monospace
                                                            ? 'font-mono text-xs'
                                                            : 'text-sm font-medium'} {row.multiline
                                                            ? 'whitespace-pre-wrap'
                                                            : ''}"
                                                    >
                                                        {row.value}
                                                    </a>
                                                {:else if row.multiline}
                                                    <code
                                                        class="mt-1 block whitespace-pre-wrap break-all text-text {row.monospace
                                                            ? 'font-mono text-xs'
                                                            : 'text-sm font-medium'}"
                                                        >{row.value}</code
                                                    >
                                                {:else}
                                                    <p
                                                        class="mt-1 break-all text-text {row.monospace
                                                            ? 'font-mono text-xs'
                                                            : 'text-sm font-medium'}"
                                                    >
                                                        {row.value}
                                                    </p>
                                                {/if}
                                            </div>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        {/each}
                    </div>
                </section>
            {/if}
        </div>
    {/if}
</div>
