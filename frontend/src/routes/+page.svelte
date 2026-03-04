<script lang="ts">
	import {
		Activity,
		ServerCrash,
		CircleCheck,
		Clock,
		TrendingUp,
		ArrowUpRight,
		Wifi,
	} from "lucide-svelte";
	import { monitorsStore } from "$lib/stores/monitors.svelte";
	import { goto } from "$app/navigation";
	import { formatDistanceToNow } from "date-fns";
	import { settingsStore } from "$lib/stores/settings.svelte";
	import Skeleton from "$lib/components/ui/skeleton.svelte";
	import Badge from "$lib/components/ui/badge.svelte";
	import Card from "$lib/components/ui/card.svelte";
	import EmptyState from "$lib/components/ui/empty-state.svelte";
	import Button from "$lib/components/ui/button.svelte";

	$effect(() => {
		monitorsStore.init();
		return () => monitorsStore.destroy();
	});

	const monitors = $derived(monitorsStore.monitors);
	const loading = $derived(monitorsStore.loading);

	const upCount = $derived(
		monitors.filter((m) => m.status === "up" && m.enabled).length,
	);
	const downCount = $derived(
		monitors.filter((m) => m.status === "down").length,
	);
	const pausedCount = $derived(monitors.filter((m) => !m.enabled).length);
	const avgLatency = $derived(
		monitors.filter((m) => m.last_latency_ms != null).length > 0
			? Math.round(
					monitors
						.filter((m) => m.last_latency_ms != null)
						.reduce((s, m) => s + (m.last_latency_ms ?? 0), 0) /
						monitors.filter((m) => m.last_latency_ms != null)
							.length,
				) + "ms"
			: "—",
	);

	const overallHealth = $derived(
		monitors.length === 0
			? null
			: monitors.filter((m) => m.status === "up").length /
					monitors.length,
	);

	type StatCard = {
		label: string;
		value: string | number;
		icon: any;
		color: string;
		bg: string;
		border: string;
		highlight?: boolean;
	};

	const statCards = $derived<StatCard[]>([
		{
			label: "Total Monitors",
			value: loading ? "—" : monitors.length,
			icon: Activity,
			color: "text-primary",
			bg: "bg-primary/10",
			border: "border-primary/20",
		},
		{
			label: "Operational",
			value: loading ? "—" : upCount,
			icon: CircleCheck,
			color: "text-success",
			bg: "bg-success/10",
			border: "border-success/20",
		},
		{
			label: "Incidents",
			value: loading ? "—" : downCount,
			icon: ServerCrash,
			color: downCount > 0 ? "text-danger" : "text-text-subtle",
			bg: downCount > 0 ? "bg-danger/10" : "bg-surface",
			border: downCount > 0 ? "border-danger/30" : "border-border",
			highlight: downCount > 0,
		},
		{
			label: "Avg Latency",
			value: loading ? "—" : avgLatency,
			icon: Wifi,
			color: "text-text-muted",
			bg: "bg-surface",
			border: "border-border",
		},
	]);

	// Build heartbeat bars (newest = rightmost) from real data
	function buildHeartbeat(monitor: any): { status: string }[] {
		const bars: { status: string }[] = Array(40)
			.fill(null)
			.map(() => ({ status: "empty" }));
		const checks = monitor.recent_checks || [];
		// checks are newest-first; fill from right
		for (let i = 0; i < Math.min(checks.length, 40); i++) {
			bars[39 - i] = { status: checks[i].status };
		}
		return bars;
	}
</script>

<svelte:head>
	<title>Dashboard – updu</title>
</svelte:head>

