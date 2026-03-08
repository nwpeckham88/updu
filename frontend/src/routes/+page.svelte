<script lang="ts">
	import {
		Activity,
		ServerCrash,
		CircleCheck,
		Clock,
		TrendingUp,
		ArrowUpRight,
		Wifi,
		BarChart3,
		EllipsisVertical,
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
	import Sparkline from "$lib/components/ui/sparkline.svelte";

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

	const avgLatencyNum = $derived(
		monitors.filter((m) => m.last_latency_ms != null).length > 0
			? Math.round(
					monitors
						.filter((m) => m.last_latency_ms != null)
						.reduce((s, m) => s + (m.last_latency_ms ?? 0), 0) /
						monitors.filter((m) => m.last_latency_ms != null)
							.length,
				)
			: null,
	);
	const avgLatency = $derived(
		avgLatencyNum != null ? avgLatencyNum + "ms" : "—",
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
		iconBg?: string;
	};

	const statCards = $derived<StatCard[]>([
		{
			label: "Total Monitors",
			value: loading ? "—" : monitors.length,
			icon: Activity,
			color: "text-primary",
			bg: "bg-primary/10",
			border: "border-primary/20",
			iconBg: "bg-primary/10 border-primary/20",
		},
		{
			label: "Operational",
			value: loading ? "—" : upCount,
			icon: CircleCheck,
			color: "text-success",
			bg: "bg-success/10",
			border: "border-success/20",
			iconBg: "bg-success/10 border-success/20",
		},
		{
			label: "Incidents",
			value: loading ? "—" : downCount,
			icon: ServerCrash,
			color: downCount > 0 ? "text-danger" : "text-text-subtle",
			bg: downCount > 0 ? "bg-danger/10" : "bg-surface",
			border: downCount > 0 ? "border-danger/30" : "border-border",
			highlight: downCount > 0,
			iconBg: downCount > 0 ? "bg-danger/10 border-danger/20" : "bg-surface border-border",
		},
		{
			label: "Avg Latency",
			value: loading ? "—" : avgLatency,
			icon: BarChart3,
			color: "text-text-muted",
			bg: "bg-surface",
			border: "border-border",
			iconBg: "bg-surface-elevated border-border",
		},
	]);

	// Build heartbeat bars (newest = rightmost) from real data
	function buildHeartbeat(
		monitor: any,
	): { status: string; latency?: number; time?: string }[] {
		const barCount = 30;
		const bars: { status: string; latency?: number; time?: string }[] =
			Array(barCount)
				.fill(null)
				.map(() => ({ status: "empty" }));
		const checks = monitor.recent_checks || [];
		// checks are newest-first; fill from right
		for (let i = 0; i < Math.min(checks.length, barCount); i++) {
			bars[barCount - 1 - i] = {
				status: checks[i].status,
				latency: checks[i].latency_ms,
				time: checks[i].checked_at,
			};
		}
		return bars;
	}

	// Extract latency values for sparkline (newest last)
	function getLatencyData(monitor: any): (number | null)[] {
		const checks = monitor.recent_checks || [];
		// checks array is newest-first, reverse to get oldest-first for sparkline
		return [...checks]
			.reverse()
			.map((c: any) => (c.latency_ms != null ? c.latency_ms : null));
	}

	// Compute the time range label for heartbeat bars
	function getTimeRangeLabel(monitor: any): string {
		const checks = monitor.recent_checks || [];
		if (checks.length < 2) return "";
		const newest = new Date(checks[0].checked_at).getTime();
		const oldest = new Date(checks[checks.length - 1].checked_at).getTime();
		const diffMin = Math.round((newest - oldest) / 60000);
		if (diffMin < 60) return `${diffMin} min`;
		return `${Math.round(diffMin / 60)}h`;
	}
</script>

<svelte:head>
	<title>Dashboard – updu</title>
</svelte:head>

<div class="space-y-5 max-w-7xl">
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
				class="card p-4 relative overflow-hidden {card.highlight
					? 'border-danger/30 bg-danger/5'
					: ''}"
			>
				{#if loading}
					<div class="space-y-3">
						<Skeleton height="h-3" width="w-24" />
						<Skeleton height="h-8" width="w-16" />
					</div>
				{:else}
					<div class="flex items-center justify-between">
						<div>
							<p
								class="text-[11px] font-medium text-text-subtle uppercase tracking-wider"
							>
								{card.label}
							</p>
							<div class="flex items-baseline gap-2 mt-1.5">
								<p
									class="text-2xl font-bold tabular-nums {card.highlight
										? 'text-danger'
										: 'text-text'}"
								>
									{card.value}
								</p>
							</div>
						</div>
						<div
							class="size-9 rounded-xl {card.iconBg ?? card.bg} border flex items-center justify-center shrink-0"
						>
							<card.icon class="size-4 {card.color}" />
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>

	<!-- Overall health bar (compact) -->
	{#if !loading && monitors.length > 0 && overallHealth !== null}
		<div class="card p-3">
			<div class="flex items-center justify-between mb-1.5">
				<div class="flex items-center gap-2">
					<TrendingUp class="size-3.5 text-text-muted" />
					<span class="text-xs font-medium text-text-muted"
						>Overall Health</span
					>
				</div>
				<span
					class="text-xs font-bold font-mono tabular-nums {overallHealth > 0.9
						? 'text-success'
						: overallHealth > 0.7
							? 'text-warning'
							: 'text-danger'}"
				>
					{(overallHealth * 100).toFixed(2)}%
				</span>
			</div>
			<div class="h-1 bg-border rounded-full overflow-hidden">
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
		<div class="flex items-center justify-between mb-3">
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
						<Skeleton height="h-4" width="w-3/4" />
						<Skeleton height="h-3" width="w-1/2" />
						<Skeleton height="h-10" width="w-full" rounded="rounded" />
						<Skeleton height="h-3" width="w-full" rounded="rounded" />
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
					<Button href="/monitors" class="mt-2">Go to Monitors</Button>
				</EmptyState>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
				{#each monitors as monitor (monitor.id)}
					{@const isDown = monitor.status === "down"}
					{@const isPaused = !monitor.enabled}
					{@const heartbeat = buildHeartbeat(monitor)}
					{@const latencyData = getLatencyData(monitor)}
					{@const timeRange = getTimeRangeLabel(monitor)}

					<button
						onclick={() => goto(`/monitors/${monitor.id}`)}
						class="card card-interactive text-left w-full p-0 flex flex-col {isDown
							? 'border-danger/30 bg-danger/5'
							: ''}"
					>
						<!-- Card header -->
						<div class="px-4 pt-3.5 pb-0">
							<!-- Top line: name + menu -->
							<div class="flex items-start justify-between gap-2">
								<h3 class="text-sm font-semibold text-text truncate leading-tight">
									{monitor.name}
								</h3>
								<EllipsisVertical class="size-4 text-text-subtle shrink-0 mt-0.5 opacity-40 hover:opacity-100 transition-opacity" />
							</div>

							<!-- Status + metrics row -->
							<div class="flex items-center justify-between mt-1.5 gap-3">
								<div class="flex items-center gap-1.5 min-w-0">
									{#if isDown}
										<span class="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-danger">
											<span class="size-1.5 rounded-full bg-danger animate-pulse shadow-[0_0_6px_hsl(0_84%_60%/0.7)]"></span>
											INCIDENT
										</span>
									{:else if isPaused}
										<span class="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-text-subtle">
											<span class="size-1.5 rounded-full bg-text-subtle"></span>
											PAUSED
										</span>
									{:else}
										<span class="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-success">
											<span class="size-1.5 rounded-full bg-success shadow-[0_0_6px_hsl(142_71%_45%/0.7)]"></span>
											OPERATIONAL
										</span>
									{/if}
								</div>

								<div class="flex items-center gap-3 shrink-0">
									{#if monitor.uptime_24h != null}
										<span
											class="text-[11px] font-mono font-bold tabular-nums {monitor.uptime_24h >= 99
												? 'text-success/80'
												: monitor.uptime_24h >= 95
													? 'text-warning/80'
													: 'text-danger/80'}"
										>
											{monitor.uptime_24h.toFixed(2)}% Uptime
										</span>
									{/if}
									{#if monitor.last_latency_ms != null}
										<span
											class="text-[11px] font-mono font-bold tabular-nums {isDown ? 'text-danger' : 'text-text'}"
										>
											{monitor.last_latency_ms}ms
										</span>
									{/if}
								</div>
							</div>

							{#if isDown}
								<div class="mt-2">
									<span class="inline-block px-2 py-0.5 text-[11px] font-bold text-danger bg-danger/15 border border-danger/20 rounded-md uppercase tracking-wider">
										DOWN
									</span>
								</div>
							{/if}
						</div>

						<!-- Sparkline chart -->
						<div class="px-4 pt-2 pb-0">
							<Sparkline
								data={latencyData}
								width={260}
								height={36}
								isDown={isDown}
							/>
						</div>

						<!-- Heartbeat bars + time labels -->
						{#if settingsStore.get("dashboard_show_heartbeat", "true") !== "false"}
							<div class="px-4 pt-1 pb-3">
								<div class="flex gap-[2px] h-4 items-end">
									{#each heartbeat as bar}
										<div
											class="flex-1 rounded-[2px] transition-colors {bar.status === 'up'
												? 'bg-success/70 hover:bg-success'
												: bar.status === 'down'
													? 'bg-danger/80 hover:bg-danger'
													: 'bg-border/30'}"
											style="height: {bar.status === 'empty' ? '30%' : '100%'}"
											title={bar.status === "empty"
												? "No data"
												: `${bar.status.toUpperCase()} · ${new Date(bar.time || "").toLocaleString()} ${bar.latency != null ? `(${bar.latency}ms)` : ""}`}
										></div>
									{/each}
								</div>
								{#if timeRange}
									<div class="flex justify-between mt-1">
										<span class="text-[9px] text-text-subtle/60 font-mono">{timeRange}</span>
										<span class="text-[9px] text-text-subtle/60 font-mono">now</span>
									</div>
								{/if}
							</div>
						{/if}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
