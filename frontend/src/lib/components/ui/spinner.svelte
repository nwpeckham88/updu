<script lang="ts">
    import { cn } from "$lib/utils";

    interface Props {
        class?: string;
        size?: "sm" | "md" | "lg";
    }

    let { class: className, size = "md" }: Props = $props();

    const sizes = {
        sm: "size-4 border-2",
        md: "size-6 border-2",
        lg: "size-8 border-2",
    };
</script>

<svg
    class={cn("animate-spin text-primary", sizes[size], className)}
    viewBox="0 0 24 24"
    fill="none"
    aria-label="Loading"
>
    <circle
        class="opacity-20"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
    />
    <path
        class="opacity-80"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
    />
</svg>