<div class="space-y-6 max-w-7xl">
	<!-- Page header -->
	<div>
		<h1 class="text-2xl font-bold tracking-tight text-text">Dashboard</h1>
		<p class="text-sm text-text-muted mt-1">
			Real-time infrastructure overview
		</p>
	</div>

	<!-- Stat cards -->
	<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
		{#each statCards as card}
			<div
				class="card p-5 relative overflow-hidden {card.highlight
					? 'border-danger/30 bg-danger/5'
					: ''}"
			>
				{#if loading}
					<div class="space-y-3">
						<Skeleton height="h-3" width="w-24" />
						<Skeleton height="h-8" width="w-16" />
					</div>
				{:else}
					<div class="flex items-start justify-between">
						<div>
							<p
								class="text-xs font-medium text-text-subtle uppercase tracking-wider"
							>
								{card.label}
							</p>
							<p
								class="text-3xl font-bold mt-2 {card.highlight
									? 'text-danger'
									: 'text-text'}"
							>
								{card.value}
							</p>
						</div>
						<div
							class="size-9 rounded-xl {card.bg} border {card.border} flex items-center justify-center shrink-0"
						>
							<card.icon class="size-4 {card.color}" />
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>

	<!-- Overall health bar -->
	{#if !loading && monitors.length > 0 && overallHealth !== null}
		<div class="card p-4">
			<div class="flex items-center justify-between mb-2">
				<div class="flex items-center gap-2">
					<TrendingUp class="size-4 text-text-muted" />
					<span class="text-sm font-medium text-text-muted"
						>Overall Health</span
					>
				</div>
				<span
					class="text-sm font-bold {overallHealth > 0.9
						? 'text-success'
						: overallHealth > 0.7
							? 'text-warning'
							: 'text-danger'}"
				>
					{(overallHealth * 100).toFixed(1)}%
				</span>
			</div>
			<div class="h-1.5 bg-border rounded-full overflow-hidden">
				<div
					class="h-full rounded-full transition-all duration-500 {overallHealth >
					0.9
						? 'bg-success'
						: overallHealth > 0.7
							? 'bg-warning'
							: 'bg-danger'}"
					style="width: {overallHealth * 100}%"
				></div>
			</div>
		</div>
	{/if}

	<!-- Monitor grid -->
	<div>
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-base font-semibold text-text">All Monitors</h2>
			{#if !loading}
				<Button href="/monitors" variant="ghost" size="sm">
					View all <ArrowUpRight class="size-3.5" />
				</Button>
			{/if}
		</div>

		{#if loading}
			<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
				{#each { length: 6 } as _}
					<div class="card p-4 space-y-3">
						<div class="flex items-center gap-3">
							<Skeleton
								height="h-10"
								width="w-10"
								rounded="rounded-full"
							/>
							<div class="flex-1 space-y-2">
								<Skeleton height="h-4" width="w-3/4" />
								<Skeleton height="h-3" width="w-1/2" />
							</div>
						</div>
						<Skeleton
							height="h-4"
							width="w-full"
							rounded="rounded"
						/>
					</div>
				{/each}
			</div>
		{:else if monitors.length === 0}
			<div class="card">
				<EmptyState
					icon={Activity}
					title="No monitors yet"
					description="Start tracking your homelab services by creating your first monitor."
				>
					<Button href="/monitors" class="mt-2">Go to Monitors</Button
					>
				</EmptyState>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
				{#each monitors as monitor (monitor.id)}
					<button
						onclick={() => goto(`/monitors/${monitor.id}`)}
						class="card card-interactive text-left w-full {monitor.status ===
						'down'
							? 'border-danger/30 bg-danger/5'
							: ''} {settingsStore.get('dashboard_style') ===
						'compact'
							? 'p-3'
							: 'p-4'}"
					>
						<!-- Top row: icon + name + latency -->
						<div class="flex items-start gap-3">
							<div
								class="size-10 rounded-xl flex items-center justify-center shrink-0 {monitor.status ===
								'up'
									? 'bg-success/15 text-success'
									: monitor.status === 'down'
										? 'bg-danger/15 text-danger'
										: 'bg-border text-text-subtle'}"
							>
								{#if monitor.status === "up"}
									<CircleCheck class="size-5" />
								{:else if monitor.status === "down"}
									<ServerCrash class="size-5" />
								{:else}
									<Activity class="size-5" />
								{/if}
							</div>

							<div class="flex-1 min-w-0">
								<div
									class="flex items-center justify-between gap-2"
								>
									<h3
										class="text-sm font-semibold text-text truncate"
									>
										{monitor.name}
									</h3>
									<span
										class="text-xs font-mono text-text-subtle shrink-0"
									>
										{monitor.last_latency_ms != null
											? monitor.last_latency_ms + "ms"
											: "—"}
									</span>
								</div>

								<!-- Meta row: type badge + status + uptime + last check -->
								<div
									class="flex items-center gap-2 mt-1 flex-wrap"
								>
									<span
										class="text-[10px] px-1.5 py-0.5 rounded bg-surface-elevated border border-border text-text-muted uppercase tracking-wider font-semibold"
									>
										{monitor.type}
									</span>
									<Badge
										status={!monitor.enabled
											? "paused"
											: monitor.status}
									/>
									{#if monitor.uptime_24h != null}
										<span
											class="text-[10px] font-mono font-bold tabular-nums {monitor.uptime_24h >=
											99
												? 'text-success'
												: monitor.uptime_24h >= 95
													? 'text-warning'
													: 'text-danger'}"
										>
											{monitor.uptime_24h.toFixed(1)}%
										</span>
									{/if}
								</div>

								{#if monitor.last_check}
									<p
										class="text-[10px] text-text-subtle mt-1"
									>
										<Clock
											class="size-2.5 inline -mt-0.5"
										/>
										{formatDistanceToNow(
											new Date(monitor.last_check),
											{ addSuffix: true },
										)}
									</p>
								{/if}
							</div>
						</div>

						<!-- Real heartbeat bar -->
						{#if settingsStore.get("dashboard_show_heartbeat", "true") !== "false"}
							<div class="flex gap-[2px] mt-3 h-3.5 items-end">
								{#each buildHeartbeat(monitor) as bar}
									<div
										class="flex-1 rounded-full transition-colors {bar.status ===
										'up'
											? 'bg-success/60 hover:bg-success/90'
											: bar.status === 'down'
												? 'bg-danger/70 hover:bg-danger'
												: 'bg-border/40'}"
									></div>
								{/each}
							</div>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
