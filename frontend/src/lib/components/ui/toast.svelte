<script lang="ts">
    import { CheckCircle2, Info, AlertTriangle, XCircle, X } from 'lucide-svelte';
    import { toastStore, type Toast } from '$lib/stores/toast.svelte';
    import { fade, slide } from '$lib/transitions';
    import { flip } from 'svelte/animate';

    const toasts = $derived(toastStore.items);

    const variantStyles: Record<Toast['variant'], { bg: string; ring: string; iconColor: string; Icon: typeof CheckCircle2 }> = {
        success: {
            bg: 'bg-success/10',
            ring: 'ring-success/30',
            iconColor: 'text-success',
            Icon: CheckCircle2,
        },
        info: {
            bg: 'bg-primary/10',
            ring: 'ring-primary/30',
            iconColor: 'text-primary',
            Icon: Info,
        },
        warning: {
            bg: 'bg-warning/10',
            ring: 'ring-warning/30',
            iconColor: 'text-warning',
            Icon: AlertTriangle,
        },
        error: {
            bg: 'bg-danger/10',
            ring: 'ring-danger/30',
            iconColor: 'text-danger',
            Icon: XCircle,
        },
    };
</script>

<div
    class="pointer-events-none fixed inset-x-0 top-4 z-[70] flex flex-col items-center gap-2 px-4 sm:right-4 sm:left-auto sm:top-4 sm:items-end sm:px-0"
    role="region"
    aria-label="Notifications"
>
    {#each toasts as toast (toast.id)}
        {@const v = variantStyles[toast.variant]}
        <div
            class="pointer-events-auto w-full max-w-sm overflow-hidden rounded-xl border border-border bg-surface-elevated shadow-[var(--shadow-dialog)] backdrop-blur-xl ring-1 {v.ring}"
            role={toast.variant === 'error' ? 'alert' : 'status'}
            aria-live={toast.variant === 'error' ? 'assertive' : 'polite'}
            onmouseenter={() => toastStore.pause(toast.id)}
            onmouseleave={() => toastStore.resume(toast.id)}
            onfocusin={() => toastStore.pause(toast.id)}
            onfocusout={() => toastStore.resume(toast.id)}
            in:slide={{ axis: 'y', distance: -12 }}
            out:fade
            animate:flip={{ duration: 200 }}
        >
            <div class="flex items-start gap-3 p-3">
                <div class="flex size-8 shrink-0 items-center justify-center rounded-lg {v.bg}">
                    <v.Icon class="size-4 {v.iconColor}" />
                </div>
                <div class="min-w-0 flex-1 pt-0.5">
                    <p class="text-sm font-semibold leading-tight text-text">
                        {toast.title}
                    </p>
                    {#if toast.description}
                        <p class="mt-1 text-xs leading-relaxed text-text-muted">
                            {toast.description}
                        </p>
                    {/if}
                </div>
                <button
                    type="button"
                    class="flex size-6 shrink-0 items-center justify-center rounded-md text-text-subtle transition-colors hover:bg-surface hover:text-text"
                    onclick={() => toastStore.dismiss(toast.id)}
                    aria-label="Dismiss notification"
                >
                    <X class="size-3.5" />
                </button>
            </div>
        </div>
    {/each}
</div>
