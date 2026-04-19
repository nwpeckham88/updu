<script lang="ts">
    // Status donut: shows a single-percentage value as a ring.
    // Color shifts based on health thresholds.
    import { cn } from "$lib/utils";

    interface Props {
        value: number; // 0-100
        size?: "sm" | "md" | "lg";
        thickness?: number;
        label?: string;
        sublabel?: string;
        thresholds?: { good: number; warn: number };
        class?: string;
    }

    let {
        value,
        size = "md",
        thickness = 10,
        label,
        sublabel,
        thresholds = { good: 99, warn: 95 },
        class: className,
    }: Props = $props();

    const dimensions = $derived(
        { sm: 96, md: 140, lg: 200 }[size],
    );

    const radius = $derived((dimensions - thickness) / 2);
    const circumference = $derived(2 * Math.PI * radius);
    const clampedValue = $derived(Math.max(0, Math.min(100, value)));
    const dashOffset = $derived(
        circumference - (clampedValue / 100) * circumference,
    );

    const ringColor = $derived(
        clampedValue >= thresholds.good
            ? "var(--color-success)"
            : clampedValue >= thresholds.warn
              ? "var(--color-warning)"
              : "var(--color-danger)",
    );

    const valueClass = $derived(
        size === "sm"
            ? "text-base font-bold"
            : size === "md"
              ? "text-2xl font-bold"
              : "text-4xl font-bold",
    );
</script>

<div
    class={cn("relative inline-flex items-center justify-center", className)}
    style="width: {dimensions}px; height: {dimensions}px;"
    role="img"
    aria-label="{label ?? 'Uptime'}: {clampedValue.toFixed(2)}%"
>
    <svg
        width={dimensions}
        height={dimensions}
        viewBox="0 0 {dimensions} {dimensions}"
        class="-rotate-90"
        aria-hidden="true"
    >
        <circle
            cx={dimensions / 2}
            cy={dimensions / 2}
            r={radius}
            fill="none"
            stroke="var(--color-border)"
            stroke-width={thickness}
            stroke-opacity="0.4"
        />
        <circle
            cx={dimensions / 2}
            cy={dimensions / 2}
            r={radius}
            fill="none"
            stroke={ringColor}
            stroke-width={thickness}
            stroke-linecap="round"
            stroke-dasharray={circumference}
            stroke-dashoffset={dashOffset}
            style="transition: stroke-dashoffset 600ms var(--ease-out-expo);"
        />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
        <span class={cn(valueClass, "font-mono tabular-nums text-text")}>
            {clampedValue.toFixed(clampedValue === 100 ? 0 : 1)}<span
                class="text-text-subtle">%</span
            >
        </span>
        {#if sublabel}
            <span class="mt-0.5 text-[10px] uppercase tracking-wider text-text-subtle">
                {sublabel}
            </span>
        {/if}
    </div>
</div>
