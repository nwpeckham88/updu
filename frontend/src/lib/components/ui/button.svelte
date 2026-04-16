<script lang="ts">
    import { cn } from "$lib/utils";
    import type {
        HTMLButtonAttributes,
        HTMLAnchorAttributes,
    } from "svelte/elements";
    import type { Snippet } from "svelte";

    type Props = {
        variant?:
            | "default"
            | "destructive"
            | "outline"
            | "secondary"
            | "ghost"
            | "link";
        size?: "default" | "sm" | "lg" | "icon";
        loading?: boolean;
        href?: string;
        class?: string;
        disabled?: boolean;
        type?: "button" | "submit" | "reset";
        children?: Snippet;
        onclick?: ((e: MouseEvent) => void) | null;
        [key: string]: any;
    };

    let {
        class: className,
        variant = "default",
        size = "default",
        loading = false,
        href,
        children,
        disabled,
        type: btnType = "button",
        onclick,
        ...restProps
    }: Props = $props();

    const variants = {
        default:
            "bg-primary text-white hover:bg-primary/90 shadow-sm shadow-primary/20",
        destructive:
            "bg-danger text-white hover:bg-danger/90 shadow-sm shadow-danger/20",
        outline:
            "border border-border bg-transparent hover:bg-surface hover:border-border text-text",
        secondary:
            "bg-surface text-text hover:bg-surface-elevated border border-border",
        ghost: "hover:bg-surface text-text-muted hover:text-text",
        link: "text-primary underline-offset-4 hover:underline p-0 h-auto",
    };

    const sizes = {
        default: "h-10 px-4 py-2 text-sm",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-11 rounded-md px-6 text-sm",
        icon: "h-10 w-10",
    };

    const base =
        "inline-flex items-center justify-center gap-2 rounded-lg font-medium tracking-wide transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-40 cursor-pointer select-none";
</script>

{#snippet spinnerIcon()}
    <svg class="size-4 animate-spin" viewBox="0 0 24 24" fill="none">
        <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
        />
        <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
    </svg>
{/snippet}

{#if href}
    <a
        {href}
        {onclick}
        class={cn(base, variants[variant], sizes[size], className)}
    >
        {#if loading}{@render spinnerIcon()}{/if}
        {@render children?.()}
    </a>
{:else}
    <button
        type={btnType}
        class={cn(base, variants[variant], sizes[size], className)}
        disabled={disabled || loading}
        {onclick}
        {...restProps}
    >
        {#if loading}{@render spinnerIcon()}{/if}
        {@render children?.()}
    </button>
{/if}
