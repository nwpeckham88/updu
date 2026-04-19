<script lang="ts">
    import { Check, Copy } from "lucide-svelte";
    import { toastStore } from "$lib/stores/toast.svelte";

    let {
        value,
        label = "Copy to clipboard",
        successMessage = "Copied to clipboard",
        size = "sm",
        testId,
    }: {
        value: string;
        label?: string;
        successMessage?: string;
        size?: "xs" | "sm";
        testId?: string;
    } = $props();

    let copied = $state(false);
    let resetTimer: ReturnType<typeof setTimeout> | undefined;

    async function copy() {
        if (!value) return;
        try {
            await navigator.clipboard.writeText(value);
            copied = true;
            toastStore.success(successMessage);
            if (resetTimer) clearTimeout(resetTimer);
            resetTimer = setTimeout(() => {
                copied = false;
            }, 1500);
        } catch {
            toastStore.error("Unable to copy to clipboard");
        }
    }

    const iconSize = $derived(size === "xs" ? "size-3" : "size-3.5");
    const padding = $derived(size === "xs" ? "p-1" : "p-1.5");
</script>

<button
    type="button"
    class="{padding} hover:bg-surface-elevated rounded-md transition-colors text-text-muted hover:text-text disabled:opacity-50 disabled:cursor-not-allowed"
    onclick={copy}
    disabled={!value}
    title={label}
    aria-label={label}
    data-testid={testId}
>
    {#if copied}
        <Check class="{iconSize} text-success" />
    {:else}
        <Copy class={iconSize} />
    {/if}
</button>
