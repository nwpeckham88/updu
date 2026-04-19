<script lang="ts">
	// Single event row, used by both detail page (recent) and events sub-page.
	import Badge from "$lib/components/ui/badge.svelte";
	import { Clock } from "lucide-svelte";
	import { formatDistanceToNow, format } from "date-fns";
	import { statusTextClass } from "$lib/monitor-tones";

	interface MonitorEvent {
		id?: string;
		status: string;
		message?: string;
		created_at: string;
	}

	interface Props {
		event: MonitorEvent;
	}

	let { event }: Props = $props();

	const fullTimestamp = $derived(
		(() => {
			try {
				return format(new Date(event.created_at), "PPpp");
			} catch {
				return event.created_at;
			}
		})(),
	);
	const relativeTimestamp = $derived(
		(() => {
			try {
				return formatDistanceToNow(new Date(event.created_at), {
					addSuffix: true,
				});
			} catch {
				return "";
			}
		})(),
	);
</script>

<div
	class="flex flex-col gap-3 p-4 transition-colors hover:bg-surface/30 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
>
	<div class="flex min-w-0 items-start gap-3">
		<div class="mt-0.5 shrink-0">
			<Badge status={event.status} size="sm" />
		</div>
		<div class="min-w-0">
			<p class="text-sm font-medium text-text">
				Status changed to
				<span class={statusTextClass(event.status)}>{event.status}</span>
			</p>
			{#if event.message}
				<p class="mt-0.5 line-clamp-2 text-xs text-text-muted">
					{event.message}
				</p>
			{/if}
		</div>
	</div>
	<div
		class="flex shrink-0 items-center gap-1.5 text-xs text-text-subtle sm:justify-end"
	>
		<Clock class="size-3" />
		<time datetime={event.created_at} title={fullTimestamp}>
			{relativeTimestamp}
		</time>
	</div>
</div>
