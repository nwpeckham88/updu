<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/stores';
    import { goto } from '$app/navigation';
    import { Activity } from 'lucide-svelte';
    import Spinner from '$lib/components/ui/spinner.svelte';
    import { toastStore } from '$lib/stores/toast.svelte';

    const from = $page.url.searchParams.get('from') || 'v0.5.1';
    const to = $page.url.searchParams.get('to') || 'v0.6.0';

    let dots = $state('');

    onMount(() => {
        const interval = setInterval(() => {
            dots = dots.length >= 3 ? '' : dots + '.';
        }, 500);

        // Poll health check
        const poll = setInterval(async () => {
            try {
                const res = await fetch('/healthz');
                if (res.ok) {
                    const data = await res.json();
                    if (data.version === to) {
                        clearInterval(poll);
                        clearInterval(interval);
                        toastStore.success(`Updated to ${to} successfully`);
                        goto('/settings/system');
                    }
                }
            } catch {
                // Ignore failures while restarting
            }
        }, 2000);

        // Timeout after 60 seconds
        const timeout = setTimeout(() => {
            clearInterval(poll);
            clearInterval(interval);
            toastStore.error('Update timed out. Please check logs.');
            goto('/settings/system');
        }, 60000);

        return () => {
            clearInterval(interval);
            clearInterval(poll);
            clearTimeout(timeout);
        };
    });
</script>

<div class="min-h-screen bg-background flex items-center justify-center p-6">
    <div class="max-w-md w-full text-center space-y-8 animate-fade-in">
        <div class="flex flex-col items-center gap-4">
            <div class="relative">
                <div class="size-16 rounded-2xl bg-primary/10 flex items-center justify-center border border-primary/20">
                    <Activity class="size-8 text-primary" />
                </div>
                <div class="absolute -bottom-1 -right-1 size-4 bg-primary rounded-full border-2 border-background animate-pulse"></div>
            </div>
            
            <div class="space-y-2">
                <h1 class="text-2xl font-bold text-text tracking-tight">Updating updu</h1>
                <p class="text-text-muted text-sm">
                    Downloading and applying {to}{dots}
                </p>
            </div>
        </div>

        <div class="bg-surface border border-border rounded-2xl p-6 space-y-6">
            <div class="flex items-center justify-center gap-8">
                <div class="text-center">
                    <p class="text-[10px] font-bold uppercase tracking-widest text-text-subtle mb-1">From</p>
                    <p class="font-mono text-sm font-semibold">{from}</p>
                </div>
                <div class="h-8 w-px bg-border"></div>
                <div class="text-center">
                    <p class="text-[10px] font-bold uppercase tracking-widest text-text-subtle mb-1">To</p>
                    <p class="font-mono text-sm font-semibold text-primary">{to}</p>
                </div>
            </div>

            <div class="space-y-3">
                <div class="flex items-center gap-3 text-sm text-text-muted justify-center">
                    <Spinner size="sm" />
                    <span>Restarting process...</span>
                </div>
                <p class="text-[11px] text-text-subtle italic">
                    The connection will drop briefly while the new binary starts.
                </p>
            </div>
        </div>
    </div>
</div>
