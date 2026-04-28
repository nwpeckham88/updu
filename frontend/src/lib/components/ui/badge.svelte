<script lang="ts">
    import { cn } from "$lib/utils";
    import { statusIcon, statusLabel } from "$lib/monitor-tones";

    interface Props {
        class?: string;
        status?: "up" | "down" | "degraded" | "pending" | "paused" | string;
        dot?: boolean;
        /** Render a status icon glyph (shape encoding) alongside or in place of the dot. */
        icon?: boolean;
        calm?: boolean;
        size?: "sm" | "md";
        children?: import("svelte").Snippet;
    }

    let {
        class: className,
        status,
        dot = true,
        icon = true,
        calm = false,
        size = "sm",
        children,
    }: Props = $props();

    // Status visual config. The `down` state uses `motion-safe:animate-pulse` so users
    // who request reduced motion get a static, higher-contrast border instead of a
    // pulsing dot. See app.css `prefers-reduced-motion` block.
    const statusConfig: Record<
        string,
        { text: string; dot: string; bg: string; border: string }
    > = {
        up: {
            text: "text-success",
            dot: "bg-success shadow-[0_0_6px_hsl(142_71%_45%/0.7)]",
            bg: "bg-success/10",
            border: "border-success/20",
        },
        down: {
            text: "text-danger",
            dot: "bg-danger motion-safe:animate-pulse shadow-[0_0_6px_hsl(0_84%_60%/0.7)]",
            bg: "bg-danger/10",
            border: "border-danger/40 motion-reduce:border-2",
        },
        degraded: {
            text: "text-warning",
            dot: "bg-warning",
            bg: "bg-warning/10",
            border: "border-warning/20",
        },
        pending: {
            text: "text-text-subtle",
            dot: "bg-text-subtle",
            bg: "bg-surface",
            border: "border-border",
        },
        paused: {
            text: "text-text-subtle",
            dot: "bg-text-subtle",
            bg: "bg-surface",
            border: "border-border",
        },
    };

    const Glyph = $derived(status ? statusIcon(status) : null);
    const ariaLabel = $derived(status ? statusLabel(status) : undefined);

    const cfg = $derived.by(() => {
        if (!status) return null;
        const config = statusConfig[status] ?? statusConfig.pending;
        if (status === "down" && calm) {
            return {
                ...config,
                dot: "bg-danger/80 shadow-[0_0_3px_hsl(0_84%_60%/0.45)]",
                border: "border-danger/30 motion-reduce:border-2",
            };
        }
        return config;
    });

    const sizes = {
        sm: "px-2 py-0.5 text-[10px] gap-1.5",
        md: "px-2.5 py-1 text-xs gap-2",
    };
</script>

{#if cfg}
    <span
        class={cn(
            "inline-flex items-center rounded-full border font-semibold uppercase tracking-wider",
            cfg.text,
            cfg.bg,
            cfg.border,
            sizes[size],
            className,
        )}
        aria-label={ariaLabel}
    >
        {#if dot}
            <span class={cn("size-1.5 rounded-full shrink-0", cfg.dot)} aria-hidden="true"></span>
        {/if}
        {#if icon && Glyph}
            <Glyph class="size-3 shrink-0" aria-hidden="true" strokeWidth={3} />
        {/if}
        {#if children}
            {@render children()}
        {:else}
            {status}
        {/if}
    </span>
{:else}
    <span
        class={cn(
            "inline-flex items-center rounded-full border bg-surface border-border text-text-muted font-semibold uppercase tracking-wider",
            sizes[size],
            className,
        )}
    >
        {#if children}
            {@render children()}
        {/if}
    </span>
{/if}
