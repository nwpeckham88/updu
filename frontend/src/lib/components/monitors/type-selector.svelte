<script lang="ts">
	// Grid of monitor-type options rendered as icon tiles. Used by MonitorForm.
	import { cn } from "$lib/utils";
	import type { Component } from "svelte";

	export interface TypeOption {
		value: string;
		label: string;
		desc?: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: any;
	}

	interface Props {
		value: string;
		options: TypeOption[];
		onchange: (value: string) => void;
		columns?: number;
		class?: string;
	}

	let {
		value,
		options,
		onchange,
		columns = 5,
		class: className,
	}: Props = $props();

	const gridCols = $derived(
		{
			3: "grid-cols-3",
			4: "grid-cols-4",
			5: "grid-cols-5",
			6: "grid-cols-6",
		}[columns] ?? "grid-cols-5",
	);
</script>

<div
	class={cn("grid gap-2", gridCols, className)}
	role="radiogroup"
	aria-label="Monitor type"
>
	{#each options as opt (opt.value)}
		{@const active = value === opt.value}
		<button
			type="button"
			role="radio"
			aria-checked={active}
			onclick={() => onchange(opt.value)}
			class={cn(
				"flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
				active
					? "border-primary/50 bg-primary/10 text-primary shadow-[0_0_16px_hsl(217_91%_60%/0.1)]"
					: "border-border bg-surface-elevated/50 text-text-muted hover:border-border hover:text-text",
			)}
		>
			<opt.icon class="size-5" />
			<span class="text-[11px] font-bold uppercase tracking-wider">
				{opt.label}
			</span>
			{#if opt.desc}
				<span class="text-[10px] opacity-60">{opt.desc}</span>
			{/if}
		</button>
	{/each}
</div>
