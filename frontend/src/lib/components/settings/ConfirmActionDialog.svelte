<script lang="ts">
    import { Dialog } from 'bits-ui';
    import { AlertOctagon, Info, X } from 'lucide-svelte';
    import type { Snippet } from 'svelte';
    import Button from '$lib/components/ui/button.svelte';

    interface Props {
        open?: boolean;
        title: string;
        description: string;
        confirmLabel?: string;
        cancelLabel?: string;
        confirmVariant?: 'default' | 'destructive';
        requireText?: string;
        countdownSeconds?: number;
        loading?: boolean;
        onConfirm?: (() => void | Promise<void>) | null;
        onCancel?: (() => void) | null;
        children?: Snippet;
    }

    let {
        open = $bindable(false),
        title,
        description,
        confirmLabel = 'Confirm',
        cancelLabel = 'Cancel',
        confirmVariant = 'destructive',
        requireText,
        countdownSeconds,
        loading = false,
        onConfirm,
        onCancel,
        children,
    }: Props = $props();

    let typedText = $state('');
    let remainingSeconds = $state(0);

    const isDestructive = $derived(confirmVariant === 'destructive');
    const requiredText = $derived(requireText ?? (isDestructive ? 'DELETE' : ''));
    const canConfirm = $derived(
        !loading &&
            remainingSeconds === 0 &&
            (!requiredText || typedText.trim() === requiredText),
    );

    $effect(() => {
        if (!open) {
            typedText = '';
            remainingSeconds = 0;
            return;
        }

        typedText = '';
        const initialRemaining = countdownSeconds ?? (isDestructive ? 2 : 0);
        remainingSeconds = initialRemaining;
        if (initialRemaining <= 0) return;

        const timer = setInterval(() => {
            remainingSeconds = Math.max(0, remainingSeconds - 1);
        }, 1000);
        return () => clearInterval(timer);
    });

    function closeDialog() {
        onCancel?.();
        open = false;
    }
</script>

<Dialog.Root bind:open>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-start gap-3">
                <div
                    class={`mt-0.5 size-9 rounded-xl flex items-center justify-center shrink-0 ${isDestructive ? 'bg-danger/10 text-danger' : 'bg-primary/10 text-primary'}`}
                >
                    {#if isDestructive}
                        <AlertOctagon class="size-4" />
                    {:else}
                        <Info class="size-4" />
                    {/if}
                </div>
                <div class="min-w-0 flex-1">
                    <div class="flex items-start justify-between gap-3">
                        <div>
                            <Dialog.Title class="text-base font-semibold text-text">
                                {title}
                            </Dialog.Title>
                            <Dialog.Description class="text-xs text-text-muted mt-1.5 max-w-sm">
                                {description}
                            </Dialog.Description>
                        </div>
                        <Dialog.Close
                            class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                            onclick={closeDialog}
                            aria-label="Close dialog"
                        >
                            <X class="size-4" />
                        </Dialog.Close>
                    </div>

                    {#if children}
                        <div class="mt-4 rounded-2xl border border-border/60 bg-background/50 p-4">
                            {@render children()}
                        </div>
                    {/if}

                    {#if requiredText}
                        <div class="mt-4 rounded-xl border border-danger/20 bg-danger/5 p-3">
                            <p class="mb-2 inline-flex items-center rounded-full border border-danger/20 bg-danger/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-danger">
                                Danger zone
                            </p>
                            <label
                                for="settings-confirm-required-text"
                                class="text-xs font-medium text-text"
                            >
                                Type <span class="font-mono font-bold text-danger">{requiredText}</span> to confirm
                            </label>
                            <input
                                id="settings-confirm-required-text"
                                class="mt-2 h-9 w-full rounded-lg border border-border bg-background px-3 font-mono text-sm text-text outline-none transition-colors placeholder:text-text-subtle focus:border-danger/50 focus:ring-2 focus:ring-danger/15"
                                autocomplete="off"
                                autocapitalize="off"
                                spellcheck="false"
                                bind:value={typedText}
                                disabled={loading}
                                aria-describedby="settings-confirm-required-help"
                            />
                            <p id="settings-confirm-required-help" class="mt-2 text-[11px] text-text-muted">
                                {remainingSeconds > 0
                                    ? `Available in ${remainingSeconds}s.`
                                    : 'This extra step prevents accidental destructive actions.'}
                            </p>
                        </div>
                    {/if}

                    <div class="mt-6 flex justify-end gap-2">
                        <Button
                            variant="outline"
                            onclick={closeDialog}
                        >
                            {cancelLabel}
                        </Button>
                        <Button
                            variant={isDestructive ? 'destructive' : 'default'}
                            loading={loading}
                            disabled={!canConfirm}
                            onclick={() => onConfirm?.()}
                        >
                            {loading
                                ? 'Working...'
                                : remainingSeconds > 0
                                  ? `Wait ${remainingSeconds}s`
                                  : confirmLabel}
                        </Button>
                    </div>
                </div>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>