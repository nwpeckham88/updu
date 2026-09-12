<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import {
		getNetworkTopology,
		probeService,
		createService,
		deleteService,
		createServiceEndpoint,
		deleteServiceEndpoint,
		type NetworkTopology,
		type Service,
		type ServiceEndpoint,
	} from "$lib/api/services";
	import {
		Network,
		Layers,
		Grid3X3,
		Activity,
		RefreshCw,
		Radio,
		Server,
		Plus,
		ShieldAlert,
		Trash2,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";
	import Modal from "$lib/components/ui/modal.svelte";
	import ZoneCardsView from "./ZoneCardsView.svelte";
	import ReachabilityMatrixView from "./ReachabilityMatrixView.svelte";
	import HopInspectorView from "./HopInspectorView.svelte";

	let topology = $state<NetworkTopology | null>(null);
	let loading = $state(true);
	let error = $state("");
	let viewMode = $state<"zones" | "matrix" | "inspector">("zones");
	let probingId = $state<string | null>(null);
	let selectedService = $state<Service | null>(null);
	let focusedScopeId = $state<string | null>(null);

	// Modals State
	let showAddServiceModal = $state(false);
	let showAddEndpointModal = $state(false);
	let submitting = $state(false);
	let formError = $state("");

	// Add Service Form
	let newServiceName = $state("");
	let newServiceType = $state("web");
	let newServiceZoneId = $state("default");
	let newServiceHAGroup = $state("");
	let newServiceScopeId = $state("public");
	let newServiceTarget = $state("https://");
	let newServiceInterval = $state(60);
	let newServiceTimeout = $state(10);

	// Add Endpoint Form
	let newEndpointName = $state("");
	let newEndpointScopeId = $state("lan");
	let newEndpointTargetType = $state("http");
	let newEndpointTarget = $state("http://");
	let newEndpointInterval = $state(30);
	let newEndpointTimeout = $state(10);
	let newEndpointIsPrimary = $state(false);

	// Deletion progress
	let deletingEndpointId = $state<string | null>(null);
	let deletingServiceId = $state<string | null>(null);

	// SSE stream
	let eventSource: EventSource | null = null;

	async function loadTopology() {
		try {
			loading = true;
			error = "";
			topology = await getNetworkTopology();
			if (selectedService && topology?.services) {
				selectedService = topology.services.find((s) => s.id === selectedService?.id) || null;
			}
		} catch (e: any) {
			error = e?.message || "Failed to load network topology";
		} finally {
			loading = false;
		}
	}

	async function runProbe(serviceId: string) {
		probingId = serviceId;
		try {
			await probeService(serviceId);
			await loadTopology();
			if (selectedService && selectedService.id === serviceId && topology) {
				selectedService = topology?.services?.find((s) => s.id === serviceId) || null;
			}
		} catch (e: any) {
			alert("Probe failed: " + (e?.message || "Unknown error"));
		} finally {
			probingId = null;
		}
	}

	function handleSelectService(service: Service, focusedScope?: string) {
		selectedService = service;
		focusedScopeId = focusedScope || null;
		viewMode = "inspector";
	}

	function handleOpenAddService(zoneId?: string) {
		formError = "";
		newServiceName = "";
		newServiceHAGroup = "";
		newServiceTarget = "https://";
		if (zoneId) newServiceZoneId = zoneId;
		showAddServiceModal = true;
	}

	function handleOpenAddEndpoint(service: Service, scopeId?: string) {
		formError = "";
		selectedService = service;
		newEndpointName = "";
		newEndpointTarget = scopeId === "public" ? "https://" : "http://";
		if (scopeId) newEndpointScopeId = scopeId;
		newEndpointIsPrimary = false;
		showAddEndpointModal = true;
	}

	function connectSSE() {
		if (eventSource) return;
		eventSource = new EventSource("/api/v1/events");

		eventSource.addEventListener("service:status", (e: MessageEvent) => {
			try {
				const data = JSON.parse(e.data);
				if (!topology) return;
				if (topology.services) {
					const idx = topology.services.findIndex((s) => s.id === data.id);
					if (idx !== -1) {
						topology.services[idx].status = data.status;
						topology.services[idx].diagnosis = data.diagnosis;
						if (data.endpoints) {
							topology.services[idx].endpoints = data.endpoints;
						}
					}
				}
				if (topology.edges) {
					topology.edges = topology.edges.map((edge) => {
						if (edge.service_id === data.id) {
							const ep = data.endpoints?.find((x: any) => x.id === edge.endpoint_id);
							if (ep) {
								return {
									...edge,
									status: ep.status,
									latency_ms: ep.last_latency_ms,
									message: ep.last_message,
									metadata: ep.last_metadata,
								};
							}
						}
						return edge;
					});
				}
				if (selectedService && selectedService.id === data.id) {
					selectedService = {
						...selectedService,
						status: data.status,
						diagnosis: data.diagnosis,
						endpoints: data.endpoints || selectedService.endpoints,
					};
				}
			} catch (err) {
				console.error("SSE parse error", err);
			}
		});

		eventSource.onerror = () => {
			eventSource?.close();
			eventSource = null;
			setTimeout(() => connectSSE(), 5000);
		};
	}

	async function handleCreateService(e: Event) {
		e.preventDefault();
		if (!newServiceName.trim()) {
			formError = "Service name is required";
			return;
		}
		submitting = true;
		formError = "";
		try {
			const targetType = newServiceType === "web" ? "http" : newServiceType;
			const config = targetType === "http"
				? { url: newServiceTarget.trim() }
				: { host: newServiceTarget.trim() };

			await createService({
				name: newServiceName.trim(),
				type: newServiceType,
				zone_id: newServiceZoneId,
				ha_group: newServiceHAGroup.trim() || undefined,
				enabled: true,
				endpoints: [
					{
						id: "",
						service_id: "",
						name: "Primary Ingress",
						scope_id: newServiceScopeId,
						target_type: targetType,
						config,
						interval_s: Number(newServiceInterval) || 60,
						timeout_s: Number(newServiceTimeout) || 10,
						retries: 2,
						is_primary: true,
					},
				],
			});
			showAddServiceModal = false;
			newServiceName = "";
			newServiceHAGroup = "";
			newServiceTarget = "https://";
			await loadTopology();
		} catch (err: any) {
			formError = err?.message || "Failed to create service";
		} finally {
			submitting = false;
		}
	}

	async function handleCreateEndpoint(e: Event) {
		e.preventDefault();
		if (!selectedService) return;
		if (!newEndpointName.trim()) {
			formError = "Endpoint name is required";
			return;
		}
		submitting = true;
		formError = "";
		try {
			const config = newEndpointTargetType === "http"
				? { url: newEndpointTarget.trim() }
				: { host: newEndpointTarget.trim() };

			await createServiceEndpoint(selectedService.id, {
				name: newEndpointName.trim(),
				scope_id: newEndpointScopeId,
				target_type: newEndpointTargetType,
				config,
				interval_s: Number(newEndpointInterval) || 30,
				timeout_s: Number(newEndpointTimeout) || 10,
				retries: 2,
				is_primary: newEndpointIsPrimary,
			});
			showAddEndpointModal = false;
			newEndpointName = "";
			newEndpointTarget = "http://";
			newEndpointIsPrimary = false;
			await loadTopology();
			if (topology && selectedService) {
				selectedService = topology.services?.find((s) => s.id === selectedService?.id) || null;
			}
		} catch (err: any) {
			formError = err?.message || "Failed to create endpoint";
		} finally {
			submitting = false;
		}
	}

	async function handleDeleteEndpoint(serviceId: string, endpointId: string) {
		if (!confirm("Are you sure you want to delete this endpoint?")) return;
		deletingEndpointId = endpointId;
		try {
			await deleteServiceEndpoint(serviceId, endpointId);
			await loadTopology();
			if (topology && selectedService) {
				selectedService = topology.services?.find((s) => s.id === selectedService?.id) || null;
			}
		} catch (err: any) {
			alert("Failed to delete endpoint: " + (err?.message || "Unknown error"));
		} finally {
			deletingEndpointId = null;
		}
	}

	async function handleDeleteService(serviceId: string) {
		const targetSvc = topology?.services?.find((s) => s.id === serviceId) || selectedService;
		if (!confirm(`Are you sure you want to delete service "${targetSvc?.name || serviceId}" and all its endpoints?`)) return;
		deletingServiceId = serviceId;
		try {
			await deleteService(serviceId);
			if (selectedService?.id === serviceId) {
				selectedService = null;
			}
			await loadTopology();
		} catch (err: any) {
			alert("Failed to delete service: " + (err?.message || "Unknown error"));
		} finally {
			deletingServiceId = null;
		}
	}

	onMount(() => {
		void loadTopology();
		connectSSE();
	});

	onDestroy(() => {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	});
</script>

<svelte:head>
	<title>Network Topology & Troubleshooting – updu</title>
</svelte:head>

<div class="space-y-6">
	<!-- Top Header Bar -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
				<Network class="w-7 h-7 text-emerald-400" />
				Network Topology & Diagnostics
			</h1>
			<p class="text-sm text-slate-400 mt-1">
				Zone perimeters, cross-fabric reachability matrices, and hop-by-hop diagnostic autopsies.
			</p>
		</div>

		<div class="flex items-center gap-2 flex-wrap">
			<!-- View Mode Toggle: Zones | Matrix | Hop Inspector -->
			<div class="inline-flex rounded-lg bg-slate-900/80 p-1 border border-slate-800">
				<button
					type="button"
					onclick={() => (viewMode = "zones")}
					class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
						viewMode === "zones"
							? "bg-emerald-500 text-slate-950 font-semibold shadow"
							: "text-slate-400 hover:text-white"
					}`}
				>
					<Layers class="w-3.5 h-3.5" />
					Zone Overview
				</button>
				<button
					type="button"
					onclick={() => (viewMode = "matrix")}
					class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
						viewMode === "matrix"
							? "bg-emerald-500 text-slate-950 font-semibold shadow"
							: "text-slate-400 hover:text-white"
					}`}
				>
					<Grid3X3 class="w-3.5 h-3.5" />
					Reachability Matrix
				</button>
				<button
					type="button"
					onclick={() => {
						viewMode = "inspector";
						if (!selectedService && topology?.services?.length) {
							selectedService = topology.services.find((s) => s.status === "degraded" || s.status === "down") || topology.services[0];
						}
					}}
					class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
						viewMode === "inspector"
							? "bg-emerald-500 text-slate-950 font-semibold shadow"
							: "text-slate-400 hover:text-white"
					}`}
				>
					<Activity class="w-3.5 h-3.5" />
					Hop Inspector
				</button>
			</div>

			<Button variant="secondary" onclick={loadTopology} disabled={loading} class="gap-1.5 text-xs">
				<RefreshCw class={`w-3.5 h-3.5 ${loading ? "animate-spin text-emerald-400" : ""}`} />
				Refresh
			</Button>

			<Button
				variant="default"
				onclick={() => handleOpenAddService()}
				class="gap-1.5 text-xs bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold"
			>
				<Plus class="w-3.5 h-3.5" />
				Add Service
			</Button>
		</div>
	</div>

	{#if loading && !topology}
		<div class="flex flex-col items-center justify-center min-h-[400px] border border-slate-800 rounded-xl bg-slate-900/40">
			<Spinner class="w-8 h-8 text-emerald-400" />
			<p class="text-sm text-slate-400 mt-3">Synthesizing network topology graph...</p>
		</div>
	{:else if error}
		<div class="p-6 rounded-xl border border-rose-900/50 bg-rose-950/20 text-rose-300">
			<p class="font-medium">Failed to load topology</p>
			<p class="text-sm text-rose-400 mt-1">{error}</p>
			<Button variant="outline" class="mt-4 text-xs" onclick={loadTopology}>Try Again</Button>
		</div>
	{:else if topology}
		<!-- Stats Banner -->
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Prober Nodes</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.nodes?.length || 0}</div>
				</div>
				<Radio class="w-6 h-6 text-emerald-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Zones</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.zones?.length || 0}</div>
				</div>
				<Layers class="w-6 h-6 text-blue-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Services</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.services?.length || 0}</div>
				</div>
				<Server class="w-6 h-6 text-indigo-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Ingress Endpoints</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.edges?.length || 0}</div>
				</div>
				<Activity class="w-6 h-6 text-emerald-400/50" />
			</div>
		</div>

		<!-- VIEW 1: Zone Boundary Cards -->
		{#if viewMode === "zones"}
			<ZoneCardsView
				{topology}
				{probingId}
				onSelectService={handleSelectService}
				onRunProbe={runProbe}
				onAddService={handleOpenAddService}
				onAddEndpoint={(s) => handleOpenAddEndpoint(s)}
			/>
		{/if}

		<!-- VIEW 2: Reachability Matrix -->
		{#if viewMode === "matrix"}
			<ReachabilityMatrixView
				{topology}
				{probingId}
				onSelectService={handleSelectService}
				onRunProbe={runProbe}
				onAddEndpoint={handleOpenAddEndpoint}
			/>
		{/if}

		<!-- VIEW 3: Hop-Based Troubleshooting Inspector -->
		{#if viewMode === "inspector"}
			<HopInspectorView
				{topology}
				service={selectedService || (topology.services?.length ? topology.services[0] : null)}
				{focusedScopeId}
				{probingId}
				onSelectService={(s) => {
					selectedService = s;
					focusedScopeId = null;
				}}
				onRunProbe={runProbe}
				onAddEndpoint={(s) => handleOpenAddEndpoint(s)}
				onDeleteEndpoint={handleDeleteEndpoint}
				onDeleteService={handleDeleteService}
				onClose={() => (viewMode = "zones")}
			/>
		{/if}
	{/if}

	<!-- Add Service Modal -->
	<Modal
		bind:open={showAddServiceModal}
		title="Add Monitored Service"
		description="Define an application service and its primary reachability endpoint."
		size="lg"
	>
		<form onsubmit={handleCreateService} class="space-y-4">
			{#if formError}
				<div class="p-3 rounded-lg bg-rose-950/40 border border-rose-800 text-xs text-rose-300">
					{formError}
				</div>
			{/if}

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="svc-name" class="text-xs font-semibold text-slate-300">Service Name *</label>
					<input
						id="svc-name"
						type="text"
						bind:value={newServiceName}
						placeholder="e.g. Vaultwarden, Plex, Home Assistant"
						required
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>

				<div class="space-y-1.5">
					<label for="svc-zone" class="text-xs font-semibold text-slate-300">Infrastructure Zone</label>
					<select
						id="svc-zone"
						bind:value={newServiceZoneId}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					>
						{#each topology?.zones || [{ id: "default", name: "Default Zone" }] as z}
							<option value={z.id}>{z.name} ({z.id})</option>
						{/each}
					</select>
				</div>
			</div>

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="svc-type" class="text-xs font-semibold text-slate-300">Service Type</label>
					<select
						id="svc-type"
						bind:value={newServiceType}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					>
						<option value="web">Web (HTTP/HTTPS)</option>
						<option value="ping">Ping (ICMP Echo)</option>
						<option value="tcp">TCP Port</option>
						<option value="dns">DNS Resolution</option>
						<option value="tls">TLS Certificate</option>
					</select>
				</div>

				<div class="space-y-1.5">
					<label for="svc-hagroup" class="text-xs font-semibold text-slate-300">HA Cluster Group (Optional)</label>
					<input
						id="svc-hagroup"
						type="text"
						bind:value={newServiceHAGroup}
						placeholder="e.g. vaultwarden-ha, adguard-cluster"
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>
			</div>

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="svc-scope" class="text-xs font-semibold text-slate-300">Primary Endpoint Scope</label>
					<select
						id="svc-scope"
						bind:value={newServiceScopeId}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					>
						<option value="public">Public WAN (Cloudflare / Ingress)</option>
						<option value="tailnet">Tailnet (Private VPN Overlay)</option>
						<option value="lan">Local LAN (Direct Host IP)</option>
					</select>
				</div>

				<div class="space-y-1.5">
					<label for="svc-target" class="text-xs font-semibold text-slate-300">
						{newServiceType === "web" ? "Target URL *" : "Target Host / IP *"}
					</label>
					<input
						id="svc-target"
						type="text"
						bind:value={newServiceTarget}
						placeholder={newServiceType === "web" ? "https://vault.example.com" : "192.168.1.10"}
						required
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="svc-interval" class="text-xs font-semibold text-slate-300">Probe Interval (sec)</label>
					<input
						id="svc-interval"
						type="number"
						min="10"
						bind:value={newServiceInterval}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>

				<div class="space-y-1.5">
					<label for="svc-timeout" class="text-xs font-semibold text-slate-300">Timeout (sec)</label>
					<input
						id="svc-timeout"
						type="number"
						min="1"
						bind:value={newServiceTimeout}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>
			</div>

			<div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
				<Button type="button" variant="secondary" onclick={() => (showAddServiceModal = false)}>
					Cancel
				</Button>
				<Button
					type="submit"
					disabled={submitting}
					class="bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold"
				>
					{submitting ? "Creating..." : "Create Service"}
				</Button>
			</div>
		</form>
	</Modal>

	<!-- Add Vantage Endpoint Modal -->
	<Modal
		bind:open={showAddEndpointModal}
		title={selectedService ? `Add Vantage Endpoint: ${selectedService.name}` : "Add Vantage Endpoint"}
		description="Configure a vantage endpoint (e.g. LAN direct IP or Tailscale domain) for differential root-cause diagnostics."
		size="lg"
	>
		<form onsubmit={handleCreateEndpoint} class="space-y-4">
			{#if formError}
				<div class="p-3 rounded-lg bg-rose-950/40 border border-rose-800 text-xs text-rose-300">
					{formError}
				</div>
			{/if}

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="ep-name" class="text-xs font-semibold text-slate-300">Endpoint Label *</label>
					<input
						id="ep-name"
						type="text"
						bind:value={newEndpointName}
						placeholder="e.g. Tailscale Direct, LAN IP"
						required
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>

				<div class="space-y-1.5">
					<label for="ep-scope" class="text-xs font-semibold text-slate-300">Vantage Scope *</label>
					<select
						id="ep-scope"
						bind:value={newEndpointScopeId}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					>
						<option value="lan">LAN (Direct Local Network)</option>
						<option value="tailnet">Tailnet (Tailscale Mesh)</option>
						<option value="public">Public WAN (Cloudflare / Ingress)</option>
					</select>
				</div>
			</div>

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="ep-type" class="text-xs font-semibold text-slate-300">Target Type</label>
					<select
						id="ep-type"
						bind:value={newEndpointTargetType}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:border-emerald-500 focus:outline-none"
					>
						<option value="http">HTTP / HTTPS</option>
						<option value="ping">Ping (ICMP)</option>
						<option value="tcp">TCP</option>
						<option value="dns">DNS</option>
						<option value="tls">TLS</option>
					</select>
				</div>

				<div class="space-y-1.5">
					<label for="ep-target" class="text-xs font-semibold text-slate-300">
						{newEndpointTargetType === "http" ? "Target URL *" : "Target Host / IP *"}
					</label>
					<input
						id="ep-target"
						type="text"
						bind:value={newEndpointTarget}
						placeholder={newEndpointTargetType === "http" ? "http://192.168.1.50:8080" : "100.64.0.1"}
						required
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<label for="ep-interval" class="text-xs font-semibold text-slate-300">Interval (sec)</label>
					<input
						id="ep-interval"
						type="number"
						min="10"
						bind:value={newEndpointInterval}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>

				<div class="space-y-1.5">
					<label for="ep-timeout" class="text-xs font-semibold text-slate-300">Timeout (sec)</label>
					<input
						id="ep-timeout"
						type="number"
						min="1"
						bind:value={newEndpointTimeout}
						class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm font-mono text-white focus:border-emerald-500 focus:outline-none"
					/>
				</div>
			</div>

			<div class="flex items-center gap-2 pt-1">
				<input
					id="ep-primary"
					type="checkbox"
					bind:checked={newEndpointIsPrimary}
					class="rounded border-slate-800 bg-slate-950 text-emerald-500 focus:ring-emerald-500"
				/>
				<label for="ep-primary" class="text-xs text-slate-300 cursor-pointer">
					Set as primary routing endpoint for this service
				</label>
			</div>

			<div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
				<Button type="button" variant="secondary" onclick={() => (showAddEndpointModal = false)}>
					Cancel
				</Button>
				<Button
					type="submit"
					disabled={submitting}
					class="bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold"
				>
					{submitting ? "Adding..." : "Add Endpoint"}
				</Button>
			</div>
		</form>
	</Modal>
</div>
