<script lang="ts">
    // Themed Dialog wrapper. Use Modal.Root/Trigger/Body composition for consistent styling.
    import { Dialog } from "bits-ui";
    import { X } from "lucide-svelte";
    import type { Snippet } from "svelte";
    import { cn } from "$lib/utils";

    interface Props {
        open?: boolean;
        title?: string;
        description?: string;
        size?: "sm" | "md" | "lg" | "xl";
        contentClass?: string;
        children: Snippet;
        footer?: Snippet;
    }

    let {
        open = $bindable(false),
        title,
        description,
        size = "md",
        contentClass,
        children,
        footer,
    }: Props = $props();

    const sizeClass = $derived(
        {
            sm: "max-w-sm",
            md: "max-w-lg",
            lg: "max-w-2xl",
            xl: "max-w-4xl",
        }[size],
    );
</script>

<Dialog.Root bind:open>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-[var(--z-overlay)] bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class={cn(
                "fixed left-1/2 top-1/2 z-[var(--z-modal)] w-full -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-2xl border border-border bg-surface-elevated/95 shadow-[var(--shadow-dialog)] backdrop-blur-2xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95",
                sizeClass,
                contentClass,
            )}
        >
            {#if title || description}
                <div class="flex items-start justify-between gap-4 border-b border-border/60 px-6 py-4">
                    <div class="min-w-0">
                        {#if title}
                            <Dialog.Title class="text-base font-semibold text-text">
                                {title}
                            </Dialog.Title>
                        {/if}
                        {#if description}
                            <Dialog.Description
                                class="mt-1 text-xs leading-relaxed text-text-muted"
                            >
                                {description}
                            </Dialog.Description>
                        {/if}
                    </div>
                    <Dialog.Close
                        class="inline-flex size-7 shrink-0 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface hover:text-text"
                        aria-label="Close dialog"
                    >
                        <X class="size-4" />
                    </Dialog.Close>
                </div>
            {/if}
            <div class="max-h-[calc(100vh-12rem)] overflow-y-auto px-6 py-5">
                {@render children()}
            </div>
            {#if footer}
                <div
                    class="flex items-center justify-end gap-2 border-t border-border/60 bg-surface/40 px-6 py-3"
                >
                    {@render footer()}
                </div>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
