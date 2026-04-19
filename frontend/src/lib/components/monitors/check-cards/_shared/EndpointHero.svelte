<script lang="ts">
    import { Network } from "lucide-svelte";
    import CopyButton from "./CopyButton.svelte";

    let {
        endpoint,
        headline = "Target",
        subline,
    }: {
        endpoint: string | undefined;
        headline?: string;
        subline?: string;
    } = $props();
</script>

<div
    class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-2"
>
    <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
            <Network class="size-4 text-primary" />
            <p
                class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
            >
                {headline}
            </p>
        </div>
        {#if endpoint}
            <CopyButton
                value={endpoint}
                label={`Copy ${headline.toLowerCase()}`}
                successMessage={`${headline} copied`}
                size="xs"
            />
        {/if}
    </div>
    {#if endpoint}
        <code
            class="block break-all rounded-lg bg-background/70 px-3 py-2 font-mono text-xs text-primary"
        >
            {endpoint}
        </code>
    {:else}
        <p class="text-sm text-text-muted italic">
            No target configured.
        </p>
    {/if}
    {#if subline}
        <p class="text-[11px] text-text-muted">{subline}</p>
    {/if}
</div>
