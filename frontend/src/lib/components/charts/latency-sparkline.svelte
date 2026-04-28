<script lang="ts">
    import { cn } from "$lib/utils";

    interface LatencyCheck {
        latency_ms?: number | null;
        checked_at?: string | null;
    }

    interface Props {
        checks?: LatencyCheck[] | null;
        fallbackLatency?: number | null;
        label?: string;
        class?: string;
    }

    let {
        checks = [],
        fallbackLatency = null,
        label = "Latency trend",
        class: className,
    }: Props = $props();

    const series = $derived.by(() => {
        const values = (checks ?? [])
            .filter((check) => check.latency_ms != null)
            .toReversed()
            .map((check) => Number(check.latency_ms))
            .filter((value) => Number.isFinite(value));

        if (values.length === 0 && fallbackLatency != null) {
            return [fallbackLatency];
        }
        return values;
    });

    const latest = $derived(series.length > 0 ? series[series.length - 1] : null);
    const average = $derived.by(() => {
        if (series.length === 0) return null;
        const total = series.reduce((sum, value) => sum + value, 0);
        return Math.round(total / series.length);
    });

    const points = $derived.by(() => {
        if (series.length === 0) return "";
        if (series.length === 1) return "0,14 100,14";

        const min = Math.min(...series);
        const max = Math.max(...series);
        const range = Math.max(max - min, 1);
        const lastIndex = series.length - 1;

        return series
            .map((value, index) => {
                const x = (index / lastIndex) * 100;
                const y = 24 - ((value - min) / range) * 20;
                return `${x.toFixed(2)},${y.toFixed(2)}`;
            })
            .join(" ");
    });

    const latestPoint = $derived.by(() => {
        if (series.length === 0) return null;
        if (series.length === 1) return { x: 100, y: 14 };

        const min = Math.min(...series);
        const max = Math.max(...series);
        const range = Math.max(max - min, 1);
        const value = series[series.length - 1];
        return { x: 100, y: 24 - ((value - min) / range) * 20 };
    });

    const toneClass = $derived.by(() => {
        if (latest == null) return "text-text-subtle";
        if (latest > 3000) return "text-danger";
        if (latest > 1000) return "text-warning";
        return "text-success";
    });

    const ariaLabel = $derived.by(() => {
        if (latest == null || average == null) return `${label}: no latency data`;
        return `${label}: latest ${latest}ms, average ${average}ms across ${series.length} checks`;
    });
</script>

<div
    class={cn("h-8 w-24 shrink-0", toneClass, className)}
    role="img"
    aria-label={ariaLabel}
>
    {#if series.length === 0}
        <div class="flex h-full items-center rounded-md border border-dashed border-border-subtle px-2">
            <span class="h-px w-full bg-border-subtle"></span>
        </div>
    {:else}
        <svg
            viewBox="0 0 100 28"
            preserveAspectRatio="none"
            class="h-full w-full overflow-visible"
            aria-hidden="true"
        >
            <line
                x1="0"
                y1="24"
                x2="100"
                y2="24"
                class="stroke-current opacity-20"
                stroke-width="1"
            />
            <polyline
                points={points}
                fill="none"
                class="stroke-current"
                stroke-width="2.4"
                stroke-linecap="round"
                stroke-linejoin="round"
                vector-effect="non-scaling-stroke"
            />
            {#if latestPoint}
                <circle
                    cx={latestPoint.x}
                    cy={latestPoint.y}
                    r="2.4"
                    class="fill-current"
                    vector-effect="non-scaling-stroke"
                />
            {/if}
        </svg>
    {/if}
</div>