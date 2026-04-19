<script lang="ts">
    import type { Snippet } from "svelte";
    import { ChevronDown, ChevronUp } from "lucide-svelte";

    let {
        typeLabel,
        description,
        hasDetails = false,
        hero,
        basics,
        details,
    }: {
        typeLabel: string;
        description: string;
        hasDetails?: boolean;
        hero: Snippet;
        basics?: Snippet;
        details?: Snippet;
    } = $props();

    let showDetails = $state(false);
</script>

<div class="card overflow-hidden" style="padding: 0;">
    <div class="px-4 py-4 sm:px-5 sm:py-5 space-y-4">
        <div
            class="flex flex-col sm:flex-row sm:items-start justify-between gap-3"
        >
            <div class="space-y-1">
                <p
                    class="text-[11px] font-semibold uppercase tracking-[0.18em] text-text-subtle"
                >
                    What This Monitor Checks
                </p>
                <h2 class="text-lg font-semibold text-text">
                    {typeLabel}
                </h2>
                <p class="text-sm text-text-muted">
                    {description}
                </p>
            </div>

            {#if hasDetails && details}
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
            {/if}
        </div>

        {#if basics}
            <div class="space-y-2">
                <p
                    class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle"
                >
                    Current Basics
                </p>
                <div
                    data-testid="monitor-current-basics"
                    class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
                >
                    {@render basics()}
                </div>
            </div>
        {/if}

        <div
            data-testid="monitor-check-summary"
            class="space-y-3"
        >
            {@render hero()}
        </div>
    </div>

    {#if hasDetails && details && showDetails}
        <div
            id="monitor-check-details"
            data-testid="monitor-check-details"
            class="border-t border-border bg-surface/20 px-4 py-4 sm:px-5 sm:py-5 space-y-4"
        >
            {@render details()}
        </div>
    {/if}
</div>
