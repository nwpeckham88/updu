<script lang="ts">
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { description }: CheckCardProps = $props();

    const monitorTypeName = $derived(description.typeLabel.toLowerCase());
    const expandedSections = $derived([
        ...description.runtimeSections,
        ...description.detailSections,
    ]);
    const hasDetails = $derived(
        expandedSections.length > 0 || description.steps.length > 0,
    );
</script>

<CheckCardShell
    typeLabel={description.typeLabel}
    description={`Target, expectations, and key configuration for this ${monitorTypeName} monitor.`}
    {hasDetails}
>
    {#snippet basics()}
        {#each description.basicItems as item (item.label)}
            <FieldTile
                label={item.label}
                value={item.value}
                href={item.href}
                monospace={item.monospace}
                multiline={item.multiline}
                testId={item.testId}
            />
        {/each}
    {/snippet}

    {#snippet hero()}
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {#each description.summaryItems as item (item.label)}
                <FieldTile
                    label={item.label}
                    value={item.value}
                    href={item.href}
                    monospace={item.monospace}
                    multiline={item.multiline}
                />
            {/each}
        </div>
    {/snippet}

    {#snippet details()}
        {#each expandedSections as section, index (`${section.title}-${index}`)}
            <DetailSection title={section.title}>
                {#each section.rows as row (`${section.title}-${row.label}`)}
                    <FieldTile
                        label={row.label}
                        value={row.value}
                        href={row.href}
                        monospace={row.monospace}
                        multiline={row.multiline}
                    />
                {/each}
            </DetailSection>
        {/each}

        {#if description.steps.length > 0}
            <DetailSection title="Transaction Steps">
                {#each description.steps as step (step.id)}
                    <div
                        data-testid={step.id}
                        class="rounded-xl border border-border/70 bg-background/60 p-4 space-y-3 sm:col-span-2"
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
                                    <FieldTile
                                        label={row.label}
                                        value={row.value}
                                        href={row.href}
                                        monospace={row.monospace}
                                        multiline={row.multiline}
                                    />
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/each}
            </DetailSection>
        {/if}
    {/snippet}
</CheckCardShell>
