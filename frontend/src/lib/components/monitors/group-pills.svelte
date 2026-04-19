<script lang="ts">
	// Renders a list of group names as small pills with overflow handling.
	import { cn } from "$lib/utils";

	interface Props {
		groups?: string[] | null;
		max?: number;
		size?: "sm" | "md";
		emptyLabel?: string;
		class?: string;
	}

	let {
		groups = [],
		max = 3,
		size = "sm",
		emptyLabel = "—",
		class: className,
	}: Props = $props();

	const list = $derived(Array.isArray(groups) ? groups : []);
	const visible = $derived(list.slice(0, max));
	const overflow = $derived(Math.max(0, list.length - max));

	const padding = $derived(size === "sm" ? "px-1.5 py-0.5" : "px-2 py-1");
	const text = $derived(size === "sm" ? "text-[10px]" : "text-xs");
</script>

{#if list.length === 0}
	<span class={cn("text-text-subtle", text)}>{emptyLabel}</span>
{:else}
	<div
		class={cn("flex flex-wrap items-center gap-1", className)}
		title={list.join(", ")}
	>
		{#each visible as group (group)}
			<span
				class={cn(
					"inline-flex items-center rounded-md border border-border/60 bg-surface-elevated/60 font-medium text-text-muted",
					padding,
					text,
				)}
			>
				{group}
			</span>
		{/each}
		{#if overflow > 0}
			<span
				class={cn(
					"inline-flex items-center rounded-md border border-border/60 bg-surface/60 font-medium text-text-subtle",
					padding,
					text,
				)}
				aria-label="{overflow} more groups"
			>
				+{overflow}
			</span>
		{/if}
	</div>
{/if}
