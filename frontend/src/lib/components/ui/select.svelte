<script lang="ts">
    // Select built on bits-ui Select primitive. Themed to match Input.
    import { Select } from "bits-ui";
    import { Check, ChevronDown } from "lucide-svelte";
    import { cn } from "$lib/utils";

    export interface SelectOption {
        value: string;
        label: string;
        disabled?: boolean;
    }

    interface Props {
        id?: string;
        value?: string;
        options: SelectOption[];
        placeholder?: string;
        disabled?: boolean;
        error?: boolean;
        class?: string;
        onValueChange?: (value: string) => void;
    }

    let {
        id,
        value = $bindable(""),
        options,
        placeholder = "Select…",
        disabled = false,
        error = false,
        class: className,
        onValueChange,
    }: Props = $props();

    const selectedLabel = $derived(
        options.find((o) => o.value === value)?.label ?? "",
    );
</script>

<Select.Root
    type="single"
    bind:value
    {disabled}
    onValueChange={(v) => onValueChange?.(v)}
>
    <Select.Trigger
        {id}
        class={cn(
            "input-base flex items-center justify-between gap-2 text-left",
            error &&
                "border-danger/50 focus:border-danger focus:shadow-[0_0_0_3px_hsl(0_84%_60%/0.12)]",
            disabled && "cursor-not-allowed opacity-50",
            className,
        )}
        aria-invalid={error ? "true" : undefined}
    >
        <span class={cn("truncate", !selectedLabel && "text-text-subtle")}>
            {selectedLabel || placeholder}
        </span>
        <ChevronDown class="size-4 shrink-0 text-text-subtle" />
    </Select.Trigger>
    <Select.Portal>
        <Select.Content
            class="z-[var(--z-dropdown)] max-h-72 w-[var(--bits-select-anchor-width)] min-w-[8rem] overflow-y-auto rounded-md border border-border bg-surface-elevated p-1 shadow-[var(--shadow-card)] backdrop-blur-xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
            sideOffset={4}
        >
            {#each options as option (option.value)}
                <Select.Item
                    value={option.value}
                    label={option.label}
                    disabled={option.disabled}
                    class="relative flex cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-text outline-none transition-colors data-[highlighted]:bg-primary/10 data-[highlighted]:text-primary data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50"
                >
                    {#snippet children({ selected })}
                        <span class="flex size-4 shrink-0 items-center justify-center">
                            {#if selected}
                                <Check class="size-3.5" />
                            {/if}
                        </span>
                        <span class="truncate">{option.label}</span>
                    {/snippet}
                </Select.Item>
            {/each}
        </Select.Content>
    </Select.Portal>
</Select.Root>
