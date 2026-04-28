<script lang="ts">
    // Global confirmation dialog. Mount once in root layout; driven by confirmStore.
    import { Dialog } from 'bits-ui';
    import { AlertOctagon, Info, X } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import { confirmStore } from '$lib/stores/confirm.svelte';

    const current = $derived(confirmStore.current);
    const loading = $derived(confirmStore.loading);
    const open = $derived(current !== null);
    let typedText = $state('');
    let remainingSeconds = $state(0);

    const isDestructive = $derived(current?.variant !== 'default');
    const requiredText = $derived(
        current?.requireText ?? (isDestructive ? 'DELETE' : ''),
    );
    const canConfirm = $derived(
        !loading &&
            remainingSeconds === 0 &&
            (!requiredText || typedText.trim() === requiredText),
    );

    $effect(() => {
        if (!current) {
            typedText = '';
            remainingSeconds = 0;
            return;
        }

        typedText = '';
        const initialRemaining = current.countdownSeconds ?? (isDestructive ? 2 : 0);
        remainingSeconds = initialRemaining;
        if (initialRemaining <= 0) return;

        const timer = setInterval(() => {
            remainingSeconds = Math.max(0, remainingSeconds - 1);
        }, 1000);
        return () => clearInterval(timer);
    });

    function handleOpenChange(value: boolean) {
        if (!value && current) {
            confirmStore.cancel();
        }
    }
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-[60] bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-[60] w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface-elevated/95 p-6 shadow-[var(--shadow-dialog)] backdrop-blur-2xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            {#if current}
                <div class="flex items-start gap-3">
                    <div
                        class="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl {isDestructive
                            ? 'bg-danger/10 text-danger'
                            : 'bg-primary/10 text-primary'}"
                    >
                        {#if isDestructive}
                            <AlertOctagon class="size-4" />
                        {:else}
                            <Info class="size-4" />
                        {/if}
                    </div>
                    <div class="min-w-0 flex-1">
                        <div class="flex items-start justify-between gap-3">
                            <div class="min-w-0">
                                <Dialog.Title class="text-base font-semibold text-text">
                                    {current.title}
                                </Dialog.Title>
                                <Dialog.Description
                                    class="mt-1.5 text-xs leading-relaxed text-text-muted"
                                >
                                    {current.description}
                                </Dialog.Description>
                            </div>
                            <button
                                type="button"
                                class="inline-flex size-7 items-center justify-center rounded-lg text-text-muted transition-colors hover:bg-surface hover:text-text"
                                onclick={() => confirmStore.cancel()}
                                aria-label="Close dialog"
                            >
                                <X class="size-4" />
                            </button>
                        </div>

                        {#if requiredText}
                            <div class="mt-5 rounded-xl border border-danger/20 bg-danger/5 p-3">
                                <p class="mb-2 inline-flex items-center rounded-full border border-danger/20 bg-danger/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-danger">
                                    Danger zone
                                </p>
                                <label
                                    for="confirm-required-text"
                                    class="text-xs font-medium text-text"
                                >
                                    Type <span class="font-mono font-bold text-danger">{requiredText}</span> to confirm
                                </label>
                                <input
                                    id="confirm-required-text"
                                    class="mt-2 h-9 w-full rounded-lg border border-border bg-background px-3 font-mono text-sm text-text outline-none transition-colors placeholder:text-text-subtle focus:border-danger/50 focus:ring-2 focus:ring-danger/15"
                                    autocomplete="off"
                                    autocapitalize="off"
                                    spellcheck="false"
                                    bind:value={typedText}
                                    disabled={loading}
                                    aria-describedby="confirm-required-help"
                                />
                                <p id="confirm-required-help" class="mt-2 text-[11px] text-text-muted">
                                    {remainingSeconds > 0
                                        ? `Available in ${remainingSeconds}s.`
                                        : 'This extra step prevents accidental destructive actions.'}
                                </p>
                            </div>
                        {/if}

                        <div class="mt-6 flex justify-end gap-2">
                            <Button variant="outline" onclick={() => confirmStore.cancel()}>
                                {current.cancelLabel ?? 'Cancel'}
                            </Button>
                            <Button
                                variant={isDestructive ? 'destructive' : 'default'}
                                {loading}
                                disabled={!canConfirm}
                                onclick={() => confirmStore.accept()}
                            >
                                {remainingSeconds > 0
                                    ? `Wait ${remainingSeconds}s`
                                    : current.confirmLabel ?? 'Confirm'}
                            </Button>
                        </div>
                    </div>
                </div>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
