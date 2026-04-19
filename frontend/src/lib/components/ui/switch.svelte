<script lang="ts">
    // Toggle switch built on bits-ui Switch primitive.
    import { Switch } from "bits-ui";
    import { cn } from "$lib/utils";

    interface Props {
        id?: string;
        checked?: boolean;
        disabled?: boolean;
        label?: string;
        description?: string;
        class?: string;
    }

    let {
        id,
        checked = $bindable(false),
        disabled = false,
        label,
        description,
        class: className,
    }: Props = $props();
</script>

<label
    for={id}
    class={cn(
        "flex cursor-pointer items-start gap-3",
        disabled && "cursor-not-allowed opacity-60",
        className,
    )}
>
    <Switch.Root
        {id}
        bind:checked
        {disabled}
        class="relative mt-0.5 inline-flex h-5 w-9 shrink-0 items-center rounded-full border border-border bg-surface transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background data-[state=checked]:border-primary data-[state=checked]:bg-primary"
    >
        <Switch.Thumb
            class="pointer-events-none block size-3.5 translate-x-0.5 rounded-full bg-text-muted shadow-sm transition-transform duration-150 data-[state=checked]:translate-x-[18px] data-[state=checked]:bg-white"
        />
    </Switch.Root>
    {#if label || description}
        <div class="min-w-0 flex-1">
            {#if label}
                <p class="text-sm font-medium text-text">{label}</p>
            {/if}
            {#if description}
                <p class="mt-0.5 text-xs text-text-muted">{description}</p>
            {/if}
        </div>
    {/if}
</label>
