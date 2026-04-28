<script lang="ts">
    import { format } from "date-fns";
    import { cn } from "$lib/utils";
    import { statusLabel, statusPattern } from "$lib/monitor-tones";

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
        minBuckets?: number;
        growThreshold?: number;
    }

    let {
        buckets,
        leftLabel = "Older",
        rightLabel = "Now",
        height = "md",
        class: className,
        minBuckets = 14,
        growThreshold = 0.3,
    }: Props = $props();

    const heightClass = $derived(
        { sm: "h-6", md: "h-9", lg: "h-12" }[height],
    );

    let focusedIndex = $state(-1);
    let containerEl: HTMLDivElement | null = $state(null);
    let liveText = $state("");

    function bucketColor(bucket: UptimeBucket | null): string {
        if (!bucket || !bucket.status) return "bg-border/30";
        switch (bucket.status) {
            case "up":
                return "bg-success/70 hover:bg-success focus-visible:bg-success";
            case "down":
                return "bg-danger/80 hover:bg-danger focus-visible:bg-danger";
            case "degraded":
                return "bg-warning/70 hover:bg-warning focus-visible:bg-warning";
            default:
                return "bg-warning/60 hover:bg-warning/80";
        }
    }

    function bucketPattern(bucket: UptimeBucket | null): string {
        if (!bucket || !bucket.status) return "";
        const pattern = statusPattern(bucket.status);
        if (pattern === "diagonal") return "ribbon-pattern-diagonal";
        if (pattern === "dotted") return "ribbon-pattern-dotted";
        return "";
    }

    function bucketTitle(bucket: UptimeBucket | null): string {
        if (!bucket) return "No data";
        const parts: string[] = [];
        if (bucket.status) parts.push(statusLabel(bucket.status));
        if (bucket.latency_ms != null) parts.push(`${bucket.latency_ms}ms`);
        if (bucket.checked_at) {
            try {
                parts.push(format(new Date(bucket.checked_at), "MMM d, HH:mm:ss"));
            } catch {
                /* ignore invalid timestamps */
            }
        }
        return parts.join(" | ");
    }

    const total = $derived(buckets.length);
    const filledBuckets = $derived.by(() => buckets.filter(Boolean) as UptimeBucket[]);
    const summary = $derived.by(() => {
        const up = filledBuckets.filter((bucket) => bucket.status === "up").length;
        const down = filledBuckets.filter((bucket) => bucket.status === "down").length;
        const degraded = filledBuckets.filter((bucket) => bucket.status === "degraded").length;
        return { up, down, degraded, total: filledBuckets.length };
    });

    const collecting = $derived(
        total > minBuckets && summary.total / total < growThreshold,
    );
    const visibleBuckets = $derived(collecting ? filledBuckets : buckets);

    function focusBucket(index: number) {
        if (!containerEl) return;
        const bars = containerEl.querySelectorAll("button[data-bucket]");
        const target = bars[index] as HTMLButtonElement | undefined;
        if (!target) return;

        target.focus();
        focusedIndex = index;
        liveText = bucketTitle(visibleBuckets[index]);
    }

    function handleKey(event: KeyboardEvent, index: number) {
        const lastIndex = visibleBuckets.length - 1;
        switch (event.key) {
            case "ArrowRight":
                event.preventDefault();
                focusBucket(Math.min(index + 1, lastIndex));
                break;
            case "ArrowLeft":
                event.preventDefault();
                focusBucket(Math.max(index - 1, 0));
                break;
            case "Home":
                event.preventDefault();
                focusBucket(0);
                break;
            case "End":
                event.preventDefault();
                focusBucket(lastIndex);
                break;
        }
    }

    function tabIndexFor(index: number): number {
        if (focusedIndex >= 0) return index === focusedIndex ? 0 : -1;
        return index === visibleBuckets.length - 1 ? 0 : -1;
    }
</script>

<div class={cn("space-y-1.5", className)}>
    <div
        bind:this={containerEl}
        class={cn("flex items-end gap-[2px]", heightClass)}
        role="group"
        aria-label="Uptime history: {summary.up} up, {summary.down} down, {summary.degraded} degraded out of {summary.total} checks"
    >
        {#if collecting}
            <div
                class="h-full flex-1 rounded-sm border border-dashed border-border-subtle bg-surface/40 flex items-center justify-center px-2"
                aria-hidden="true"
            >
                <span class="type-kicker text-text-subtle">
                    Collecting data...
                </span>
            </div>
        {/if}

        {#each visibleBuckets as bucket, index (bucket?.id ?? bucket?.checked_at ?? `slot-${index}`)}
            <button
                type="button"
                data-bucket
                tabindex={tabIndexFor(index)}
                onkeydown={(event) => handleKey(event, index)}
                onfocus={() => {
                    focusedIndex = index;
                    liveText = bucketTitle(bucket);
                }}
                class={cn(
                    "h-full rounded-sm transition-colors duration-150",
                    collecting ? "w-2" : "flex-1",
                    "focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1 focus-visible:ring-offset-background",
                    bucketColor(bucket),
                    bucketPattern(bucket),
                )}
                title={bucketTitle(bucket)}
                aria-label={bucketTitle(bucket)}
            ></button>
        {/each}
    </div>
    <div class="type-micro flex items-center justify-between text-text-subtle">
        <span>{leftLabel}</span>
        <span class="type-numeric">{total} buckets</span>
        <span>{rightLabel}</span>
    </div>
    <div class="sr-only" aria-live="polite" aria-atomic="true">{liveText}</div>
</div>

<style>
    :global(.ribbon-pattern-diagonal) {
        background-image: repeating-linear-gradient(
            45deg,
            transparent 0,
            transparent 2px,
            rgba(0, 0, 0, 0.35) 2px,
            rgba(0, 0, 0, 0.35) 4px
        );
    }

    :global(.ribbon-pattern-dotted) {
        background-image: radial-gradient(
            rgba(0, 0, 0, 0.35) 1px,
            transparent 1px
        );
        background-size: 4px 4px;
    }
</style>
