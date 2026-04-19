<script lang="ts">
    // Tabs built on bits-ui Tabs primitive. Supports horizontal (default) + vertical.
    import { Tabs } from "bits-ui";
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    export interface TabItem {
        value: string;
        label: string;
        icon?: Snippet;
        badge?: string;
    }

    interface Props {
        items: TabItem[];
        value?: string;
        orientation?: "horizontal" | "vertical";
        class?: string;
        listClass?: string;
        children: Snippet<[{ value: string }]>;
    }

    let {
        items,
        value = $bindable(items[0]?.value ?? ""),
        orientation = "horizontal",
        class: className,
        listClass,
        children,
    }: Props = $props();
</script>

<Tabs.Root
    bind:value
    {orientation}
    class={cn(
        orientation === "vertical"
            ? "flex w-full gap-6"
            : "flex w-full flex-col gap-4",
        className,
    )}
>
    <Tabs.List
        class={cn(
            orientation === "vertical"
                ? "flex w-48 shrink-0 flex-col gap-0.5 border-r border-border pr-3"
                : "flex items-center gap-1 overflow-x-auto border-b border-border",
            listClass,
        )}
    >
        {#each items as item (item.value)}
            <Tabs.Trigger
                value={item.value}
                class={cn(
                    "inline-flex items-center gap-2 text-sm font-medium text-text-muted transition-colors duration-150 hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
                    orientation === "vertical"
                        ? "justify-start rounded-md px-3 py-2 data-[state=active]:bg-primary/10 data-[state=active]:text-primary"
                        : "relative -mb-px border-b-2 border-transparent px-3 py-2.5 data-[state=active]:border-primary data-[state=active]:text-primary",
                )}
            >
                {#if item.icon}
                    {@render item.icon()}
                {/if}
                <span>{item.label}</span>
                {#if item.badge}
                    <span
                        class="rounded-full bg-surface-elevated px-1.5 py-0.5 text-[10px] font-semibold text-text-muted"
                    >
                        {item.badge}
                    </span>
                {/if}
            </Tabs.Trigger>
        {/each}
    </Tabs.List>
    <div class="min-w-0 flex-1">
        {@render children({ value })}
    </div>
</Tabs.Root>
