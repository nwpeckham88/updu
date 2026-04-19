<script lang="ts">
    // Tooltip wrapper around bits-ui Tooltip. Use on icon-only buttons + truncated text.
    import { Tooltip } from "bits-ui";
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    interface Props {
        content: string;
        side?: "top" | "right" | "bottom" | "left";
        align?: "start" | "center" | "end";
        delay?: number;
        class?: string;
        children: Snippet;
    }

    let {
        content,
        side = "top",
        align = "center",
        delay = 200,
        class: className,
        children,
    }: Props = $props();
</script>

<Tooltip.Provider delayDuration={delay}>
    <Tooltip.Root>
        <Tooltip.Trigger class={cn("inline-flex", className)}>
            {@render children()}
        </Tooltip.Trigger>
        <Tooltip.Portal>
            <Tooltip.Content
                {side}
                {align}
                sideOffset={6}
                class="z-[var(--z-tooltip)] rounded-md border border-border bg-surface-elevated px-2 py-1 text-xs text-text shadow-[var(--shadow-card)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
            >
                {content}
            </Tooltip.Content>
        </Tooltip.Portal>
    </Tooltip.Root>
</Tooltip.Provider>
