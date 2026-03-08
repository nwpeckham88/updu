<script lang="ts">
    interface Props {
        /** Array of latency values (newest last). null/undefined = no data */
        data: (number | null | undefined)[];
        /** SVG width */
        width?: number;
        /** SVG height */
        height?: number;
        /** Line color */
        color?: string;
        /** Fill gradient color (bottom) */
        fillColor?: string;
        /** Whether the monitor is down */
        isDown?: boolean;
    }

    let {
        data,
        width = 200,
        height = 40,
        color = "hsl(142, 71%, 45%)",
        fillColor = "hsl(142, 71%, 45%)",
        isDown = false,
    }: Props = $props();

    const effectiveColor = $derived(isDown ? "hsl(0, 84%, 60%)" : color);
    const effectiveFill = $derived(isDown ? "hsl(0, 84%, 60%)" : fillColor);

    const gradientId = $derived(`sparkline-grad-${Math.random().toString(36).slice(2, 8)}`);

    const pathData = $derived.by(() => {
        const valid = data.filter((v): v is number => v != null && v >= 0);
        if (valid.length < 2) return { line: "", area: "", points: [] };

        const padding = 2;
        const w = width - padding * 2;
        const h = height - padding * 2;

        const max = Math.max(...valid, 1);
        const min = Math.min(...valid, 0);
        const range = max - min || 1;

        // Map data points to coordinates
        const points: [number, number][] = [];
        let dataIdx = 0;
        for (let i = 0; i < data.length; i++) {
            const v = data[i];
            if (v == null || v < 0) continue;
            const x = padding + (dataIdx / Math.max(valid.length - 1, 1)) * w;
            const y = padding + h - ((v - min) / range) * h;
            points.push([x, y]);
            dataIdx++;
        }

        if (points.length < 2) return { line: "", area: "", points: [] };

        // Build smooth curve using catmull-rom to bezier
        let line = `M ${points[0][0]},${points[0][1]}`;
        for (let i = 0; i < points.length - 1; i++) {
            const p0 = points[Math.max(0, i - 1)];
            const p1 = points[i];
            const p2 = points[i + 1];
            const p3 = points[Math.min(points.length - 1, i + 2)];

            const tension = 0.3;
            const cp1x = p1[0] + (p2[0] - p0[0]) * tension;
            const cp1y = p1[1] + (p2[1] - p0[1]) * tension;
            const cp2x = p2[0] - (p3[0] - p1[0]) * tension;
            const cp2y = p2[1] - (p3[1] - p1[1]) * tension;

            line += ` C ${cp1x},${cp1y} ${cp2x},${cp2y} ${p2[0]},${p2[1]}`;
        }

        // Area fill (close path to bottom)
        const area =
            line +
            ` L ${points[points.length - 1][0]},${height} L ${points[0][0]},${height} Z`;

        return { line, area, points };
    });
</script>

<svg
    {width}
    {height}
    viewBox="0 0 {width} {height}"
    class="overflow-visible"
    preserveAspectRatio="none"
>
    <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color={effectiveFill} stop-opacity="0.3" />
            <stop offset="100%" stop-color={effectiveFill} stop-opacity="0.02" />
        </linearGradient>
    </defs>

    {#if pathData.area}
        <!-- Fill area -->
        <path d={pathData.area} fill="url(#{gradientId})" />
        <!-- Line -->
        <path
            d={pathData.line}
            fill="none"
            stroke={effectiveColor}
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
        />
        <!-- End dot -->
        {#if pathData.points.length > 0}
            {@const last = pathData.points[pathData.points.length - 1]}
            <circle
                cx={last[0]}
                cy={last[1]}
                r="2"
                fill={effectiveColor}
            />
        {/if}
    {:else}
        <!-- No data line -->
        <line
            x1="2"
            y1={height / 2}
            x2={width - 2}
            y2={height / 2}
            stroke="var(--color-border)"
            stroke-width="1"
            stroke-dasharray="4 3"
        />
    {/if}
</svg>
