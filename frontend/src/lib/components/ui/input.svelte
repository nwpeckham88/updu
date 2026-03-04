<script lang="ts">
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";

    interface Props {
        class?: string;
        label?: string;
        id?: string;
        placeholder?: string;
        value?: string | number;
        type?: string;
        required?: boolean;
        disabled?: boolean;
        error?: string;
        hint?: string;
        icon?: Snippet;
        [key: string]: any;
    }

    let {
        class: className,
        label,
        id,
        placeholder,
        value = $bindable(""),
        type = "text",
        required = false,
        disabled = false,
        error,
        hint,
        icon,
        ...restProps
    }: Props = $props();
</script>

<div class={cn("flex flex-col gap-1.5", className)}>
    {#if label}
        <label for={id} class="text-sm font-medium text-text-muted">
            {label}
            {#if required}<span class="text-danger ml-0.5">*</span>{/if}
        </label>
    {/if}
    <div class="relative">
        {#if icon}
            <div
                class="absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle pointer-events-none"
            >
                {@render icon()}
            </div>
        {/if}
        <input
            {id}
            {type}
            {placeholder}
            {required}
            {disabled}
            bind:value
            class={cn(
                "input-base",
                icon ? "pl-9" : "",
                error
                    ? "border-danger/50 focus:border-danger focus:shadow-[0_0_0_3px_hsl(0_84%_60%/0.12)]"
                    : "",
                disabled ? "opacity-50 cursor-not-allowed" : "",
            )}
            aria-invalid={error ? "true" : undefined}
            aria-describedby={error
                ? `${id}-error`
                : hint
                  ? `${id}-hint`
                  : undefined}
            {...restProps}
        />
    </div>
    {#if error}
        <p id="{id}-error" class="text-xs text-danger">{error}</p>
    {:else if hint}
        <p id="{id}-hint" class="text-xs text-text-subtle">{hint}</p>
    {/if}
</div>
