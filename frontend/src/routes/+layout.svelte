<script lang="ts">
	import "../app.css";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { onMount } from "svelte";
	import {
		Activity,
		LayoutDashboard,
		Settings,
		TriangleAlert,
		Server,
		FileText,
		LogOut,
		User,
		Wrench,
		Radio,
		Sun,
		Moon,
		BarChart3,
		Rows3,
		Rows4,
		Rows2,
	} from "lucide-svelte";
	import type { Icon } from "lucide-svelte";
	import { authStore } from "$lib/stores/auth.svelte";
	import { themeStore } from "$lib/stores/theme.svelte";
	import { densityStore } from "$lib/stores/density.svelte";
	import { settingsStore } from "$lib/stores/settings.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";
	import Toast from "$lib/components/ui/toast.svelte";
	import ConfirmDialog from "$lib/components/ui/confirm-dialog.svelte";
	import { fetchAPI } from "$lib/api/client";

	let { children } = $props();

	const isLoginPage = $derived($page.url.pathname === "/login");
	const isStatusPage = $derived($page.url.pathname.startsWith("/status/"));

	// Custom CSS injection
	let customCSS = $state("");
	let navMonitors = $state<NavMonitorStatus[]>([]);
	let unresolvedIncidentCount = $state(0);
	let navEventSource: EventSource | null = null;
	let navRefreshTimer: ReturnType<typeof setInterval> | null = null;

	interface NavMonitorStatus {
		id: string;
		status?: string;
		enabled?: boolean;
	}

	interface NavIncidentSummary {
		status?: string;
		resolved_at?: string | null;
	}

	// Sanitize CSS to prevent XSS via </style> tag injection
	function sanitizeCSSForInjection(css: string): string {
		return css.replace(/<\s*\/\s*style/gi, "/* blocked */");
	}

	onMount(async () => {
		themeStore.init();
		densityStore.init();
		settingsStore.init();
		try {
			const res = await fetch("/api/v1/custom.css");
			if (res.ok) {
				customCSS = sanitizeCSSForInjection(await res.text());
			}
		} catch {
			// ignore
		}
	});

	$effect(() => {
		if (!authStore.initialized) {
			authStore.init();
		}
	});

	$effect(() => {
		if (
			authStore.initialized &&
			!authStore.user &&
			!isLoginPage &&
			!isStatusPage
		) {
			goto("/login");
		}
	});

	$effect(() => {
		if (!authStore.user || isLoginPage || isStatusPage) {
			stopNavRealtime();
			return;
		}

		void refreshNavCounts();
		startNavRealtime();
		return stopNavRealtime;
	});

	const downMonitorCount = $derived(
		navMonitors.filter((monitor) => monitor.enabled !== false && monitor.status === "down").length,
	);

	type NavBadgeTone = "danger" | "warning";
	type NavLink = {
		href: string;
		label: string;
		icon: typeof Icon;
		badge?: number;
		badgeTone?: NavBadgeTone;
		badgeLabel?: string;
	};
	type NavSection = { label: string; links: NavLink[] };

	const navSections = $derived<NavSection[]>([
		{
			label: "Monitoring",
			links: [
				{ href: "/", label: "Dashboard", icon: LayoutDashboard },
				{
					href: "/monitors",
					label: "Monitors",
					icon: Server,
					badge: downMonitorCount,
					badgeTone: "danger",
					badgeLabel: `${downMonitorCount} down monitor${downMonitorCount === 1 ? "" : "s"}`,
				},
				{ href: "/stats", label: "Analytics", icon: BarChart3 },
			],
		},
		{
			label: "Reliability",
			links: [
				{
					href: "/incidents",
					label: "Incidents",
					icon: TriangleAlert,
					badge: unresolvedIncidentCount,
					badgeTone: "warning",
					badgeLabel: `${unresolvedIncidentCount} unresolved incident${unresolvedIncidentCount === 1 ? "" : "s"}`,
				},
				{ href: "/status-pages", label: "Status Pages", icon: FileText },
				{ href: "/maintenance", label: "Maintenance", icon: Wrench },
			],
		},
	]);

	function isActive(href: string) {
		const path = $page.url.pathname;
		return href === "/" ? path === "/" : path.startsWith(href);
	}

	function navBadgeText(count: number): string {
		return count > 99 ? "99+" : String(count);
	}

	function isUnresolvedIncident(incident: NavIncidentSummary): boolean {
		return incident.status !== "resolved" && !incident.resolved_at;
	}

	async function refreshNavCounts() {
		await Promise.allSettled([loadNavMonitors(), loadNavIncidents()]);
	}

	async function loadNavMonitors() {
		const data = await fetchAPI<{ monitors?: NavMonitorStatus[] }>("/api/v1/dashboard");
		navMonitors = data.monitors ?? [];
	}

	async function loadNavIncidents() {
		const data = await fetchAPI<NavIncidentSummary[]>("/api/v1/incidents");
		unresolvedIncidentCount = (data ?? []).filter(isUnresolvedIncident).length;
	}

	function startNavRealtime() {
		if (!navEventSource) {
			navEventSource = new EventSource("/api/v1/events");
			navEventSource.addEventListener("monitor:status", (event: MessageEvent) => {
				try {
					const payload = JSON.parse(event.data);
					applyNavMonitorStatus(payload);
				} catch {
					void loadNavMonitors();
				}
			});
			navEventSource.addEventListener("incident:change", () => {
				void loadNavIncidents();
			});
			navEventSource.onerror = () => {
				navEventSource?.close();
				navEventSource = null;
			};
		}

		if (!navRefreshTimer) {
			navRefreshTimer = setInterval(() => {
				void refreshNavCounts();
			}, 30000);
		}
	}

	function stopNavRealtime() {
		navEventSource?.close();
		navEventSource = null;
		if (navRefreshTimer) clearInterval(navRefreshTimer);
		navRefreshTimer = null;
	}

	function applyNavMonitorStatus(payload: { id?: string; status?: string }) {
		if (!payload.id) return;
		let found = false;
		navMonitors = navMonitors.map((monitor) => {
			if (monitor.id !== payload.id) return monitor;
			found = true;
			return { ...monitor, status: payload.status ?? monitor.status };
		});
		if (!found) void loadNavMonitors();
	}

	let sidebarOpen = $state(false);
