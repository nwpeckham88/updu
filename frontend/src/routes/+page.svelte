<script lang="ts">
	import {
		Activity,
		ServerCrash,
		CircleCheck,
		ArrowUpRight,
		EllipsisVertical,
		Waves,
	} from "lucide-svelte";
	import { resolve } from "$app/paths";
	import { monitorsStore } from "$lib/stores/monitors.svelte";
	import { settingsStore } from "$lib/stores/settings.svelte";
	import { densityStore } from "$lib/stores/density.svelte";
	import Skeleton from "$lib/components/ui/skeleton.svelte";
	import EmptyState from "$lib/components/ui/empty-state.svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Badge from "$lib/components/ui/badge.svelte";
	import Sparkline from "$lib/components/ui/sparkline.svelte";
	import StatusDonut from "$lib/components/charts/status-donut.svelte";
	import BulletBar from "$lib/components/charts/bullet-bar.svelte";
	import { isFlapping } from "$lib/monitor-tones";

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
	const attentionMonitors = $derived(
		monitors.filter(
			(m) =>
				m.enabled &&
				(m.status === "down" || m.status === "degraded" || monitorFlapping(m)),
		),
	);

	function monitorFlapping(monitor: any): boolean {
		return monitor.enabled && isFlapping(monitor.recent_checks);
	}

	function monitorPriority(monitor: any): number {
		if (monitor.enabled && monitor.status === "down") return 0;
		if (monitorFlapping(monitor)) return 1;
		if (monitor.enabled && monitor.status === "degraded") return 2;
		if (monitor.status === "pending") return 3;
		if (!monitor.enabled) return 4;
		return 5;
	}

	const displayMonitors = $derived.by(() =>
		[...monitors].sort((a, b) => {
			const priority = monitorPriority(a) - monitorPriority(b);
			if (priority !== 0) return priority;
			return a.name.localeCompare(b.name);
		}),
	);

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

	const apdexValues = $derived.by(() => {
		const values: number[] = [];
		for (const monitor of monitors) {
			for (const check of monitor.recent_checks ?? []) {
				if (check.latency_ms != null) values.push(check.latency_ms);
			}
		}
		return values;
	});

	const overallHealth = $derived(
		monitors.length === 0
			? null
			: monitors.filter((m) => m.status === "up").length /
					monitors.length,
	);

	type HeroMetric = {
		label: string;
		value: string | number;
		icon: typeof Activity;
		tone: "neutral" | "primary" | "success" | "warning" | "danger";
	};

	const heroMetrics = $derived<HeroMetric[]>([
		{
			label: "Total",
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
	]);

	const toneText: Record<HeroMetric["tone"], string> = {
		neutral: "text-text",
		primary: "text-primary",
		success: "text-success",
		warning: "text-warning",
		danger: "text-danger",
	};

	// Density-driven sparkline height (Sparkline expects a numeric prop)
	const sparkHeight = $derived(
		densityStore.current === "comfortable"
			? 36
			: densityStore.current === "compact"
				? 22
				: 30,
	);

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
			<p class="mt-1 type-caption text-text-muted">
			Real-time infrastructure overview
			</p>
		</div>
		{#if !loading && monitors.length > 0}
			<p class="type-caption text-text-muted">
				{upCount} healthy, {downCount} active {downCount === 1
					? "incident"
					: "incidents"}, {pausedCount} paused
			</p>
		{/if}
	</div>

	{#if !loading && attentionMonitors.length > 0}
		<section
			class="rounded-lg border border-danger/30 bg-danger/5 p-4"
			aria-labelledby="attention-heading"
		>
			<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="flex items-start gap-3">
					<div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-danger/10 text-danger">
						<ServerCrash class="size-4" aria-hidden="true" />
					</div>
					<div>
						<h2 id="attention-heading" class="type-section-title text-text">
							Needs attention
						</h2>
						<p class="mt-0.5 type-caption text-text-muted">
							{attentionMonitors.length} monitor{attentionMonitors.length === 1 ? "" : "s"} require triage.
						</p>
					</div>
				</div>
				<Button href="/monitors" variant="outline" size="sm">
					Open monitors <ArrowUpRight class="size-3.5" />
				</Button>
			</div>
			<div class="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
				{#each attentionMonitors.slice(0, 3) as monitor (monitor.id)}
					{@const flapping = monitorFlapping(monitor)}
					<a
						href={resolve("/monitors/[id]", { id: monitor.id })}
						class="flex items-center justify-between gap-3 rounded-lg border border-danger/20 bg-background/40 px-3 py-2 text-sm transition-colors hover:border-danger/40 hover:bg-danger/10"
					>
						<span class="min-w-0">
							<span class="block truncate font-medium text-text">{monitor.name}</span>
							<span class="type-numeric text-xs text-text-muted">
								{monitor.last_latency_ms != null ? `${monitor.last_latency_ms}ms` : "No latency sample"}
							</span>
						</span>
						<div class="flex shrink-0 items-center gap-1.5">
							{#if flapping}
								<span
									class="type-kicker inline-flex items-center gap-1 rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 text-warning"
									aria-label="Flapping: three or more status changes in ten minutes"
								>
									<Waves class="size-3" aria-hidden="true" />
									Flapping
								</span>
							{/if}
							<Badge status={monitor.status} calm={flapping} />
						</div>
					</a>
				{/each}
			</div>
		</section>
	{/if}

	<!-- Hero: health donut + inline KPI metrics, all in one card -->
	<div
		class="card flex flex-col gap-5 lg:flex-row lg:items-center lg:gap-6"
		style="padding: var(--d-hero-pad);"
		aria-busy={loading}
	>
		<!-- Left: donut + summary -->
		<div class="flex items-center gap-4 lg:shrink-0">
			{#if loading}
				<Skeleton height="h-[120px]" width="w-[120px]" rounded="rounded-full" />
				<div class="space-y-2">
					<Skeleton height="h-3" width="w-20" />
					<Skeleton height="h-5" width="w-32" />
				</div>
			{:else if overallHealth !== null}
				<StatusDonut
					value={overallHealth * 100}
					size="md"
					label="Fleet health"
					sublabel="Overall"
					apdexValues={apdexValues}
				/>
				<div class="min-w-0 space-y-1">
					<p
						class="type-kicker text-text-subtle"
						style="font-size: var(--d-stat-label);"
					>
						Health
					</p>
					<p class="type-data-title text-text">
						{upCount} of {monitors.length} operational
					</p>
					{#if downCount > 0}
						<p class="type-caption font-medium text-danger">
							{downCount}
							{downCount === 1 ? "incident" : "incidents"} active
						</p>
					{:else if pausedCount > 0}
						<p class="type-caption text-text-muted">{pausedCount} paused</p>
					{:else}
						<p class="type-caption text-success">All systems nominal</p>
					{/if}
				</div>
			{:else}
				<StatusDonut value={0} size="md" sublabel="No data" />
				<p class="type-caption text-text-muted">
					Add a monitor to see health.
				</p>
			{/if}
		</div>

		<!-- Vertical separator (only at lg+) -->
		<div class="hidden h-16 w-px bg-border/60 lg:block" aria-hidden="true"></div>

		<!-- Right: inline KPI metrics -->
		<div
			class="grid flex-1 grid-cols-2 sm:grid-cols-4"
			style="gap: var(--d-gap);"
		>
			{#each heroMetrics as metric (metric.label)}
				<div class="flex flex-col gap-1 min-w-0">
					<div class="flex items-center gap-1.5 text-text-subtle">
						<metric.icon class="size-3 shrink-0" aria-hidden="true" />
						<span
							class="type-kicker truncate"
							style="font-size: var(--d-stat-label);"
						>
							{metric.label}
						</span>
					</div>
					<span
						class="type-numeric font-bold {toneText[metric.tone]}"
						style="font-size: var(--d-stat-value);"
					>
						{metric.value}
					</span>
				</div>
			{/each}
			<div class="min-w-0">
				<BulletBar
					label="Avg latency"
					value={loading ? null : avgLatencyNum}
					target={500}
					warning={1000}
					danger={3000}
				/>
			</div>
		</div>
	</div>

	<!-- Monitor grid -->
	<div>
		<div class="mb-3 flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<h2 class="type-section-title text-text">All Monitors</h2>
				{#if !loading}
					<span
						class="type-numeric rounded-full border border-border/60 bg-surface/40 px-2 py-0.5 text-text-muted"
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
			<div class="dashboard-grid">
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
			<div class="dashboard-grid">
				{#each displayMonitors as monitor (monitor.id)}
					{@const isDown = monitor.status === "down"}
					{@const isPaused = !monitor.enabled}
					{@const flapping = monitorFlapping(monitor)}
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
						<div
							style="padding-left: var(--d-card-pad-x); padding-right: var(--d-card-pad-x); padding-top: var(--d-card-pad-y);"
						>
							<!-- Top line: name + menu -->
							<div class="flex items-start justify-between gap-2">
								<h3 class="type-data-title min-w-0 flex-1 truncate text-text">
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
									{#if flapping}
										<span class="type-kicker inline-flex items-center gap-1 text-warning">
											<Waves class="size-3" aria-hidden="true" />
											FLAPPING
										</span>
									{:else if isDown}
										<span class="type-kicker inline-flex items-center gap-1 text-danger">
											<span class="size-1.5 rounded-full bg-danger motion-safe:animate-pulse shadow-[0_0_6px_hsl(0_84%_60%/0.7)]"></span>
											INCIDENT
										</span>
									{:else if isPaused}
										<span class="type-kicker inline-flex items-center gap-1 text-text-subtle">
											<span class="size-1.5 rounded-full bg-text-subtle"></span>
											PAUSED
										</span>
									{:else}
										<span class="type-kicker inline-flex items-center gap-1 text-success">
											<span class="size-1.5 rounded-full bg-success shadow-[0_0_6px_hsl(142_71%_45%/0.7)]"></span>
											OPERATIONAL
										</span>
									{/if}
								</div>

								<div class="flex shrink-0 flex-col items-end gap-1 text-right leading-none">
									{#if monitor.uptime_24h != null}
										<span
											class="type-numeric type-micro font-bold {monitor.uptime_24h >= 99
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
											class="type-numeric type-micro font-bold {isDown ? 'text-danger' : 'text-text'}"
										>
											{monitor.last_latency_ms}ms
										</span>
									{/if}
								</div>
							</div>
						</div>

						<!-- Sparkline chart -->
						<div
							style="padding-left: var(--d-card-pad-x); padding-right: var(--d-card-pad-x); padding-top: 0.375rem;"
						>
							<Sparkline
								data={latencyData}
								width={240}
								height={sparkHeight}
								isDown={isDown}
							/>
						</div>

						<!-- Heartbeat bars + time labels -->
						{#if settingsStore.get("dashboard_show_heartbeat", "true") !== "false"}
							<div
								style="padding-left: var(--d-card-pad-x); padding-right: var(--d-card-pad-x); padding-top: 0.25rem; padding-bottom: var(--d-card-pad-y);"
							>
								<div
									class="flex items-end gap-[2px]"
									style="height: var(--d-heartbeat-h);"
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
										<span class="type-numeric type-micro text-text-subtle/60">{timeRange}</span>
										<span class="type-numeric type-micro text-text-subtle/60">now</span>
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
