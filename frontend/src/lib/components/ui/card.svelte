<script lang="ts">
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    interface Props {
        class?: string;
        variant?: "default" | "success" | "warning" | "danger" | "info";
        noPad?: boolean;
        children?: Snippet;
    }

    let {
        class: className,
        variant = "default",
        noPad = false,
        children,
    }: Props = $props();

    const variantStyles = {
        default: "",
        success: "border-success/20 bg-success/5",
        warning: "border-warning/20 bg-warning/5",
        danger: "border-danger/20 bg-danger/5",
        info: "border-primary/20 bg-primary/5",
    };
</script>

<div class={cn("card", variantStyles[variant], noPad ? "" : "p-5", className)}>
    {@render children?.()}
</div>
