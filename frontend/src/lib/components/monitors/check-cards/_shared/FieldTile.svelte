<script lang="ts">
    import type { Snippet } from "svelte";
    import CopyButton from "./CopyButton.svelte";

    let {
        label,
        value,
        href,
        monospace = false,
        multiline = false,
        copyable = false,
        copyValue,
        copyLabel,
        testId,
        size = "md",
        tone = "default",
        children,
        trailing,
    }: {
        label: string;
        value?: string | number | null;
        href?: string;
        monospace?: boolean;
        multiline?: boolean;
        copyable?: boolean;
        copyValue?: string;
        copyLabel?: string;
        testId?: string;
        size?: "sm" | "md" | "lg";
        tone?: "default" | "primary" | "success" | "warning" | "danger";
        children?: Snippet;
        trailing?: Snippet;
    } = $props();

    const display = $derived(
        value === undefined || value === null ? "" : `${value}`,
    );
    const showValue = $derived(display.length > 0);
    const copyText = $derived(copyValue ?? display);
    const valueClass = $derived(
        [
            monospace ? "font-mono text-xs" : "text-sm font-medium",
            multiline ? "whitespace-pre-wrap" : "",
            "break-all",
        ]
            .filter(Boolean)
            .join(" "),
    );

    const containerPadding = $derived(
        size === "sm" ? "p-2.5" : size === "lg" ? "p-4" : "p-3",
    );
    const toneClasses: Record<string, string> = {
        default: "border-border/70 bg-background/60",
        primary: "border-primary/30 bg-primary/5",
        success: "border-success/30 bg-success/5",
        warning: "border-warning/30 bg-warning/5",
        danger: "border-danger/30 bg-danger/5",
    };
</script>

<div
    class="rounded-xl border {toneClasses[tone]} {containerPadding} {multiline
        ? 'sm:col-span-2'
        : ''}"
    data-testid={testId}
>
    <div class="flex items-start justify-between gap-2">
        <p
            class="text-[10px] font-semibold uppercase tracking-[0.16em] text-text-subtle"
        >
            {label}
        </p>
        {#if copyable && copyText}
            <CopyButton
                value={copyText}
                label={copyLabel ?? `Copy ${label}`}
                size="xs"
            />
        {:else if trailing}
            {@render trailing()}
        {/if}
    </div>

    {#if children}
        <div class="mt-1">
            {@render children()}
        </div>
    {:else if showValue}
        {#if href}
            <a
                {href}
                target="_blank"
                rel="noopener noreferrer"
                class="mt-1 block text-primary hover:underline {valueClass}"
            >
                {display}
            </a>
        {:else if multiline}
            <code class="mt-1 block text-text {valueClass}">{display}</code>
        {:else}
            <p class="mt-1 text-text {valueClass}">
                {display}
            </p>
        {/if}
    {:else}
        <p class="mt-1 text-sm text-text-subtle italic">—</p>
    {/if}
</div>
