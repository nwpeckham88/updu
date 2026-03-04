<script lang="ts">
    import { cn } from "$lib/utils";
    import type { Component, Snippet } from "svelte";

    interface Props {
        class?: string;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        icon?: any; // lucide-svelte components don't satisfy Component<{},{},string> exactly
        title?: string;
        description?: string;
        children?: Snippet;
    }

    let {
        class: className,
        icon: Icon,
        title,
        description,
        children,
    }: Props = $props();
</script>

<div
    class={cn(
        "flex flex-col items-center justify-center text-center py-16 px-8 gap-4",
        className,
    )}
>
    {#if Icon}
        <div
            class="size-16 rounded-2xl bg-surface flex items-center justify-center border border-border text-text-subtle"
        >
            <Icon class="size-8" />
        </div>
    {/if}
    {#if title}
        <div class="space-y-1.5">
            <h3 class="text-base font-semibold text-text">{title}</h3>
            {#if description}
                <p class="text-sm text-text-muted max-w-xs">{description}</p>
            {/if}
        </div>
    {/if}
    {#if children}
        {@render children()}
    {/if}
</div>
