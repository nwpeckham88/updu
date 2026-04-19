<script lang="ts">
    // Form field wrapper. Pairs label + control + help/error in a consistent layout.
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    interface Props {
        id?: string;
        label?: string;
        required?: boolean;
        hint?: string;
        error?: string;
        class?: string;
        children: Snippet<[{ id?: string; describedBy?: string; invalid: boolean }]>;
    }

    let {
        id,
        label,
        required = false,
        hint,
        error,
        class: className,
        children,
    }: Props = $props();

    const describedBy = $derived(
        error ? `${id}-error` : hint ? `${id}-hint` : undefined,
    );
    const invalid = $derived(!!error);
</script>

<div class={cn("flex flex-col gap-1.5", className)}>
    {#if label}
        <label for={id} class="text-sm font-medium text-text-muted">
            {label}
            {#if required}<span class="ml-0.5 text-danger">*</span>{/if}
        </label>
    {/if}
    {@render children({ id, describedBy, invalid })}
    {#if error}
        <p id="{id}-error" class="text-xs text-danger">{error}</p>
    {:else if hint}
        <p id="{id}-hint" class="text-xs text-text-subtle">{hint}</p>
    {/if}
</div>
