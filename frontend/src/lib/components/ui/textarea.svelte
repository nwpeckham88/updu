<script lang="ts">
    import { cn } from "$lib/utils";

    interface Props {
        id?: string;
        value?: string;
        placeholder?: string;
        rows?: number;
        disabled?: boolean;
        required?: boolean;
        error?: boolean;
        class?: string;
        [key: string]: any;
    }

    let {
        id,
        value = $bindable(""),
        placeholder,
        rows = 4,
        disabled = false,
        required = false,
        error = false,
        class: className,
        ...rest
    }: Props = $props();
</script>

<textarea
    {id}
    {placeholder}
    {rows}
    {required}
    {disabled}
    bind:value
    class={cn(
        "w-full resize-y rounded-md border border-border bg-background px-3 py-2 font-sans text-sm text-text outline-none transition-[border-color,box-shadow] duration-150 placeholder:text-text-subtle focus:border-primary focus:shadow-[0_0_0_3px_hsl(217_91%_60%/0.12)]",
        error &&
            "border-danger/50 focus:border-danger focus:shadow-[0_0_0_3px_hsl(0_84%_60%/0.12)]",
        disabled && "cursor-not-allowed opacity-50",
        className,
    )}
    aria-invalid={error ? "true" : undefined}
    {...rest}
></textarea>
