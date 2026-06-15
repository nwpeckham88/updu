<script lang="ts">
	// Tabbed grid of monitor-type options rendered as icon tiles. Used by MonitorForm.
	import { cn } from "$lib/utils";

	export interface TypeOption {
		value: string;
		label: string;
		desc?: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: any;
	}

	export interface TypeGroup {
		label: string;
		options: TypeOption[];
	}

	interface Props {
		value: string;
		groups: TypeGroup[];
		onchange: (value: string) => void;
		columns?: number;
		class?: string;
	}

	let {
		value,
		groups,
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

	// Find the initial tab based on the selected value
	let activeTab = $state(
		groups.find((g) => g.options.some((o) => o.value === value))?.label ??
			groups[0]?.label ??
			"",
	);

	const activeGroup = $derived(
		groups.find((g) => g.label === activeTab) ?? groups[0],
	);
</script>

<div class={cn("space-y-4", className)}>
	<!-- Tabs -->
	<div class="flex gap-2 border-b border-border/50 pb-px overflow-x-auto no-scrollbar">
		{#each groups as group}
			{@const active = activeTab === group.label}
			<button
				type="button"
				onclick={() => (activeTab = group.label)}
				class={cn(
					"px-3 py-2 text-[13px] font-medium transition-colors border-b-2 whitespace-nowrap",
					active
						? "border-primary text-primary"
						: "border-transparent text-text-muted hover:text-text hover:border-border"
				)}
			>
				{group.label}
			</button>
		{/each}
	</div>

	<!-- Grid -->
	<div
		class={cn("grid gap-2", gridCols)}
		role="radiogroup"
		aria-label="Monitor type"
	>
		{#each activeGroup?.options || [] as opt (opt.value)}
			{@const active = value === opt.value}
			<button
				type="button"
				role="radio"
				aria-checked={active}
				onclick={() => {
					onchange(opt.value);
				}}
				class={cn(
					"flex flex-col items-center justify-center gap-1.5 rounded-xl border p-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
					active
						? "border-primary/50 bg-primary/10 text-primary shadow-[0_0_16px_hsl(217_91%_60%/0.1)]"
						: "border-border bg-surface-elevated/50 text-text-muted hover:border-border hover:text-text",
				)}
			>
				<opt.icon class="size-5" />
				<span class="text-[11px] font-bold uppercase tracking-wider text-center leading-tight">
					{opt.label}
				</span>
				{#if opt.desc}
					<span class="text-[10px] opacity-60 text-center leading-tight">{opt.desc}</span>
				{/if}
			</button>
		{/each}
	</div>
</div>