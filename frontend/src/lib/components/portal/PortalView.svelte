<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import {
		Activity,
		CheckCircle2,
		AlertTriangle,
		Wrench,
		Clock,
		ExternalLink,
		LifeBuoy,
		RefreshCw,
		CalendarClock,
		ShieldCheck,
		ChevronRight,
		MessageSquarePlus,
		Check,
		XCircle,
		Sparkles,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Badge from "$lib/components/ui/badge.svelte";
	import Skeleton from "$lib/components/ui/skeleton.svelte";
	import { fetchAPI } from "$lib/api/client";
	import { authStore } from "$lib/stores/auth.svelte";
	import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
	import SubmitTicketModal from "./SubmitTicketModal.svelte";

	interface ServiceItem {
		id: string;
		name: string;
		type: string;
		status?: string;
		diagnosis?: string;
		enabled?: boolean;
		primary_latency_ms?: number;
		endpoints?: Array<{
			id: string;
			url?: string;
			target?: string;
			is_primary?: boolean;
			status?: string;
		}>;
	}

	interface MaintenanceItem {
		id: string;
		title: string;
		starts_at: string;
		ends_at: string;
		recurring?: string;
		monitor_ids?: string[];
	}

	interface TicketItem {
		id: string;
		title: string;
		description: string;
		status: "open" | "in_progress" | "resolved" | "closed";
		severity: "low" | "medium" | "high";
		service_id?: string;
		service_name?: string;
		created_by: string;
		created_at: string;
		resolved_at?: string;
	}

	let services = $state<ServiceItem[]>([]);
	let maintenanceWindows = $state<MaintenanceItem[]>([]);
	let tickets = $state<TicketItem[]>([]);
	let loading = $state(true);
	let refreshing = $state(false);

	// Ticket modal
	let ticketModalOpen = $state(false);
	let selectedServiceIdForTicket = $state<string>("");

	let eventSource: EventSource | null = null;
	let refreshInterval: ReturnType<typeof setInterval> | null = null;

	onMount(async () => {
		await loadAllData();
		startRealtime();
	});

	onDestroy(() => {
		stopRealtime();
	});

	async function loadAllData() {
		loading = true;
		try {
			await Promise.allSettled([
				loadServices(),
				loadMaintenance(),
				loadTickets(),
			]);
		} finally {
			loading = false;
		}
	}

	async function refreshData() {
		refreshing = true;
		try {
			await Promise.allSettled([
				loadServices(),
				loadMaintenance(),
				loadTickets(),
			]);
			toastStore.success("Status updated");
		} catch (e) {
			toastFromError(e, "Failed to refresh status");
		} finally {
			refreshing = false;
		}
	}

	async function loadServices() {
		try {
			const data = await fetchAPI<ServiceItem[]>("/api/v1/services");
			services = data || [];
		} catch {
			services = [];
		}
	}

	async function loadMaintenance() {
		try {
			const data = await fetchAPI<MaintenanceItem[]>("/api/v1/maintenance");
			maintenanceWindows = data || [];
		} catch {
			maintenanceWindows = [];
		}
	}

	async function loadTickets() {
		try {
			const data = await fetchAPI<TicketItem[]>("/api/v1/tickets");
			tickets = data || [];
		} catch {
			tickets = [];
		}
	}

	function startRealtime() {
		if (typeof window === "undefined") return;
		try {
			eventSource = new EventSource("/api/v1/events");
			eventSource.addEventListener("ticket:create", () => void loadTickets());
			eventSource.addEventListener("ticket:update", () => void loadTickets());
			eventSource.addEventListener("ticket:delete", () => void loadTickets());
			eventSource.addEventListener("monitor:status", () => void loadServices());
			eventSource.onerror = () => {
				eventSource?.close();
				eventSource = null;
			};
		} catch {
			// fallback
		}

		refreshInterval = setInterval(() => {
			void loadServices();
			void loadMaintenance();
			void loadTickets();
		}, 30000);
	}

	function stopRealtime() {
		eventSource?.close();
		eventSource = null;
		if (refreshInterval) clearInterval(refreshInterval);
		refreshInterval = null;
	}

	// Maintenance calculations
	function isWindowActive(mw: MaintenanceItem): boolean {
		const now = Date.now();
		const start = new Date(mw.starts_at).getTime();
		const end = new Date(mw.ends_at).getTime();
		return now >= start && now <= end;
	}

	function isWindowUpcoming(mw: MaintenanceItem): boolean {
		const now = Date.now();
		const start = new Date(mw.starts_at).getTime();
		// Within next 7 days
		return start > now && start <= now + 7 * 24 * 60 * 60 * 1000;
	}

	const activeMaintenance = $derived(
		maintenanceWindows.filter(isWindowActive),
	);
	const upcomingMaintenance = $derived(
		maintenanceWindows.filter(isWindowUpcoming),
	);

	// Status calculations
	const downServices = $derived(
		services.filter((s) => s.enabled !== false && s.status === "down"),
	);
	const degradedServices = $derived(
		services.filter((s) => s.enabled !== false && s.status === "degraded"),
	);
	const operationalCount = $derived(
		services.filter((s) => s.enabled !== false && (s.status === "up" || !s.status)).length,
	);

	const systemStatus = $derived.by(() => {
		if (activeMaintenance.length > 0) {
			return {
				tone: "maintenance",
				title: "Scheduled Maintenance in Progress",
				description: activeMaintenance[0].title,
				subtext: "Some homelab services may be briefly restarting or undergoing planned upgrades.",
			};
		}
		if (downServices.length > 0) {
			return {
				tone: "outage",
				title: `${downServices.length} Service${downServices.length === 1 ? "" : "s"} Experiencing Outage`,
				description: downServices.map((s) => s.name).join(", "),
				subtext: "Our automated monitoring detected an interruption. Administrators have been notified.",
			};
		}
		if (degradedServices.length > 0) {
			return {
				tone: "degraded",
				title: `${degradedServices.length} Service${degradedServices.length === 1 ? "" : "s"} Degraded`,
				description: degradedServices.map((s) => s.name).join(", "),
				subtext: "Performance may be slower than usual while systems recover.",
			};
		}
		return {
			tone: "operational",
			title: "All Systems Operational",
			description: `All ${services.length} accessible homelab services are running smoothly.`,
			subtext: "Everything is monitored and verified in real-time.",
		};
	});

	function formatDateTime(iso: string): string {
		try {
			const d = new Date(iso);
			return d.toLocaleString(undefined, {
				weekday: "short",
				month: "short",
				day: "numeric",
				hour: "numeric",
				minute: "2-digit",
			});
		} catch {
			return iso;
		}
	}

	function getPrimaryUrl(svc: ServiceItem): string | null {
		if (!svc.endpoints || svc.endpoints.length === 0) return null;
		const primary = svc.endpoints.find((ep) => ep.is_primary) || svc.endpoints[0];
		if (primary && primary.url && primary.url.startsWith("http")) {
			return primary.url;
		}
		return null;
	}

	function openTicketForService(serviceId: string) {
		selectedServiceIdForTicket = serviceId;
		ticketModalOpen = true;
	}

	async function handleCloseUserTicket(ticketId: string) {
		try {
			await fetchAPI(`/api/v1/tickets/${ticketId}`, {
				method: "PUT",
				body: JSON.stringify({ status: "closed" }),
			});
			toastStore.success("Ticket marked as closed");
			await loadTickets();
		} catch (e) {
			toastFromError(e, "Failed to close ticket");
		}
	}
</script>

<div class="max-w-5xl mx-auto space-y-8 py-2">
	<!-- Top Greeting & Header Bar -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border/60 pb-5">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-2xl font-bold tracking-tight text-text">
					Homelab Service Portal
				</h1>
				{#if authStore.user?.groups?.length}
					<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary border border-primary/20">
						{authStore.user.groups.join(", ")}
					</span>
				{/if}
			</div>
			<p class="text-sm text-text-subtle mt-1">
				Welcome, <strong class="text-text font-medium">{authStore.user?.username || "Guest"}</strong>. Real-time availability of your homelab applications.
			</p>
		</div>

		<div class="flex items-center gap-2.5">
			<Button
				variant="secondary"
				size="sm"
				onclick={refreshData}
				disabled={refreshing}
				class="flex items-center gap-1.5"
			>
				<RefreshCw class="size-3.5 {refreshing ? 'animate-spin' : ''}" />
				<span>Refresh</span>
			</Button>

			<Button
				variant="default"
				size="sm"
				onclick={() => openTicketForService("")}
				class="flex items-center gap-1.5 shadow-sm shadow-primary/20"
			>
				<MessageSquarePlus class="size-4" />
				<span>Report an Issue</span>
			</Button>
		</div>
	</div>

	<!-- System Status Hero Banner -->
	{#if loading}
		<div class="p-6 rounded-2xl border border-border bg-surface animate-pulse space-y-3">
			<div class="h-6 w-48 bg-surface-elevated rounded"></div>
			<div class="h-4 w-96 bg-surface-elevated rounded"></div>
		</div>
	{:else}
		<div
			class="relative overflow-hidden rounded-2xl border p-6 lg:p-7 transition-all {systemStatus.tone === 'operational'
				? 'border-success/30 bg-gradient-to-br from-success/10 via-surface to-surface'
				: systemStatus.tone === 'maintenance'
				? 'border-warning/30 bg-gradient-to-br from-warning/10 via-surface to-surface'
				: systemStatus.tone === 'degraded'
				? 'border-warning/40 bg-gradient-to-br from-warning/15 via-surface to-surface'
				: 'border-danger/40 bg-gradient-to-br from-danger/15 via-surface to-surface'}"
		>
			<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-5 relative z-10">
				<div class="flex items-start gap-4">
					<div
						class="size-12 rounded-2xl flex items-center justify-center shrink-0 border {systemStatus.tone === 'operational'
							? 'bg-success/15 text-success border-success/30 shadow-[0_0_20px_hsl(142_71%_45%/0.25)]'
							: systemStatus.tone === 'maintenance'
							? 'bg-warning/15 text-warning border-warning/30 shadow-[0_0_20px_hsl(38_92%_50%/0.25)]'
							: systemStatus.tone === 'degraded'
							? 'bg-warning/15 text-warning border-warning/30 shadow-[0_0_20px_hsl(38_92%_50%/0.25)]'
							: 'bg-danger/15 text-danger border-danger/30 shadow-[0_0_20px_hsl(0_84%_60%/0.25)]'}"
					>
						{#if systemStatus.tone === 'operational'}
							<CheckCircle2 class="size-6" />
						{:else if systemStatus.tone === 'maintenance'}
							<Wrench class="size-6" />
						{:else}
							<AlertTriangle class="size-6" />
						{/if}
					</div>

					<div class="space-y-1">
						<div class="flex items-center gap-2">
							<h2 class="text-xl font-bold tracking-tight text-text">
								{systemStatus.title}
							</h2>
							<span
								class="size-2 rounded-full animate-ping {systemStatus.tone === 'operational'
									? 'bg-success'
									: systemStatus.tone === 'maintenance'
									? 'bg-warning'
									: 'bg-danger'}"
							></span>
						</div>
						<p class="text-sm font-medium text-text-muted">
							{systemStatus.description}
						</p>
						<p class="text-xs text-text-subtle">
							{systemStatus.subtext}
						</p>
					</div>
				</div>

				<div class="shrink-0 flex items-center gap-3">
					<Button
						variant="outline"
						size="sm"
						onclick={() => openTicketForService("")}
						class="bg-surface/80 hover:bg-surface-elevated flex items-center gap-1.5"
					>
						<LifeBuoy class="size-3.5 text-primary" />
						<span>Need Help?</span>
					</Button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Active Maintenance Notice Card -->
	{#if activeMaintenance.length > 0}
		<div class="space-y-3">
			<h3 class="text-xs font-semibold uppercase tracking-wider text-warning flex items-center gap-1.5">
				<Wrench class="size-3.5" />
				<span>Ongoing Scheduled Maintenance</span>
			</h3>
			<div class="grid gap-3">
				{#each activeMaintenance as mw (mw.id)}
					<div class="p-4 rounded-xl border border-warning/30 bg-warning/5 backdrop-blur flex flex-col sm:flex-row sm:items-center justify-between gap-3">
						<div class="space-y-1">
							<div class="flex items-center gap-2">
								<span class="text-sm font-semibold text-text">{mw.title}</span>
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-warning/20 text-warning border border-warning/30">
									In Progress
								</span>
							</div>
							<p class="text-xs text-text-subtle flex items-center gap-2">
								<Clock class="size-3 text-warning" />
								<span>{formatDateTime(mw.starts_at)} → {formatDateTime(mw.ends_at)}</span>
							</p>
						</div>
						<div class="text-xs text-text-muted">
							Planned downtime. No action required.
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Upcoming Scheduled Maintenance Banner -->
	{#if upcomingMaintenance.length > 0}
		<div class="p-4 rounded-xl border border-primary/20 bg-primary/5 flex items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<div class="size-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary shrink-0">
					<CalendarClock class="size-4" />
				</div>
				<div>
					<p class="text-xs font-semibold text-text">
						Upcoming Maintenance: {upcomingMaintenance[0].title}
					</p>
					<p class="text-[11px] text-text-subtle">
						Scheduled for {formatDateTime(upcomingMaintenance[0].starts_at)}
					</p>
				</div>
			</div>
			<span class="text-xs text-primary font-medium">Scheduled</span>
		</div>
	{/if}

	<!-- Services Grid -->
	<div class="space-y-4">
		<div class="flex items-center justify-between">
			<div>
				<h3 class="text-lg font-bold text-text">Your Homelab Services</h3>
				<p class="text-xs text-text-subtle">
					Services provisioned for your account group ({services.length} available)
				</p>
			</div>
		</div>

		{#if loading}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each Array(6) as _}
					<div class="p-5 rounded-xl border border-border bg-surface space-y-3 animate-pulse">
						<div class="h-4 w-32 bg-surface-elevated rounded"></div>
						<div class="h-3 w-48 bg-surface-elevated rounded"></div>
					</div>
				{/each}
			</div>
		{:else if services.length === 0}
			<div class="p-8 rounded-2xl border border-border bg-surface text-center space-y-2">
				<ShieldCheck class="size-8 text-text-muted mx-auto" />
				<p class="text-sm font-semibold text-text">No services assigned</p>
				<p class="text-xs text-text-subtle">
					Your account does not have access to any specific service groups yet. Contact your administrator.
				</p>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each services as svc (svc.id)}
					{@const isDown = svc.status === 'down'}
					{@const isDegraded = svc.status === 'degraded'}
					{@const isUp = svc.status === 'up' || !svc.status}
					{@const primaryUrl = getPrimaryUrl(svc)}

					<div class="group relative p-5 rounded-xl border border-border bg-surface hover:border-border-hover hover:bg-surface-elevated/60 transition-all flex flex-col justify-between gap-4">
						<div class="space-y-2.5">
							<div class="flex items-start justify-between gap-2">
								<div class="min-w-0">
									<h4 class="text-sm font-semibold text-text truncate group-hover:text-primary transition-colors">
										{svc.name}
									</h4>
									<p class="text-[11px] text-text-subtle capitalize mt-0.5">
										{svc.type || "Web Application"}
									</p>
								</div>

								<!-- Status Pill -->
								{#if isDown}
									<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-danger/10 text-danger border border-danger/20">
										<span class="size-1.5 rounded-full bg-danger"></span>
										Outage
									</span>
								{:else if isDegraded}
									<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-warning/10 text-warning border border-warning/20">
										<span class="size-1.5 rounded-full bg-warning"></span>
										Degraded
									</span>
								{:else}
									<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-success/10 text-success border border-success/20">
										<span class="size-1.5 rounded-full bg-success"></span>
										Operational
									</span>
								{/if}
							</div>

							{#if svc.diagnosis && (isDown || isDegraded)}
								<p class="text-[11px] text-text-muted bg-surface-elevated/80 p-2 rounded-lg border border-border/50 line-clamp-2">
									{svc.diagnosis}
								</p>
							{/if}

							{#if svc.primary_latency_ms !== undefined}
								<div class="text-[11px] text-text-subtle">
									Response time: <strong class="text-text font-medium">{svc.primary_latency_ms}ms</strong>
								</div>
							{/if}
						</div>

						<!-- Card Footer Buttons -->
						<div class="pt-3 border-t border-border/60 flex items-center justify-between gap-2">
							{#if primaryUrl}
								<a
									href={primaryUrl}
									target="_blank"
									rel="noopener noreferrer"
									class="inline-flex items-center gap-1.5 text-xs font-medium text-primary hover:underline"
								>
									<span>Open Service</span>
									<ExternalLink class="size-3" />
								</a>
							{:else}
								<span class="text-[11px] text-text-subtle">Internal network</span>
							{/if}

							<button
								type="button"
								onclick={() => openTicketForService(svc.id)}
								class="text-[11px] text-text-muted hover:text-text hover:underline transition-colors ml-auto"
							>
								Report issue
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- User's Submitted Tickets Section -->
	<div class="space-y-4 pt-4 border-t border-border/60">
		<div class="flex items-center justify-between">
			<div>
				<h3 class="text-lg font-bold text-text">Your Reported Issues</h3>
				<p class="text-xs text-text-subtle">
					Track tickets you have submitted to homelab administrators
				</p>
			</div>

			<Button
				variant="outline"
				size="sm"
				onclick={() => openTicketForService("")}
				class="flex items-center gap-1.5 text-xs"
			>
				<LifeBuoy class="size-3.5 text-primary" />
				<span>New Ticket</span>
			</Button>
		</div>

		{#if tickets.length === 0}
			<div class="p-6 rounded-xl border border-dashed border-border bg-surface/40 text-center space-y-1.5">
				<CheckCircle2 class="size-5 text-success mx-auto" />
				<p class="text-xs font-medium text-text">No active issues reported</p>
				<p class="text-[11px] text-text-subtle">
					If you experience streaming buffers, broken links, or login troubles, report them here.
				</p>
			</div>
		{:else}
			<div class="divide-y divide-border border border-border rounded-xl bg-surface overflow-hidden">
				{#each tickets as ticket (ticket.id)}
					<div class="p-4 hover:bg-surface-elevated/40 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-3">
						<div class="space-y-1 min-w-0">
							<div class="flex items-center gap-2 flex-wrap">
								<span class="text-sm font-semibold text-text">{ticket.title}</span>
								{#if ticket.service_name}
									<span class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-surface-elevated border border-border text-text-muted">
										{ticket.service_name}
									</span>
								{/if}
								<!-- Status Pill -->
								{#if ticket.status === 'open'}
									<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-warning/10 text-warning border border-warning/20">
										Open (Investigating)
									</span>
								{:else if ticket.status === 'in_progress'}
									<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-primary/10 text-primary border border-primary/20">
										In Progress
									</span>
								{:else if ticket.status === 'resolved'}
									<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-success/10 text-success border border-success/20">
										Resolved
									</span>
								{:else}
									<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-surface-elevated text-text-subtle border border-border">
										Closed
									</span>
								{/if}
							</div>
							<p class="text-xs text-text-muted line-clamp-1">
								{ticket.description}
							</p>
							<p class="text-[10px] text-text-subtle">
								Submitted on {formatDateTime(ticket.created_at)}
								{#if ticket.resolved_at}
									· Resolved on {formatDateTime(ticket.resolved_at)}
								{/if}
							</p>
						</div>

						{#if ticket.status === 'open' || ticket.status === 'in_progress'}
							<div class="shrink-0">
								<Button
									variant="ghost"
									size="sm"
									onclick={() => handleCloseUserTicket(ticket.id)}
									class="text-xs text-text-subtle hover:text-text"
								>
									Mark Solved
								</Button>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Ticket Modal -->
<SubmitTicketModal
	bind:open={ticketModalOpen}
	services={services.map((s) => ({ id: s.id, name: s.name }))}
	initialServiceId={selectedServiceIdForTicket}
	onSubmitted={() => void loadTickets()}
/>
