<script lang="ts">
    import { cn } from "$lib/utils";

    interface Props {
        class?: string;
        height?: string;
        width?: string;
        rounded?: string;
    }

    let {
        class: className,
        height = "h-4",
        width = "w-full",
        rounded = "rounded-md",
    }: Props = $props();
</script>

<div
    class={cn("animate-shimmer", height, width, rounded, className)}
    aria-hidden="true"
></div>
