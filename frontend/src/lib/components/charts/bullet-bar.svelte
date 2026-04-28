<script lang="ts">
    import { cn } from "$lib/utils";

    interface Props {
        value?: number | null;
        target?: number;
        warning?: number;
        danger?: number;
        label?: string;
        unit?: string;
        class?: string;
    }

    let {
        value = null,
        target = 500,
        warning = 1000,
        danger = 3000,
        label = "Latency",
        unit = "ms",
        class: className,
    }: Props = $props();

    const domainMax = $derived(Math.max(danger, warning, target, value ?? 0, 1));
    const valuePercent = $derived(Math.min(((value ?? 0) / domainMax) * 100, 100));
    const targetPercent = $derived(Math.min((target / domainMax) * 100, 100));
    const warningPercent = $derived(Math.min((warning / domainMax) * 100, 100));
    const valueText = $derived(value != null ? `${value}${unit}` : "No data");

    const toneClass = $derived.by(() => {
        if (value == null) return "bg-text-subtle";
        if (value > warning) return "bg-danger";
        if (value > target) return "bg-warning";
        return "bg-success";
    });

    const valueTextClass = $derived.by(() => {
        if (value == null) return "text-text-subtle";
        if (value > warning) return "text-danger";
        if (value > target) return "text-warning";
        return "text-success";
    });

    const ariaLabel = $derived.by(() => {
        if (value == null) return `${label}: no latency data. Target ${target}${unit}.`;
        return `${label}: ${value}${unit}. Target ${target}${unit}; warning threshold ${warning}${unit}.`;
    });
</script>

<div class={cn("min-w-0 space-y-2", className)} role="img" aria-label={ariaLabel}>
    <div class="flex items-baseline justify-between gap-3">
        <p class="type-kicker truncate text-text-subtle">
            {label}
        </p>
        <p class={cn("type-numeric shrink-0 text-sm font-bold", valueTextClass)}>
            {valueText}
        </p>
    </div>

    <div
        class="relative h-3 overflow-hidden rounded-full border border-border/60 bg-danger/10"
        aria-hidden="true"
    >
        <div class="absolute inset-y-0 left-0 bg-success/20" style:width={`${targetPercent}%`}></div>
        <div
            class="absolute inset-y-0 bg-warning/20"
            style:left={`${targetPercent}%`}
            style:width={`${Math.max(warningPercent - targetPercent, 0)}%`}
        ></div>
        <div
            class={cn("absolute inset-y-[2px] left-0 rounded-r-full", toneClass)}
            style:width={`${valuePercent}%`}
        ></div>
        <div
            class="absolute inset-y-[-2px] w-px bg-text/70"
            style:left={`${targetPercent}%`}
        ></div>
    </div>

    <div class="type-numeric type-micro flex items-center justify-between text-text-subtle">
        <span>0</span>
        <span>target {target}{unit}</span>
        <span>{domainMax}{unit}</span>
    </div>
</div>