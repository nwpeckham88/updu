<script lang="ts">
    import { ExternalLink, Globe } from "lucide-svelte";
    import CopyButton from "./CopyButton.svelte";

    let {
        method,
        url,
        headline = "Request",
        tone = "primary",
    }: {
        method: string;
        url: string | undefined;
        headline?: string;
        tone?: "primary" | "default";
    } = $props();

    const display = $derived(url ? `${method.toUpperCase()} ${url}` : method);
    const toneClasses = $derived(
        tone === "primary"
            ? "border-primary/30 bg-primary/5 text-primary"
            : "border-border/70 bg-background/60 text-text",
    );
</script>

<div class="rounded-2xl border {toneClasses} p-4 sm:p-5 space-y-3">
    <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
            <Globe class="size-4" />
            <p
                class="text-[11px] font-semibold uppercase tracking-[0.18em]"
            >
                {headline}
            </p>
        </div>
        <div class="flex items-center gap-1">
            {#if url}
                <a
                    href={url}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="p-1 hover:bg-surface-elevated rounded-md transition-colors text-text-muted hover:text-text"
                    title="Open URL in new tab"
                    aria-label="Open URL in new tab"
                >
                    <ExternalLink class="size-3.5" />
                </a>
                <CopyButton
                    value={url}
                    label="Copy URL"
                    successMessage="URL copied"
                    size="xs"
                />
            {/if}
        </div>
    </div>
    <code
        class="block break-all rounded-lg bg-background/70 px-3 py-2 font-mono text-xs"
    >
        {display}
    </code>
</div>