</script>

<svelte:head>
	{#if customCSS}
		<style id="updu-custom-css">{customCSS}</style>
	{/if}
</svelte:head>

<Toast />
<ConfirmDialog />

<a
	href="#main-content"
	class="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-lg focus:bg-primary focus:text-white focus:shadow-lg focus:outline-2 focus:outline-primary"
>
	Skip to content
</a>

{#if isLoginPage || isStatusPage}
	{@render children()}
{:else if !authStore.initialized || authStore.loading}
	<div class="min-h-screen bg-background flex items-center justify-center">
		<div class="flex flex-col items-center gap-4">
			<div class="relative">
				<div
					class="size-12 rounded-2xl bg-primary/10 flex items-center justify-center border border-primary/20"
				>
					<Activity class="size-6 text-primary" />
				</div>
				<div
					class="absolute -bottom-1 -right-1 size-3 bg-primary rounded-full border-2 border-background animate-pulse"
				></div>
			</div>
			<div class="flex items-center gap-3">
				<Spinner size="sm" />
				<p class="text-sm text-text-subtle">Loading updu...</p>
			</div>
		</div>
	</div>
{:else if authStore.user}
	<!-- Mobile overlay -->
	{#if sidebarOpen}
		<button
			class="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm lg:hidden"
			onclick={() => (sidebarOpen = false)}
			aria-label="Close sidebar"
		></button>
	{/if}

	<div class="min-h-screen bg-background flex text-text font-sans">
		<!-- Sidebar -->
		<aside
			class="fixed inset-y-0 left-0 z-40 w-60 border-r border-border bg-surface/80 backdrop-blur-xl flex flex-col shrink-0 transition-transform duration-200 lg:translate-x-0 lg:static lg:z-auto {sidebarOpen
				? 'translate-x-0'
				: '-translate-x-full'}"
		>
			<!-- Logo -->
			<div
				class="px-4 py-4 flex items-center gap-3 border-b border-border/60"
			>
				<div class="relative shrink-0">
					<div
						class="size-8 rounded-xl bg-primary/15 flex items-center justify-center ring-1 ring-primary/20"
					>
						<Activity class="size-4 text-primary" />
					</div>
					<span
						class="absolute -bottom-0.5 -right-0.5 size-2.5 bg-success rounded-full border-2 border-surface shadow-[0_0_6px_hsl(142_71%_45%/0.7)]"
					></span>
				</div>
				<div>
					<span class="text-base font-bold tracking-tight text-text"
						>updu</span
					>
					<p class="text-[10px] text-text-subtle leading-none mt-0.5">
						Uptime Monitor
					</p>
				</div>
			</div>

			<!-- Nav -->
			<nav class="flex-1 px-2.5 py-3 space-y-4 overflow-y-auto" aria-label="Main">
				{#each navSections as section (section.label)}
					<div class="space-y-0.5">
						<p class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider text-text-subtle">
							{section.label}
						</p>
						{#each section.links as { href, label, icon: Icon, badge = 0, badgeTone = "warning", badgeLabel } (href)}
							{@const active = isActive(href)}
							<a
								{href}
								onclick={() => (sidebarOpen = false)}
								aria-current={active ? "page" : undefined}
								aria-label={badge > 0 && badgeLabel ? `${label}: ${badgeLabel}` : undefined}
								class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 relative group {active
									? 'bg-primary/10 text-primary'
									: 'text-text-muted hover:text-text hover:bg-surface-elevated'}"
							>
								{#if active}
									<span
										class="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-5 bg-primary rounded-r-full shadow-[0_0_8px_hsl(217_91%_60%/0.5)]"
									></span>
								{/if}
								<Icon class="size-4 shrink-0" />
								<span class="min-w-0 flex-1 truncate">{label}</span>
								{#if badge > 0 && badgeLabel}
									<span
										class="ml-auto inline-flex min-w-5 items-center justify-center rounded-full border px-1.5 py-0.5 text-[10px] font-bold tabular-nums leading-none {badgeTone === 'danger'
											? 'border-danger/30 bg-danger/10 text-danger'
											: 'border-warning/30 bg-warning/10 text-warning'}"
										aria-label={badgeLabel}
										title={badgeLabel}
									>
										{navBadgeText(badge)}
									</span>
								{/if}
							</a>
						{/each}
					</div>
				{/each}
			</nav>

			<!-- Bottom: settings -->
			<div class="px-2.5 py-3 border-t border-border/60 space-y-0.5">
				<a
					href="/settings"
					class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 {isActive(
						'/settings',
					)
						? 'bg-primary/10 text-primary'
						: 'text-text-muted hover:text-text hover:bg-surface-elevated'}"
				>
					<Settings class="size-4 shrink-0" />
					<span>Settings</span>
				</a>
			</div>
		</aside>

		<!-- Main -->
		<div class="flex-1 flex flex-col min-w-0 min-h-screen">
			<!-- Header -->
			<header
				class="h-[var(--app-header-h)] border-b border-border bg-surface/50 backdrop-blur-xl sticky top-0 z-20 flex items-center justify-between px-4 lg:px-6 gap-4"
			>
				<!-- Mobile: hamburger -->
				<button
					class="lg:hidden p-2 rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
					onclick={() => (sidebarOpen = !sidebarOpen)}
					aria-label="Toggle sidebar"
				>
					<svg
						class="size-5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
					>
						<line x1="3" y1="6" x2="21" y2="6" />
						<line x1="3" y1="12" x2="21" y2="12" />
						<line x1="3" y1="18" x2="21" y2="18" />
					</svg>
				</button>

				<!-- Page title derived from url -->
				<div class="hidden lg:block flex-1">
					<p
						class="text-xs text-text-subtle font-medium capitalize tracking-wide"
					>
						{$page.url.pathname === "/"
							? "Dashboard"
							: $page.url.pathname.replace("/", "").split("/")[0]}
					</p>
				</div>

				<!-- Right side -->
				<div class="flex items-center gap-2 ml-auto">
					<!-- Live indicator -->
					<div
						class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-success/10 border border-success/20 text-success text-xs font-medium"
					>
						<Radio class="size-3.5" />
						<span class="hidden sm:inline">Live</span>
					</div>

					<div class="w-px h-5 bg-border"></div>

					<!-- Density cycle -->
					<button
						onclick={() => densityStore.cycle()}
						class="size-8 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
						title="Density: {densityStore.current} (click to cycle)"
						aria-label="UI density: {densityStore.current}. Click to cycle."
					>
						{#if densityStore.current === "comfortable"}
							<Rows2 class="size-4" />
						{:else if densityStore.current === "cozy"}
							<Rows3 class="size-4" />
						{:else}
							<Rows4 class="size-4" />
						{/if}
					</button>

					<!-- Theme toggle -->
					<button
						onclick={() => themeStore.toggle()}
						class="size-8 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
						title={themeStore.current === "dark"
							? "Switch to light mode"
							: "Switch to dark mode"}
						aria-label={themeStore.current === "dark"
							? "Switch to light mode"
							: "Switch to dark mode"}
					>
						{#if themeStore.current === "dark"}
							<Sun class="size-4" />
						{:else}
							<Moon class="size-4" />
						{/if}
					</button>

					<!-- User -->
					<div class="flex items-center gap-2">
						<div
							class="size-7 rounded-lg bg-primary/15 flex items-center justify-center shrink-0"
						>
							<User class="size-3.5 text-primary" />
						</div>
						<div class="hidden sm:block">
							<p
								class="text-xs font-semibold text-text leading-none"
							>
								{authStore.user.username}
							</p>
							{#if authStore.user.role === "admin"}
								<p
									class="text-[9px] text-primary uppercase font-bold tracking-wider leading-none mt-0.5"
								>
									Admin
								</p>
							{/if}
						</div>
					</div>

					<div class="w-px h-5 bg-border"></div>

					<button
						onclick={() => authStore.logout()}
						class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs text-text-muted hover:text-danger hover:bg-danger/10 transition-colors font-medium"
						aria-label="Sign out"
					>
						<LogOut class="size-3.5" />
						<span class="hidden sm:inline">Sign out</span>
					</button>
				</div>
			</header>

			<!-- Content -->
			<main id="main-content" class="flex-1 p-4 lg:p-6 animate-fade-in" tabindex="-1">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
