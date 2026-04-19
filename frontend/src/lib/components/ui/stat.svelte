<script lang="ts">
    // Reusable stat tile. Replaces ad-hoc card-with-icon-and-number patterns.
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    type Tone = "neutral" | "primary" | "success" | "warning" | "danger";

    interface Props {
        label: string;
        value: string | number;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        icon?: any;
        tone?: Tone;
        trend?: { delta: number; label?: string };
        footer?: Snippet;
        class?: string;
    }

    let {
        label,
        value,
        icon: Icon,
        tone = "neutral",
        trend,
        footer,
        class: className,
    }: Props = $props();

    const toneStyles: Record<Tone, { bg: string; iconBg: string; iconColor: string; border: string }> = {
        neutral: {
            bg: "",
            iconBg: "bg-surface-elevated",
            iconColor: "text-text-muted",
            border: "border-border",
        },
        primary: {
            bg: "",
            iconBg: "bg-primary/10",
            iconColor: "text-primary",
            border: "border-primary/20",
        },
        success: {
            bg: "",
            iconBg: "bg-success/10",
            iconColor: "text-success",
            border: "border-success/20",
        },
        warning: {
            bg: "",
            iconBg: "bg-warning/10",
            iconColor: "text-warning",
            border: "border-warning/20",
        },
        danger: {
            bg: "",
            iconBg: "bg-danger/10",
            iconColor: "text-danger",
            border: "border-danger/20",
        },
    };

    const t = $derived(toneStyles[tone]);
</script>

<div class={cn("card p-4", t.border, className)}>
    <div class="flex items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
            <p
                class="text-[10px] font-semibold uppercase tracking-wider text-text-subtle"
            >
                {label}
            </p>
            <p
                class={cn(
                    "mt-2 font-mono text-2xl font-bold tabular-nums",
                    tone === "neutral" ? "text-text" : t.iconColor,
                )}
            >
                {value}
            </p>
            {#if trend}
                {@const positive = trend.delta > 0}
                {@const isZero = trend.delta === 0}
                <p
                    class={cn(
                        "mt-1.5 inline-flex items-center gap-1 text-[11px] font-medium",
                        isZero
                            ? "text-text-subtle"
                            : positive
                              ? "text-success"
                              : "text-danger",
                    )}
                >
                    <span aria-hidden="true">
                        {isZero ? "→" : positive ? "↑" : "↓"}
                    </span>
                    {Math.abs(trend.delta).toFixed(1)}%
                    {#if trend.label}
                        <span class="text-text-subtle">{trend.label}</span>
                    {/if}
                </p>
            {/if}
        </div>
        {#if Icon}
            <div
                class={cn(
                    "flex size-9 shrink-0 items-center justify-center rounded-xl",
                    t.iconBg,
                )}
            >
                <Icon class={cn("size-4", t.iconColor)} />
            </div>
        {/if}
    </div>
    {#if footer}
        <div class="mt-3 border-t border-border/60 pt-3">
            {@render footer()}
        </div>
    {/if}
</div>
