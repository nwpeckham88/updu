<script lang="ts">
	import { onMount } from "svelte";
	import {
		getNetworkTopology,
		probeService,
		type NetworkTopology,
		type Service,
		type TopologyEdge,
		type TopologyNode,
	} from "$lib/api/services";
	import {
		Network,
		Columns3,
		RefreshCw,
		CheckCircle2,
		AlertTriangle,
		XCircle,
		ShieldAlert,
		Server,
		Radio,
		Layers,
		Activity,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";

	let topology = $state<NetworkTopology | null>(null);
	let loading = $state(true);
	let error = $state("");
	let viewMode = $state<"flow" | "swimlanes">("flow");
	let probingId = $state<string | null>(null);
	let selectedEdge = $state<TopologyEdge | null>(null);
	let selectedService = $state<Service | null>(null);

	async function loadTopology() {
		try {
			loading = true;
			error = "";
			topology = await getNetworkTopology();
		} catch (e: any) {
			error = e?.message || "Failed to load network topology";
		} finally {
			loading = false;
		}
	}

	async function runProbe(serviceId: string) {
		probingId = serviceId;
		try {
			const res = await probeService(serviceId);
			// Refresh topology to update edge latencies and diagnoses
			await loadTopology();
			if (selectedService && selectedService.id === serviceId && topology) {
				selectedService = topology.services.find((s) => s.id === serviceId) || null;
			}
		} catch (e: any) {
			alert("Probe failed: " + (e?.message || "Unknown error"));
		} finally {
			probingId = null;
		}
	}

	onMount(() => {
		void loadTopology();
	});

	function getStatusColor(status?: string): string {
		switch (status) {
			case "up":
				return "#10b981"; // emerald-500
			case "degraded":
				return "#f59e0b"; // amber-500
			case "down":
				return "#ef4444"; // rose-500
			default:
				return "#64748b"; // slate-500
		}
	}

	function getStatusBadgeClass(status?: string): string {
		switch (status) {
			case "up":
				return "bg-emerald-500/10 text-emerald-400 border-emerald-500/20";
			case "degraded":
				return "bg-amber-500/10 text-amber-400 border-amber-500/20";
			case "down":
				return "bg-rose-500/10 text-rose-400 border-rose-500/20";
			default:
				return "bg-slate-500/10 text-slate-400 border-slate-500/20";
		}
	}
</script>

<svelte:head>
	<title>Network Topology & Zones – updu</title>
</svelte:head>

<div class="space-y-6">
	<!-- Top Bar -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
				<Network class="w-7 h-7 text-emerald-400" />
				Network Topology & Zones
			</h1>
			<p class="text-sm text-slate-400 mt-1">
				Real-time multi-vantage service reachability, vantage nodes, and diagnostic correlation.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<!-- View Mode Toggle -->
			<div class="inline-flex rounded-lg bg-slate-900/80 p-1 border border-slate-800">
				<button
					onclick={() => (viewMode = "flow")}
					class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
						viewMode === "flow"
							? "bg-emerald-500 text-slate-950 font-semibold shadow"
							: "text-slate-400 hover:text-white"
					}`}
				>
					<Network class="w-3.5 h-3.5" />
					Flow Map
				</button>
				<button
					onclick={() => (viewMode = "swimlanes")}
					class={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
						viewMode === "swimlanes"
							? "bg-emerald-500 text-slate-950 font-semibold shadow"
							: "text-slate-400 hover:text-white"
					}`}
				>
					<Columns3 class="w-3.5 h-3.5" />
					Zone Swimlanes
				</button>
			</div>

			<Button variant="secondary" onclick={loadTopology} disabled={loading} class="gap-1.5 text-xs">
				<RefreshCw class={`w-3.5 h-3.5 ${loading ? "animate-spin text-emerald-400" : ""}`} />
				Refresh
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
					<div class="text-xl font-bold text-white mt-0.5">{topology.nodes.length}</div>
				</div>
				<Radio class="w-6 h-6 text-emerald-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Active Zones</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.zones.length}</div>
				</div>
				<Layers class="w-6 h-6 text-blue-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Services</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.services.length}</div>
				</div>
				<Server class="w-6 h-6 text-indigo-400/50" />
			</div>
			<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/60 flex items-center justify-between">
				<div>
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Probe Edges</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.edges.length}</div>
				</div>
				<Activity class="w-6 h-6 text-emerald-400/50" />
			</div>
		</div>

		<!-- VIEW 1: Interactive Flow Map (SVG / Canvas) -->
		{#if viewMode === "flow"}
			<div class="relative rounded-2xl border border-slate-800 bg-slate-950/80 p-6 overflow-x-auto min-h-[520px]">
				<div class="min-w-[800px] flex justify-between items-start gap-12 relative">
					<!-- Left: Probing Nodes Column -->
					<div class="w-64 space-y-4 z-10">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Radio class="w-4 h-4 text-emerald-400" />
							Prober Nodes
						</div>
						{#each topology.nodes as node}
							<div class="p-3.5 rounded-xl border border-slate-800 bg-slate-900/90 shadow-lg hover:border-slate-700 transition-all">
								<div class="flex items-center justify-between">
									<span class="font-semibold text-sm text-white">{node.name}</span>
									<span class={`text-[10px] uppercase font-bold px-2 py-0.5 rounded-full border ${node.status === "online" ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-rose-500/10 text-rose-400 border-rose-500/20"}`}>
										{node.status}
									</span>
								</div>
								<div class="flex flex-wrap gap-1 mt-2.5">
									{#each node.scopes as sc}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">
											{sc}
										</span>
									{/each}
								</div>
							</div>
						{/each}
					</div>

					<!-- Middle: Network Scopes & Dynamic Path Connectors -->
					<div class="flex-1 flex flex-col items-center justify-center space-y-8 py-8 z-10">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-500">Reachability Fabrics</div>
						<div class="flex flex-col gap-4 w-full max-w-[200px]">
							{#each topology.scopes as sc}
								<div class="p-3 rounded-lg border border-slate-800/80 bg-slate-900/40 text-center backdrop-blur shadow-sm">
									<div class="text-xs font-mono font-bold text-emerald-400 uppercase tracking-wide">
										{sc.name}
									</div>
									<div class="text-[10px] text-slate-500 mt-0.5 truncate">{sc.description || sc.id}</div>
								</div>
							{/each}
						</div>
					</div>

					<!-- Right: Services Column -->
					<div class="w-80 space-y-4 z-10">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Server class="w-4 h-4 text-indigo-400" />
							Services & Targets
						</div>
						{#each topology.services as svc}
							<div
								onclick={() => (selectedService = svc)}
								class={`p-3.5 rounded-xl border cursor-pointer transition-all ${
									selectedService?.id === svc.id
										? "border-emerald-500 bg-slate-900 shadow-md shadow-emerald-500/10"
										: "border-slate-800 bg-slate-900/90 hover:border-slate-700"
								}`}
							>
								<div class="flex items-center justify-between">
									<div class="font-semibold text-sm text-white flex items-center gap-2">
										<span class="w-2 h-2 rounded-full" style={`background-color: ${getStatusColor(svc.status)}`}></span>
										{svc.name}
									</div>
									<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(svc.status)}`}>
										{svc.status || "pending"}
									</span>
								</div>

								{#if svc.diagnosis}
									<p class="text-[11px] text-slate-400 mt-1.5 truncate">
										{svc.diagnosis}
									</p>
								{/if}

								<!-- Endpoints List -->
								<div class="mt-2.5 space-y-1">
									{#each svc.endpoints || [] as ep}
										<div class="flex items-center justify-between text-[11px] bg-slate-950/60 px-2 py-1 rounded border border-slate-800/60">
											<span class="text-slate-300 truncate max-w-[140px]">{ep.name}</span>
											<div class="flex items-center gap-1.5">
												<span class="text-[9px] font-mono text-slate-400 uppercase">{ep.scope_id}</span>
												{#if ep.last_latency_ms != null}
													<span class="font-mono text-emerald-400 font-semibold">{ep.last_latency_ms}ms</span>
												{/if}
											</div>
										</div>
									{/each}
								</div>

								<!-- Probe Action -->
								<div class="mt-3 flex justify-end">
									<button
										onclick={(e) => {
											e.stopPropagation();
											runProbe(svc.id);
										}}
										disabled={probingId === svc.id}
										class="text-xs font-medium text-emerald-400 hover:text-emerald-300 flex items-center gap-1 disabled:opacity-50"
									>
										<RefreshCw class={`w-3 h-3 ${probingId === svc.id ? "animate-spin" : ""}`} />
										Run Probe
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		{/if}

		<!-- VIEW 2: Zone Swimlanes (Column / Matrix Layout) -->
		{#if viewMode === "swimlanes"}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each topology.zones as zone}
					{@const zoneServices = topology.services.filter((s) => s.zone_id === zone.id || (zone.id === "default" && !s.zone_id))}
					<div class="rounded-2xl border border-slate-800 bg-slate-950/60 flex flex-col h-full overflow-hidden shadow-sm">
						<!-- Swimlane Header -->
						<div class="p-4 border-b border-slate-800 bg-slate-900/60 flex items-center justify-between">
							<div class="flex items-center gap-2">
								<Layers class="w-5 h-5 text-emerald-400" />
								<div>
									<h2 class="font-bold text-sm text-white">{zone.name}</h2>
									<p class="text-[11px] text-slate-400">{zone.description || "Failure domain & physical location"}</p>
								</div>
							</div>
							<span class="text-xs font-mono px-2 py-0.5 rounded-full bg-slate-800 text-slate-300">
								{zoneServices.length} {zoneServices.length === 1 ? "service" : "services"}
							</span>
						</div>

						<!-- Swimlane Body -->
						<div class="p-4 space-y-3.5 flex-1">
							{#if zoneServices.length === 0}
								<div class="text-center py-12 text-slate-500 text-xs">
									No services assigned to this zone yet.
								</div>
							{:else}
								{#each zoneServices as svc}
									<div class="p-4 rounded-xl border border-slate-800/90 bg-slate-900/90 hover:border-slate-700 transition-all shadow">
										<div class="flex items-center justify-between">
											<span class="font-bold text-sm text-white">{svc.name}</span>
											<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(svc.status)}`}>
												{svc.status || "pending"}
											</span>
										</div>

										<!-- Root Cause Diagnosis Alert Pill -->
										{#if svc.status === "degraded" || svc.status === "down"}
											<div class="mt-2.5 p-2 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-start gap-2">
												<ShieldAlert class="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
												<div class="text-[11px] text-amber-300 leading-tight">
													<span class="font-semibold">Diagnosis:</span> {svc.diagnosis || "Probe failure detected"}
												</div>
											</div>
										{/if}

										<!-- Endpoints Status Badges -->
										<div class="mt-3 flex flex-wrap gap-1.5">
											{#each svc.endpoints || [] as ep}
												<div class="text-[11px] px-2 py-1 rounded bg-slate-950 border border-slate-800 flex items-center gap-1.5">
													<span class="w-1.5 h-1.5 rounded-full" style={`background-color: ${getStatusColor(ep.status)}`}></span>
													<span class="text-slate-300 font-medium">{ep.name}</span>
													<span class="text-[9px] font-mono text-slate-500 uppercase">({ep.scope_id})</span>
													{#if ep.last_latency_ms != null}
														<span class="text-emerald-400 font-mono text-[10px] font-semibold">{ep.last_latency_ms}ms</span>
													{/if}
												</div>
											{/each}
										</div>

										<!-- Run Probe -->
										<div class="mt-3.5 pt-2.5 border-t border-slate-800/60 flex items-center justify-between text-xs">
											<span class="text-[11px] text-slate-500 font-mono">
												Type: {svc.type}
											</span>
											<button
												onclick={() => runProbe(svc.id)}
												disabled={probingId === svc.id}
												class="text-xs font-medium text-emerald-400 hover:text-emerald-300 flex items-center gap-1 disabled:opacity-50"
											>
												<RefreshCw class={`w-3 h-3 ${probingId === svc.id ? "animate-spin" : ""}`} />
												Run Probe
											</button>
										</div>
									</div>
								{/each}
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Selected Service Diagnostics Modal / Drawer -->
		{#if selectedService}
			<div class="p-6 rounded-2xl border border-slate-800 bg-slate-900/95 shadow-xl space-y-4">
				<div class="flex items-center justify-between border-b border-slate-800 pb-3">
					<div class="flex items-center gap-2.5">
						<Server class="w-5 h-5 text-emerald-400" />
						<h3 class="font-bold text-base text-white">{selectedService.name} – Diagnostic Autopsy</h3>
					</div>
					<button onclick={() => (selectedService = null)} class="text-slate-400 hover:text-white text-sm">✕ Close</button>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
						<div class="text-xs uppercase font-bold text-slate-400">Composite Health</div>
						<div class="text-lg font-bold text-white flex items-center gap-2">
							<span class="w-3 h-3 rounded-full" style={`background-color: ${getStatusColor(selectedService.status)}`}></span>
							{selectedService.status?.toUpperCase() || "PENDING"}
						</div>
						<p class="text-xs text-slate-300">{selectedService.diagnosis || "Nominal operation."}</p>
					</div>

					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
						<div class="text-xs uppercase font-bold text-slate-400">Endpoints Configured</div>
						<div class="text-lg font-bold text-white">{selectedService.endpoints?.length || 0}</div>
						<p class="text-xs text-slate-300">Vantage paths across LAN, Tailnet, and Public WAN.</p>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>
