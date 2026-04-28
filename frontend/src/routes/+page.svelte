<script lang="ts">
	import {
		Activity,
		ServerCrash,
		ArrowUpRight,
		EllipsisVertical,
		Waves,
		ShieldAlert,
		ShieldCheck,
		ShieldX,
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
	import {
		isFlapping,
		latencyTextClass,
		statusIcon,
		statusLabel,
		statusPattern,
		statusTextClass,
		uptimeTextClass,
	} from "$lib/monitor-tones";

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
		monitors.filter((m) => m.enabled && m.status === "down").length,
	);
	const pausedCount = $derived(monitors.filter((m) => !m.enabled).length);
	const activeMonitorCount = $derived(monitors.filter((m) => m.enabled).length);
	const pendingCount = $derived(
		monitors.filter((m) => m.enabled && m.status === "pending").length,
	);
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

	type DashboardVerdict = {
		key: "empty" | "operational" | "degraded" | "outage";
		label: string;
		sub: string;
		tone: "neutral" | "success" | "warning" | "danger";
		icon: typeof ShieldCheck;
	};

	const dashboardVerdict = $derived.by((): DashboardVerdict => {
		if (monitors.length === 0) {
			return {
				key: "empty",
				label: "No monitors configured",
				sub: "Add a monitor to begin tracking services.",
				tone: "neutral",
				icon: ShieldAlert,
			};
		}

		if (activeMonitorCount === 0) {
			return {
				key: "empty",
				label: "No active monitors",
				sub: `${pausedCount} monitor${pausedCount === 1 ? "" : "s"} paused.`,
				tone: "neutral",
				icon: ShieldAlert,
			};
		}

		if (downCount > 0) {
			return {
				key: "outage",
				label: "Service outage",
				sub: `${downCount} monitor${downCount === 1 ? "" : "s"} down.`,
				tone: "danger",
				icon: ShieldX,
			};
		}

		if (attentionMonitors.length > 0) {
			return {
				key: "degraded",
				label: "Degraded performance",
				sub: `${attentionMonitors.length} monitor${attentionMonitors.length === 1 ? "" : "s"} need triage.`,
				tone: "warning",
				icon: ShieldAlert,
			};
		}

		if (pendingCount > 0) {
			return {
				key: "empty",
				label: "Checks pending",
				sub: `${upCount} healthy, ${pendingCount} waiting for first result.`,
				tone: "neutral",
				icon: ShieldAlert,
			};
		}

		return {
			key: "operational",
			label: "All systems operational",
			sub: `${upCount} of ${activeMonitorCount} active monitor${activeMonitorCount === 1 ? "" : "s"} healthy.`,
			tone: "success",
			icon: ShieldCheck,
		};
	});

	type AttentionTone = "danger" | "warning";

	const verdictRing: Record<DashboardVerdict["tone"], string> = {
		neutral: "border-border bg-surface",
		success: "border-success/30 bg-success/5",
		warning: "border-warning/30 bg-warning/5",
		danger: "border-danger/35 bg-danger/5",
	};

	const verdictIconTone: Record<DashboardVerdict["tone"], string> = {
		neutral: "border-border bg-surface-elevated text-text-muted",
		success: "border-success/25 bg-success/10 text-success",
		warning: "border-warning/25 bg-warning/10 text-warning",
		danger: "border-danger/25 bg-danger/10 text-danger",
	};

	const attentionTone: AttentionTone = $derived(
		downCount > 0 ? "danger" : "warning",
	);
	const attentionClasses: Record<AttentionTone, string> = {
		danger: "border-danger/30 bg-danger/5",
		warning: "border-warning/30 bg-warning/5",
	};
	const attentionIconClasses: Record<AttentionTone, string> = {
		danger: "bg-danger/10 text-danger",
		warning: "bg-warning/10 text-warning",
	};
	const attentionItemClasses: Record<AttentionTone, string> = {
		danger:
			"border-danger/20 bg-background/40 hover:border-danger/40 hover:bg-danger/10",
		warning:
			"border-warning/20 bg-background/40 hover:border-warning/40 hover:bg-warning/10",
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
			const status = monitor.enabled ? checks[i].status : "paused";
			bars[barCount - 1 - i] = {
				status,
				latency: monitor.enabled ? checks[i].latency_ms : undefined,
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
				else if (bar.status === "degraded" || bar.status === "warning") {
					summary.degraded += 1;
				} else if (bar.status === "paused") {
					summary.paused += 1;
				} else summary.empty += 1;
				return summary;
			},
			{ up: 0, down: 0, degraded: 0, paused: 0, empty: 0 },
		);

		const segments = [`${monitor.name} heartbeat history`];
		if (timeRange) segments.push(`over ${timeRange}`);
		segments.push(
			`${counts.up} healthy checks`,
			`${counts.down} failed checks`,
			`${counts.degraded} degraded checks`,
		);
		if (counts.paused > 0) {
			segments.push(`${counts.paused} paused periods`);
		}
		if (counts.empty > 0) {
			segments.push(`${counts.empty} periods without data`);
		}
		return segments.join(", ");
	}

	function heartbeatStatusClass(status: string): string {
		switch (status) {
			case "up":
				return "bg-success/70 hover:bg-success";
			case "down":
				return "bg-danger/80 hover:bg-danger";
			case "degraded":
			case "warning":
				return "bg-warning/75 hover:bg-warning";
			case "paused":
				return "bg-text-subtle/45 hover:bg-text-subtle/60";
			default:
				return "bg-border/30";
		}
	}

	function heartbeatPatternClass(status: string): string {
		const pattern = statusPattern(status);
		if (pattern === "diagonal") return "dashboard-pattern-diagonal";
		if (pattern === "dotted") return "dashboard-pattern-dotted";
		return "";
	}

	function heartbeatTitle(bar: { status: string; latency?: number; time?: string }): string {
		if (bar.status === "empty") return "No data";
		const parts = [statusLabel(bar.status)];
		if (bar.time) parts.push(new Date(bar.time).toLocaleString());
		if (bar.latency != null) parts.push(`${bar.latency}ms`);
		return parts.join(" | ");
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

	{#if loading}
		<section class="card border border-border p-5" aria-busy="true">
			<Skeleton height="h-20" width="w-full" />
		</section>
	{:else}
		{@const VerdictIcon = dashboardVerdict.icon}
		<section
			class="card flex flex-col gap-4 border p-5 sm:flex-row sm:items-center sm:justify-between {verdictRing[dashboardVerdict.tone]}"
			aria-label="System health: {dashboardVerdict.label}. {dashboardVerdict.sub}"
		>
			<div class="flex items-center gap-4">
				<div
					class="flex size-12 shrink-0 items-center justify-center rounded-2xl border {verdictIconTone[dashboardVerdict.tone]}"
				>
					<VerdictIcon class="size-6" aria-hidden="true" />
				</div>
				<div class="min-w-0">
					<p class="type-kicker text-text-subtle">System health</p>
					<h2 class="text-xl font-bold tracking-tight text-text">
						{dashboardVerdict.label}
					</h2>
					<p class="type-caption text-text-muted">{dashboardVerdict.sub}</p>
				</div>
			</div>

			<dl
				class="grid w-full grid-cols-2 gap-3 text-left sm:w-auto sm:grid-cols-3 sm:gap-8 sm:text-right"
				aria-label="Key dashboard metrics"
			>
				<div>
					<dt class="type-kicker text-text-subtle">Down</dt>
					<dd class="type-numeric text-2xl font-bold tabular-nums text-text">
						{downCount}/{monitors.length}
					</dd>
				</div>
				<div>
					<dt class="type-kicker text-text-subtle">Needs attention</dt>
					<dd class="type-numeric text-2xl font-bold tabular-nums {attentionMonitors.length > 0 ? 'text-warning' : 'text-text'}">
						{attentionMonitors.length}
					</dd>
				</div>
				<div class="col-span-2 sm:col-span-1">
					<dt class="type-kicker text-text-subtle">Avg latency</dt>
					<dd class="type-numeric text-2xl font-bold tabular-nums {latencyTextClass(avgLatencyNum)}">
						{avgLatencyNum == null ? "—" : `${avgLatencyNum}ms`}
					</dd>
				</div>
			</dl>
		</section>
	{/if}

	{#if !loading && attentionMonitors.length > 0}
		<section
			class="rounded-lg border p-4 {attentionClasses[attentionTone]}"
			aria-labelledby="attention-heading"
		>
			<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="flex items-start gap-3">
					<div class="flex size-9 shrink-0 items-center justify-center rounded-lg {attentionIconClasses[attentionTone]}">
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
					{@const itemTone = monitor.status === "down" ? "danger" : "warning"}
					<a
						href={resolve("/monitors/[id]", { id: monitor.id })}
						class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm transition-colors {attentionItemClasses[itemTone]}"
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
					{@const isPaused = !monitor.enabled}
					{@const isDown = monitor.enabled && monitor.status === "down"}
					{@const displayStatus = isPaused ? "paused" : monitor.status}
					{@const StatusIcon = statusIcon(displayStatus)}
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
										<span
											class="type-kicker inline-flex items-center gap-1 text-warning"
											aria-label="Flapping: {statusLabel(displayStatus)}"
										>
											<Waves class="size-3" aria-hidden="true" />
											Flapping
										</span>
									{:else}
										<span
											class="type-kicker inline-flex items-center gap-1 {statusTextClass(displayStatus)}"
											aria-label="Status: {statusLabel(displayStatus)}"
										>
											<StatusIcon class="size-3" aria-hidden="true" />
											{statusLabel(displayStatus)}
										</span>
									{/if}
								</div>

								<div class="flex shrink-0 flex-col items-end gap-1 text-right leading-none">
									{#if monitor.uptime_24h != null}
										<span
											class="type-numeric type-micro font-bold {uptimeTextClass(monitor.uptime_24h)}"
										>
											{monitor.uptime_24h.toFixed(2)}% Uptime
										</span>
									{/if}
									{#if monitor.last_latency_ms != null}
										<span
											class="type-numeric type-micro font-bold {isDown ? 'text-danger' : latencyTextClass(monitor.last_latency_ms)}"
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
											class="flex-1 rounded-[2px] transition-colors {heartbeatStatusClass(bar.status)} {heartbeatPatternClass(bar.status)}"
											style="height: {bar.status === 'empty' ? '30%' : '100%'}"
											aria-hidden="true"
											title={heartbeatTitle(bar)}
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

<style>
	:global(.dashboard-pattern-diagonal) {
		background-image: repeating-linear-gradient(
			45deg,
			transparent 0,
			transparent 2px,
			rgba(0, 0, 0, 0.35) 2px,
			rgba(0, 0, 0, 0.35) 4px
		);
	}

	:global(.dashboard-pattern-dotted) {
		background-image: radial-gradient(
			rgba(0, 0, 0, 0.35) 1px,
			transparent 1px
		);
		background-size: 4px 4px;
	}
</style>
