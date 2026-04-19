<script lang="ts">
    // Uptime ribbon: visualizes a fixed-width history of check buckets.
    // Each bucket maps to up/down/degraded/unknown. Tooltip exposes timestamp + latency.
    import { format } from "date-fns";
    import { cn } from "$lib/utils";

    export interface UptimeBucket {
        status?: string | null;
        latency_ms?: number | null;
        checked_at?: string | null;
        id?: string;
    }

    interface Props {
        buckets: (UptimeBucket | null)[];
        leftLabel?: string;
        rightLabel?: string;
        height?: "sm" | "md" | "lg";
        class?: string;
    }

    let {
        buckets,
        leftLabel = "Older",
        rightLabel = "Now",
        height = "md",
        class: className,
    }: Props = $props();

    const heightClass = $derived(
        { sm: "h-6", md: "h-9", lg: "h-12" }[height],
    );

    function bucketColor(b: UptimeBucket | null): string {
        if (!b || !b.status) return "bg-border/30";
        switch (b.status) {
            case "up":
                return "bg-success/70 hover:bg-success";
            case "down":
                return "bg-danger/80 hover:bg-danger";
            case "degraded":
                return "bg-warning/70 hover:bg-warning";
            default:
                return "bg-warning/60 hover:bg-warning/80";
        }
    }

    function bucketTitle(b: UptimeBucket | null): string {
        if (!b) return "No data";
        const parts: string[] = [];
        if (b.status) parts.push(b.status.toUpperCase());
        if (b.latency_ms != null) parts.push(`${b.latency_ms}ms`);
        if (b.checked_at) {
            try {
                parts.push(format(new Date(b.checked_at), "MMM d, HH:mm:ss"));
            } catch {
                /* ignore */
            }
        }
        return parts.join(" · ");
    }

    const total = $derived(buckets.length);
    const summary = $derived.by(() => {
        const filled = buckets.filter((b): b is UptimeBucket => !!b);
        const up = filled.filter((b) => b.status === "up").length;
        const down = filled.filter((b) => b.status === "down").length;
        const degraded = filled.filter((b) => b.status === "degraded").length;
        return { up, down, degraded, total: filled.length };
    });
</script>

<div class={cn("space-y-1.5", className)}>
    <div
        class={cn("flex items-end gap-[2px]", heightClass)}
        role="img"
        aria-label="Uptime history: {summary.up} up, {summary.down} down, {summary.degraded} degraded out of {summary.total} checks"
    >
        {#each buckets as bucket, i (bucket?.id ?? bucket?.checked_at ?? `slot-${i}`)}
            <div
                class={cn(
                    "h-full flex-1 rounded-sm transition-colors duration-150",
                    bucketColor(bucket),
                )}
                title={bucketTitle(bucket)}
            ></div>
        {/each}
    </div>
    <div class="flex items-center justify-between text-[10px] text-text-subtle">
        <span>{leftLabel}</span>
        <span class="font-mono">{total} buckets</span>
        <span>{rightLabel}</span>
    </div>
</div>
