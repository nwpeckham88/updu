<script lang="ts">
	import {
		Activity,
		ServerCrash,
		CircleCheck,
		ArrowUpRight,
		BarChart3,
		EllipsisVertical,
	} from "lucide-svelte";
	import { resolve } from "$app/paths";
	import { monitorsStore } from "$lib/stores/monitors.svelte";
	import { settingsStore } from "$lib/stores/settings.svelte";
	import Skeleton from "$lib/components/ui/skeleton.svelte";
	import EmptyState from "$lib/components/ui/empty-state.svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Sparkline from "$lib/components/ui/sparkline.svelte";
	import Stat from "$lib/components/ui/stat.svelte";
	import StatusDonut from "$lib/components/charts/status-donut.svelte";

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
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: any;
		tone: "neutral" | "primary" | "success" | "warning" | "danger";
	};

	const statCards = $derived<StatCard[]>([
		{
			label: "Total Monitors",
			value: loading ? "—" : monitors.length,
			icon: Activity,
			tone: "primary",
		},
		{
			label: "Operational",
			value: loading ? "—" : upCount,
			icon: CircleCheck,
			tone: "success",
		},
		{
			label: "Incidents",
			value: loading ? "—" : downCount,
			icon: ServerCrash,
			tone: downCount > 0 ? "danger" : "neutral",
		},
		{
			label: "Avg Latency",
			value: loading ? "—" : avgLatency,
			icon: BarChart3,
			tone: "neutral",
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

	function getHeartbeatAriaLabel(
		monitor: any,
		heartbeat: { status: string }[],
		timeRange: string,
	): string {
		const counts = heartbeat.reduce(
			(summary, bar) => {
				if (bar.status === "up") summary.up += 1;
				else if (bar.status === "down") summary.down += 1;
				else summary.empty += 1;
				return summary;
			},
			{ up: 0, down: 0, empty: 0 },
		);

		const segments = [`${monitor.name} heartbeat history`];
		if (timeRange) segments.push(`over ${timeRange}`);
		segments.push(`${counts.up} healthy checks`, `${counts.down} failed checks`);
		if (counts.empty > 0) {
			segments.push(`${counts.empty} periods without data`);
		}
		return segments.join(", ");
	}
</script>

<svelte:head>
	<title>Dashboard – updu</title>
</svelte:head>


<div class="w-full space-y-4">
	<!-- Page header -->
	<div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-text">Dashboard</h1>
			<p class="mt-1 text-sm text-text-muted">
			Real-time infrastructure overview
			</p>
		</div>
		{#if !loading && monitors.length > 0}
			<p class="text-xs text-text-muted">
				{upCount} healthy, {downCount} active {downCount === 1
					? "incident"
					: "incidents"}, {pausedCount} paused
			</p>
		{/if}
	</div>

	<!-- Hero: health donut + stat tiles -->
	<div class="grid grid-cols-1 gap-3 xl:grid-cols-12">
		<!-- Health donut -->
		<div
			class="card flex flex-col gap-4 p-5 md:flex-row md:items-center xl:col-span-4 xl:h-full"
			aria-busy={loading}
		>
			{#if loading}
				<Skeleton height="h-[140px]" width="w-[140px]" rounded="rounded-full" />
				<div class="flex-1 space-y-2">
					<Skeleton height="h-3" width="w-24" />
					<Skeleton height="h-6" width="w-32" />
				</div>
			{:else if overallHealth !== null}
				<StatusDonut
					value={overallHealth * 100}
					size="md"
					sublabel="Overall"
				/>
				<div class="min-w-0 flex-1 space-y-1.5">
					<p class="text-[10px] font-semibold uppercase tracking-wider text-text-subtle">
						Health
					</p>
					<p class="text-base font-semibold text-text">
						{upCount} of {monitors.length} operational
					</p>
					{#if downCount > 0}
						<p class="text-xs font-medium text-danger">
							{downCount} {downCount === 1 ? "incident" : "incidents"} active
						</p>
					{:else if pausedCount > 0}
						<p class="text-xs text-text-muted">
							{pausedCount} paused
						</p>
					{:else}
						<p class="text-xs text-success">All systems nominal</p>
					{/if}
				</div>
			{:else}
				<StatusDonut value={0} size="md" sublabel="No data" />
				<div class="min-w-0 flex-1">
					<p class="text-sm text-text-muted">
						Add a monitor to see health.
					</p>
				</div>
			{/if}
		</div>

		<!-- Stat tiles -->
		<div class="grid grid-cols-2 gap-3 lg:grid-cols-4 xl:col-span-8">
			{#each statCards as card (card.label)}
				{#if loading}
					<div class="card h-full space-y-3 p-4">
						<Skeleton height="h-3" width="w-24" />
						<Skeleton height="h-8" width="w-16" />
					</div>
				{:else}
					<Stat
						label={card.label}
						value={card.value}
						icon={card.icon}
						tone={card.tone}
						class="h-full"
					/>
				{/if}
			{/each}
		</div>
	</div>

	<!-- Monitor grid -->
	<div>
		<div class="mb-3 flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<h2 class="text-base font-semibold text-text">All Monitors</h2>
				{#if !loading}
					<span
						class="rounded-full border border-border/60 bg-surface/40 px-2 py-0.5 text-[11px] font-mono text-text-muted"
					>
						{monitors.length}
					</span>
				{/if}
			</div>
			{#if !loading}
				<Button href="/monitors" variant="ghost" size="sm">
					View all <ArrowUpRight class="size-3.5" />
				</Button>
			{/if}
		</div>

		{#if loading}
			<div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5">
				{#each { length: 6 } as _, index (index)}
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
			<div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5">
				{#each monitors as monitor (monitor.id)}
					{@const isDown = monitor.status === "down"}
					{@const isPaused = !monitor.enabled}
					{@const heartbeat = buildHeartbeat(monitor)}
					{@const latencyData = getLatencyData(monitor)}
					{@const timeRange = getTimeRangeLabel(monitor)}

					<a
						href={resolve("/monitors/[id]", { id: monitor.id })}
						data-sveltekit-preload-data="hover"
						class="card card-interactive text-left w-full p-0 flex flex-col {isDown
							? 'border-danger/30 bg-danger/5'
							: ''}"
					>
						<!-- Card header -->
						<div class="px-3.5 pt-3 pb-0">
							<!-- Top line: name + menu -->
							<div class="flex items-start justify-between gap-2">
								<h3 class="min-w-0 flex-1 truncate text-sm font-semibold leading-tight text-text">
									{monitor.name}
								</h3>
								<EllipsisVertical
									class="pointer-events-none mt-0.5 size-4 shrink-0 text-text-subtle opacity-40 transition-opacity hover:opacity-100"
									aria-hidden="true"
								/>
							</div>

							<!-- Status + metrics row -->
							<div class="mt-1.5 flex items-start justify-between gap-3">
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

								<div class="flex shrink-0 flex-col items-end gap-1 text-right leading-none">
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
						</div>

						<!-- Sparkline chart -->
						<div class="px-3.5 pt-1.5 pb-0">
							<Sparkline
								data={latencyData}
								width={240}
								height={30}
								isDown={isDown}
							/>
						</div>

						<!-- Heartbeat bars + time labels -->
						{#if settingsStore.get("dashboard_show_heartbeat", "true") !== "false"}
							<div class="px-3.5 pt-1 pb-2.5">
								<div
									class="flex h-3.5 items-end gap-[2px]"
									role="img"
									aria-label={getHeartbeatAriaLabel(
										monitor,
										heartbeat,
										timeRange,
									)}
								>
									{#each heartbeat as bar, index (`${monitor.id}-${index}`)}
										<div
											class="flex-1 rounded-[2px] transition-colors {bar.status === 'up'
												? 'bg-success/70 hover:bg-success'
												: bar.status === 'down'
													? 'bg-danger/80 hover:bg-danger'
													: 'bg-border/30'}"
											style="height: {bar.status === 'empty' ? '30%' : '100%'}"
											aria-hidden="true"
											title={bar.status === "empty"
												? "No data"
												: `${bar.status.toUpperCase()} · ${new Date(bar.time || "").toLocaleString()} ${bar.latency != null ? `(${bar.latency}ms)` : ""}`}
										></div>
									{/each}
								</div>
								{#if timeRange}
									<div class="mt-1 flex justify-between">
										<span class="text-[8px] text-text-subtle/60 font-mono">{timeRange}</span>
										<span class="text-[8px] text-text-subtle/60 font-mono">now</span>
									</div>
								{/if}
							</div>
						{/if}
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
