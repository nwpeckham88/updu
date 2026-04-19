<script lang="ts">
    // Breadcrumbs for nested routes.
    import { ChevronRight } from "lucide-svelte";
    import { cn } from "$lib/utils";

    export interface Crumb {
        label: string;
        href?: string;
    }

    interface Props {
        items: Crumb[];
        class?: string;
    }

    let { items, class: className }: Props = $props();
</script>

<nav aria-label="Breadcrumb" class={cn("min-w-0", className)}>
    <ol class="flex items-center gap-1 overflow-x-auto text-xs">
        {#each items as item, i (i)}
            {@const isLast = i === items.length - 1}
            <li class="flex items-center gap-1">
                {#if item.href && !isLast}
                    <a
                        href={item.href}
                        class="rounded px-1 text-text-muted transition-colors hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
                    >
                        {item.label}
                    </a>
                {:else}
                    <span
                        class={cn(
                            "px-1",
                            isLast ? "font-medium text-text" : "text-text-muted",
                        )}
                        aria-current={isLast ? "page" : undefined}
                    >
                        {item.label}
                    </span>
                {/if}
                {#if !isLast}
                    <ChevronRight class="size-3 text-text-subtle" />
                {/if}
            </li>
        {/each}
    </ol>
</nav>
