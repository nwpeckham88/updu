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
		Bell,
		Sun,
		Moon,
		BarChart3,
	} from "lucide-svelte";
	import type { Icon } from "lucide-svelte";
	import { authStore } from "$lib/stores/auth.svelte";
	import { themeStore } from "$lib/stores/theme.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";

	let { children } = $props();

	const isLoginPage = $derived($page.url.pathname === "/login");
	const isStatusPage = $derived($page.url.pathname.startsWith("/status/"));

	// Custom CSS injection
	let customCSS = $state("");
	onMount(async () => {
		themeStore.init();
		try {
			const res = await fetch("/api/v1/custom.css");
			if (res.ok) {
				customCSS = await res.text();
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

	type NavLink = { href: string; label: string; icon: typeof Icon };

	const navLinks: NavLink[] = [
		{ href: "/", label: "Dashboard", icon: LayoutDashboard },
		{ href: "/monitors", label: "Monitors", icon: Server },
		{ href: "/stats", label: "Analytics", icon: BarChart3 },
		{ href: "/incidents", label: "Incidents", icon: TriangleAlert },
		{ href: "/status-pages", label: "Status Pages", icon: FileText },
		{ href: "/maintenance", label: "Maintenance", icon: Wrench },
		{ href: "/settings", label: "Notifications", icon: Bell },
	];

	function isActive(href: string) {
		const path = $page.url.pathname;
		return href === "/" ? path === "/" : path.startsWith(href);
	}

	let sidebarOpen = $state(false);
</script>

<svelte:head>
	{#if customCSS}
		{@html `<style id="updu-custom-css">${customCSS}</style>`}
	{/if}
</svelte:head>

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
			<nav class="flex-1 px-2.5 py-3 space-y-0.5 overflow-y-auto">
				{#each navLinks as { href, label, icon: Icon }}
					{@const active = isActive(href)}
					<a
						{href}
						onclick={() => (sidebarOpen = false)}
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
						<span>{label}</span>
					</a>
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
				class="h-14 border-b border-border bg-surface/50 backdrop-blur-xl sticky top-0 z-20 flex items-center justify-between px-4 lg:px-6 gap-4"
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

					<!-- Theme toggle -->
					<button
						onclick={() => themeStore.toggle()}
						class="size-8 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
						title={themeStore.current === "dark"
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
					>
						<LogOut class="size-3.5" />
						<span class="hidden sm:inline">Sign out</span>
					</button>
				</div>
			</header>

			<!-- Content -->
			<main class="flex-1 p-4 lg:p-6 animate-fade-in">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
