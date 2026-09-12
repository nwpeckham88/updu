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
		Filter,
		Info,
		ArrowRight,
		Plus,
		Trash2,
		Globe,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";
	import Modal from "$lib/components/ui/modal.svelte";

	let topology = $state<NetworkTopology | null>(null);
	let loading = $state(true);
	let error = $state("");
	let viewMode = $state<"flow" | "swimlanes">("flow");
	let probingId = $state<string | null>(null);
	let selectedService = $state<Service | null>(null);

	// Modals State
	let showAddServiceModal = $state(false);
	let showAddEndpointModal = $state(false);
	let submitting = $state(false);
	let formError = $state("");

	// Add Service Form
	let newServiceName = $state("");
	let newServiceType = $state("web");
	let newServiceZoneId = $state("default");
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

	// Interactive Line Highlights & Tooltip state
	let hoveredEdgeId = $state<string | null>(null);
	let hoveredNodeId = $state<string | null>(null);
	let hoveredScopeId = $state<string | null>(null);
	let hoveredZoneId = $state<string | null>(null);
	let hoveredServiceId = $state<string | null>(null);

	// Filters
	let statusFilter = $state<string>("all");
	let scopeFilter = $state<string>("all");

	// Flow map anchor coordinates
	let containerEl = $state<HTMLElement | null>(null);
	let anchors = $state<Record<string, { x: number; y: number }>>({});
	let svgDimensions = $state({ width: 1200, height: 600 });

	async function loadTopology() {
		try {
			loading = true;
			error = "";
			topology = await getNetworkTopology();
			setTimeout(() => updateAnchors(), 50);
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

	function updateAnchors() {
		if (!containerEl) return;
		const containerRect = containerEl.getBoundingClientRect();
		const scrollLeft = containerEl.scrollLeft;
		const scrollTop = containerEl.scrollTop;

		svgDimensions = {
			width: Math.max(containerEl.scrollWidth, containerEl.clientWidth),
			height: Math.max(containerEl.scrollHeight, containerEl.clientHeight),
		};

		const newAnchors: Record<string, { x: number; y: number }> = {};
		const elements = containerEl.querySelectorAll<HTMLElement>("[data-anchor-id]");
		elements.forEach((el) => {
			const id = el.getAttribute("data-anchor-id");
			const side = el.getAttribute("data-anchor-side") || "center";
			if (!id) return;
			const rect = el.getBoundingClientRect();
			let x = rect.left - containerRect.left + scrollLeft;
			if (side === "right") {
				x = rect.right - containerRect.left + scrollLeft;
			} else if (side === "center") {
				x = rect.left + rect.width / 2 - containerRect.left + scrollLeft;
			}
			const y = rect.top + rect.height / 2 - containerRect.top + scrollTop;
			newAnchors[id] = { x, y };
		});
		anchors = newAnchors;
	}

	$effect(() => {
		if (!containerEl || viewMode !== "flow") return;

		updateAnchors();

		const ro = new ResizeObserver(() => {
			updateAnchors();
		});
		ro.observe(containerEl);

		const items = containerEl.querySelectorAll("[data-anchor-id]");
		items.forEach((item) => ro.observe(item));

		window.addEventListener("resize", updateAnchors);

		return () => {
			ro.disconnect();
			window.removeEventListener("resize", updateAnchors);
		};
	});

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

		eventSource.addEventListener("monitor:status", () => {
			setTimeout(() => updateAnchors(), 50);
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

	async function handleDeleteEndpoint(endpointId: string) {
		if (!selectedService) return;
		if (!confirm("Are you sure you want to delete this endpoint?")) return;
		deletingEndpointId = endpointId;
		try {
			await deleteServiceEndpoint(selectedService.id, endpointId);
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

	async function handleDeleteService() {
		if (!selectedService) return;
		if (!confirm(`Are you sure you want to delete service "${selectedService.name}" and all its vantage endpoints?`)) return;
		deletingServiceId = selectedService.id;
		try {
			await deleteService(selectedService.id);
			selectedService = null;
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

	const filteredEdges = $derived(
		(topology?.edges || []).filter((e) => {
			if (statusFilter !== "all" && e.status !== statusFilter) return false;
			if (scopeFilter !== "all" && e.scope_id !== scopeFilter) return false;
			return true;
		})
	);

	const activeHoveredEdge = $derived(
		hoveredEdgeId ? topology?.edges?.find((e) => e.id === hoveredEdgeId) : null
	);

	function getPathForEdge(edge: TopologyEdge): string | null {
		const nodeAnchor = anchors[`node-out-${edge.node_id}`];
		const scopeInAnchor = anchors[`scope-in-${edge.scope_id}`];
		const scopeOutAnchor = anchors[`scope-out-${edge.scope_id}`];
		const zoneId = edge.zone_id || (topology?.services?.find((s) => s.id === edge.service_id)?.zone_id) || "default";
		const zoneInAnchor = anchors[`zone-in-${zoneId}`];
		const zoneOutAnchor = anchors[`zone-out-${zoneId}`];
		const serviceAnchor = anchors[`endpoint-in-${edge.endpoint_id}`] || anchors[`service-in-${edge.service_id}`];

		if (!nodeAnchor || !scopeInAnchor || !scopeOutAnchor || !zoneInAnchor || !zoneOutAnchor || !serviceAnchor) {
			return null;
		}

		const dx1 = Math.abs(scopeInAnchor.x - nodeAnchor.x) * 0.45;
		const dx2 = Math.abs(zoneInAnchor.x - scopeOutAnchor.x) * 0.45;
		const dx3 = Math.abs(serviceAnchor.x - zoneOutAnchor.x) * 0.45;

		return (
			`M ${nodeAnchor.x} ${nodeAnchor.y} ` +
			`C ${nodeAnchor.x + dx1} ${nodeAnchor.y}, ${scopeInAnchor.x - dx1} ${scopeInAnchor.y}, ${scopeInAnchor.x} ${scopeInAnchor.y} ` +
			`L ${scopeOutAnchor.x} ${scopeOutAnchor.y} ` +
			`C ${scopeOutAnchor.x + dx2} ${scopeOutAnchor.y}, ${zoneInAnchor.x - dx2} ${zoneInAnchor.y}, ${zoneInAnchor.x} ${zoneInAnchor.y} ` +
			`L ${zoneOutAnchor.x} ${zoneOutAnchor.y} ` +
			`C ${zoneOutAnchor.x + dx3} ${zoneOutAnchor.y}, ${serviceAnchor.x - dx3} ${serviceAnchor.y}, ${serviceAnchor.x} ${serviceAnchor.y}`
		);
	}

	function isEdgeHighlighted(edge: TopologyEdge): boolean {
		if (hoveredEdgeId) return hoveredEdgeId === edge.id;
		if (hoveredNodeId) return hoveredNodeId === edge.node_id;
		if (hoveredScopeId) return hoveredScopeId === edge.scope_id;
		if (hoveredZoneId) {
			const zoneId = edge.zone_id || (topology?.services?.find((s) => s.id === edge.service_id)?.zone_id) || "default";
			return hoveredZoneId === zoneId;
		}
		if (hoveredServiceId) return hoveredServiceId === edge.service_id;
		return false;
	}

	function hasAnyHover(): boolean {
		return !!(hoveredEdgeId || hoveredNodeId || hoveredScopeId || hoveredZoneId || hoveredServiceId);
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
				Real-time multi-vantage service reachability, zone traversal flows, and diagnostic autopsy.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<!-- View Mode Toggle -->
			<div class="inline-flex rounded-lg bg-slate-900/80 p-1 border border-slate-800">
				<button
					type="button"
					onclick={() => {
						viewMode = "flow";
						setTimeout(() => updateAnchors(), 50);
					}}
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
					type="button"
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

			<Button
				variant="default"
				onclick={() => {
					formError = "";
					showAddServiceModal = true;
				}}
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
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Active Zones</span>
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
					<span class="text-xs text-slate-400 uppercase tracking-wider font-semibold">Active Probes</span>
					<div class="text-xl font-bold text-white mt-0.5">{topology.edges?.length || 0}</div>
				</div>
				<Activity class="w-6 h-6 text-emerald-400/50" />
			</div>
		</div>

		<!-- VIEW 1: Interactive Flow Map with Probe Lines -->
		{#if viewMode === "flow"}
			<!-- Filter & Probe Inspector HUD -->
			<div class="rounded-2xl border border-slate-800 bg-slate-900/80 p-4 space-y-3">
				<div class="flex flex-wrap items-center justify-between gap-3 text-xs">
					<!-- Filters -->
					<div class="flex flex-wrap items-center gap-2">
						<span class="text-slate-400 font-semibold flex items-center gap-1">
							<Filter class="w-3.5 h-3.5 text-slate-400" />
							Filter:
						</span>
						<div class="inline-flex rounded-lg bg-slate-950 p-0.5 border border-slate-800">
							{#each ["all", "up", "degraded", "down"] as st}
								<button
									type="button"
									onclick={() => (statusFilter = st)}
									class={`px-2.5 py-1 rounded-md text-[11px] font-medium uppercase tracking-wider transition-all ${
										statusFilter === st
											? "bg-slate-800 text-white font-bold shadow-sm"
											: "text-slate-400 hover:text-slate-200"
									}`}
								>
									{st}
								</button>
							{/each}
						</div>

						<div class="inline-flex rounded-lg bg-slate-950 p-0.5 border border-slate-800">
							{#each ["all", "lan", "tailnet", "public"] as sc}
								<button
									type="button"
									onclick={() => (scopeFilter = sc)}
									class={`px-2.5 py-1 rounded-md text-[11px] font-medium uppercase tracking-wider transition-all ${
										scopeFilter === sc
											? "bg-slate-800 text-white font-bold shadow-sm"
											: "text-slate-400 hover:text-slate-200"
									}`}
								>
									{sc}
								</button>
							{/each}
						</div>
					</div>

					<div class="text-[11px] text-slate-400 flex items-center gap-1.5">
						<span class="inline-block w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
						Live Probing Topology • Hover elements to trace probe paths
					</div>
				</div>

				<!-- Dynamic Probe Trace Breadcrumb HUD -->
				{#if activeHoveredEdge}
					{@const hEdge = activeHoveredEdge}
					{@const hNode = topology?.nodes?.find((n) => n.id === hEdge.node_id)}
					{@const hScope = topology?.scopes?.find((s) => s.id === hEdge.scope_id)}
					{@const hZoneId = hEdge.zone_id || (topology?.services?.find((s) => s.id === hEdge.service_id)?.zone_id) || "default"}
					{@const hZone = topology?.zones?.find((z) => z.id === hZoneId)}
					{@const hService = topology?.services?.find((s) => s.id === hEdge.service_id)}
					{@const hEndpoint = hService?.endpoints?.find((e) => e.id === hEdge.endpoint_id)}

					<div class="p-3 rounded-xl bg-slate-950 border border-slate-800 flex flex-wrap items-center justify-between gap-3 animate-fade-in">
						<div class="flex items-center gap-2 text-xs font-mono">
							<span class="px-2 py-0.5 rounded bg-slate-800 text-slate-200 font-bold">
								{hNode?.name || hEdge.node_id}
							</span>
							<ArrowRight class="w-3.5 h-3.5 text-slate-500" />
							<span class="px-2 py-0.5 rounded bg-emerald-950/60 border border-emerald-800/60 text-emerald-400 font-bold uppercase">
								{hScope?.name || hEdge.scope_id}
							</span>
							<ArrowRight class="w-3.5 h-3.5 text-slate-500" />
							<span class="px-2 py-0.5 rounded bg-blue-950/60 border border-blue-800/60 text-blue-400 font-bold">
								{hZone?.name || hZoneId}
							</span>
							<ArrowRight class="w-3.5 h-3.5 text-slate-500" />
							<span class="px-2 py-0.5 rounded bg-indigo-950/60 border border-indigo-800/60 text-indigo-300 font-bold">
								{hService?.name || hEdge.service_id}
							</span>
							{#if hEndpoint}
								<span class="text-slate-400 font-normal">({hEndpoint.name})</span>
							{/if}
						</div>

						<div class="flex items-center gap-2 text-xs">
							<span class={`font-bold px-2.5 py-0.5 rounded-full border ${getStatusBadgeClass(hEdge.status)}`}>
								{hEdge.status.toUpperCase()}
								{#if hEdge.latency_ms != null}
									• {hEdge.latency_ms}ms
								{/if}
							</span>
							{#if hEdge.message}
								<span class="text-slate-400 max-w-xs truncate">{hEdge.message}</span>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<!-- Visual Flow Map Container -->
			<div
				bind:this={containerEl}
				onscroll={updateAnchors}
				class="relative rounded-2xl border border-slate-800 bg-slate-950/90 p-6 overflow-x-auto min-h-[580px]"
			>
				<!-- SVG Connectors Layer -->
				<svg
					class="absolute inset-0 pointer-events-none z-0 overflow-visible"
					width={svgDimensions.width}
					height={svgDimensions.height}
				>
					<defs>
						<filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
							<feGaussianBlur stdDeviation="3" result="blur" />
							<feComposite in="SourceGraphic" in2="blur" operator="over" />
						</filter>
					</defs>

					{#each filteredEdges as edge (edge.id)}
						{@const path = getPathForEdge(edge)}
						{#if path}
							{@const isHovered = isEdgeHighlighted(edge)}
							{@const isDimmed = hasAnyHover() && !isHovered}
							{@const strokeColor = getStatusColor(edge.status)}

							<!-- Invisible wide interaction stroke -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<path
								d={path}
								fill="none"
								stroke="transparent"
								stroke-width="16"
								class="pointer-events-auto cursor-pointer"
								onmouseenter={() => (hoveredEdgeId = edge.id)}
								onmouseleave={() => (hoveredEdgeId = null)}
							/>

							<!-- Visible probe flow path -->
							<path
								d={path}
								fill="none"
								stroke={strokeColor}
								stroke-width={isHovered ? 3.5 : 2}
								stroke-opacity={isDimmed ? 0.12 : (isHovered ? 1 : 0.75)}
								filter={isHovered ? "url(#glow)" : undefined}
								class={edge.status === "up" ? "flow-line-animated" : (edge.status === "degraded" ? "flow-line-pulse" : "")}
							/>
						{/if}
					{/each}
				</svg>

				<!-- 4-Tier Topology Layout Grid -->
				<div class="min-w-[1100px] flex justify-between items-start gap-8 relative z-10">
					<!-- Tier 1: Prober Nodes (Origin) -->
					<div class="w-60 space-y-3.5 shrink-0">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Radio class="w-4 h-4 text-emerald-400" />
							Prober Nodes
						</div>
						{#each topology?.nodes || [] as node}
							<div
								data-anchor-id={"node-out-" + node.id}
								data-anchor-side="right"
								role="region"
								aria-label={node.name}
								onmouseenter={() => (hoveredNodeId = node.id)}
								onmouseleave={() => (hoveredNodeId = null)}
								class={`p-3.5 rounded-xl border transition-all relative ${
									hoveredNodeId === node.id
										? "border-emerald-500 bg-slate-900 shadow-lg shadow-emerald-500/10"
										: "border-slate-800 bg-slate-900/95 hover:border-slate-700"
								}`}
							>
								<div class="flex items-center justify-between">
									<span class="font-bold text-sm text-white flex items-center gap-1.5">
										<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
										{node.name}
									</span>
									{#if node.is_local}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-emerald-950/80 text-emerald-400 border border-emerald-800/60 font-semibold uppercase">
											Origin
										</span>
									{:else}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 uppercase">
											Peer
										</span>
									{/if}
								</div>
								<div class="mt-2 flex flex-wrap gap-1">
									{#each node.scopes || [] as sc}
										<span class="text-[9px] font-mono px-1.5 py-0.5 rounded bg-slate-800/80 text-slate-300">
											{sc}
										</span>
									{/each}
								</div>

								<!-- Outbound anchor pin -->
								<div class="absolute -right-1.5 top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-emerald-400 border-2 border-slate-900"></div>
							</div>
						{/each}
					</div>

					<!-- Tier 2: Reachability Scopes (Network Fabrics) -->
					<div class="w-48 space-y-3.5 shrink-0">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Network class="w-4 h-4 text-emerald-400" />
							Transit Fabrics
						</div>
						{#each topology?.scopes || [] as sc}
							{@const scopeProbes = (topology?.edges || []).filter((e) => e.scope_id === sc.id)}
							<div
								role="region"
								aria-label={sc.name}
								onmouseenter={() => (hoveredScopeId = sc.id)}
								onmouseleave={() => (hoveredScopeId = null)}
								class={`p-3.5 rounded-xl border transition-all relative ${
									hoveredScopeId === sc.id
										? "border-emerald-500 bg-slate-900 shadow-lg shadow-emerald-500/10"
										: "border-slate-800 bg-slate-900/95 hover:border-slate-700"
								}`}
							>
								<!-- Inbound anchor pin -->
								<div
									data-anchor-id={"scope-in-" + sc.id}
									data-anchor-side="left"
									class="absolute -left-1.5 top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-emerald-400 border-2 border-slate-900"
								></div>

								<div class="text-xs font-mono font-bold text-emerald-400 uppercase tracking-wide">
									{sc.name}
								</div>
								<div class="text-[11px] text-slate-400 mt-1 truncate">{sc.description || sc.id}</div>
								<div class="mt-2.5 flex items-center justify-between text-[10px] text-slate-400">
									<span>Probes:</span>
									<span class="font-mono font-bold text-white px-1.5 py-0.5 rounded bg-slate-800">
										{scopeProbes.length}
									</span>
								</div>

								<!-- Outbound anchor pin -->
								<div
									data-anchor-id={"scope-out-" + sc.id}
									data-anchor-side="right"
									class="absolute -right-1.5 top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-emerald-400 border-2 border-slate-900"
								></div>
							</div>
						{/each}
					</div>

					<!-- Tier 3: Target Zones (Failure Domains) -->
					<div class="w-52 space-y-3.5 shrink-0">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Layers class="w-4 h-4 text-blue-400" />
							Target Zones
						</div>
						{#each topology?.zones || [] as zone}
							{@const zoneServices = (topology?.services || []).filter((s) => s.zone_id === zone.id || (zone.id === "default" && !s.zone_id))}
							<div
								role="region"
								aria-label={zone.name}
								onmouseenter={() => (hoveredZoneId = zone.id)}
								onmouseleave={() => (hoveredZoneId = null)}
								class={`p-3.5 rounded-xl border transition-all relative ${
									hoveredZoneId === zone.id
										? "border-blue-500 bg-slate-900 shadow-lg shadow-blue-500/10"
										: "border-slate-800 bg-slate-900/95 hover:border-slate-700"
								}`}
							>
								<!-- Inbound anchor pin -->
								<div
									data-anchor-id={"zone-in-" + zone.id}
									data-anchor-side="left"
									class="absolute -left-1.5 top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-blue-400 border-2 border-slate-900"
								></div>

								<div class="text-xs font-bold text-white flex items-center justify-between">
									<span>{zone.name}</span>
									<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-blue-400">
										{zoneServices.length} svc
									</span>
								</div>
								<div class="text-[11px] text-slate-400 mt-1 truncate">
									{zone.description || "Zone domain"}
								</div>

								<!-- Outbound anchor pin -->
								<div
									data-anchor-id={"zone-out-" + zone.id}
									data-anchor-side="right"
									class="absolute -right-1.5 top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-blue-400 border-2 border-slate-900"
								></div>
							</div>
						{/each}
					</div>

					<!-- Tier 4: Services & Endpoints (Destination Targets) -->
					<div class="w-80 space-y-4 shrink-0">
						<div class="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5 pb-2 border-b border-slate-800">
							<Server class="w-4 h-4 text-indigo-400" />
							Services & Targets
						</div>
						{#each topology?.services || [] as svc}
							<div
								data-anchor-id={"service-in-" + svc.id}
								data-anchor-side="left"
								role="button"
								tabindex="0"
								onmouseenter={() => (hoveredServiceId = svc.id)}
								onmouseleave={() => (hoveredServiceId = null)}
								onclick={() => (selectedService = svc)}
								onkeydown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										selectedService = svc;
									}
								}}
								class={`p-3.5 rounded-xl border cursor-pointer transition-all relative ${
									selectedService?.id === svc.id || hoveredServiceId === svc.id
										? "border-emerald-500 bg-slate-900 shadow-md shadow-emerald-500/10"
										: "border-slate-800 bg-slate-900/95 hover:border-slate-700"
								}`}
							>
								<!-- Inbound anchor pin -->
								<div class="absolute -left-1.5 top-4 w-3 h-3 rounded-full bg-indigo-400 border-2 border-slate-900"></div>

								<div class="flex items-center justify-between">
									<div class="font-semibold text-sm text-white flex items-center gap-2">
										<span class="w-2 h-2 rounded-full" style={`background-color: ${getStatusColor(svc.status)}`}></span>
										{svc.name}
									</div>
									<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(svc.status)}`}>
										{svc.status || "pending"}
									</span>
								</div>

								<div class="mt-1 flex items-center gap-2 text-[11px] text-slate-400">
									<span class="text-slate-500">Zone:</span>
									<span class="font-mono text-slate-300">{svc.zone_id || "default"}</span>
								</div>

								{#if svc.diagnosis}
									<p class="text-[11px] text-slate-400 mt-1.5 truncate">
										{svc.diagnosis}
									</p>
								{/if}

								<!-- Endpoints List -->
								<div class="mt-2.5 space-y-1">
									{#each svc.endpoints || [] as ep}
										<div
											data-anchor-id={"endpoint-in-" + ep.id}
											data-anchor-side="left"
											class="flex items-center justify-between text-[11px] bg-slate-950/80 px-2 py-1 rounded border border-slate-800/80 relative"
										>
											<span class="text-slate-300 truncate max-w-[130px]">{ep.name}</span>
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
										type="button"
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
				{#each topology?.zones || [] as zone}
					{@const zoneServices = (topology?.services || []).filter((s) => s.zone_id === zone.id || (zone.id === "default" && !s.zone_id))}
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
												type="button"
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
			<div class="p-6 rounded-2xl border border-slate-800 bg-slate-900/95 shadow-xl space-y-5">
				<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800 pb-4">
					<div class="flex items-center gap-3">
						<Server class="w-6 h-6 text-emerald-400" />
						<div>
							<div class="flex items-center gap-2">
								<h3 class="font-bold text-lg text-white">{selectedService.name}</h3>
								<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(selectedService.status)}`}>
									{selectedService.status?.toUpperCase() || "PENDING"}
								</span>
							</div>
							<p class="text-xs text-slate-400 mt-0.5">
								Zone: <span class="font-mono text-slate-300">{selectedService.zone_id || "default"}</span> • Type: <span class="font-mono text-slate-300">{selectedService.type}</span>
							</p>
						</div>
					</div>

					<div class="flex items-center gap-2 flex-wrap">
						<Button
							variant="secondary"
							onclick={() => runProbe(selectedService!.id)}
							disabled={probingId === selectedService.id}
							class="gap-1 text-xs"
						>
							<RefreshCw class={`w-3.5 h-3.5 ${probingId === selectedService.id ? "animate-spin text-emerald-400" : ""}`} />
							Run Full Probe
						</Button>

						<Button
							variant="secondary"
							onclick={() => {
								formError = "";
								newEndpointName = "";
								newEndpointTarget = "http://";
								newEndpointIsPrimary = false;
								showAddEndpointModal = true;
							}}
							class="gap-1 text-xs bg-slate-800 hover:bg-slate-700 text-white"
						>
							<Plus class="w-3.5 h-3.5 text-emerald-400" />
							Add Vantage Endpoint
						</Button>

						<Button
							variant="outline"
							onclick={handleDeleteService}
							disabled={deletingServiceId === selectedService.id}
							class="gap-1 text-xs text-rose-400 border-rose-900/40 hover:bg-rose-950/40"
						>
							<Trash2 class="w-3.5 h-3.5" />
							Delete Service
						</Button>

						<button
							type="button"
							onclick={() => (selectedService = null)}
							class="text-slate-400 hover:text-white px-2 py-1 text-sm rounded-lg hover:bg-slate-800"
							aria-label="Close autopsy drawer"
						>
							✕ Close
						</button>
					</div>
				</div>

				<!-- Diagnostic Summary Cards -->
				<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-1.5">
						<div class="text-[11px] uppercase font-bold text-slate-400">Composite Health</div>
						<div class="text-base font-bold text-white flex items-center gap-2">
							<span class="w-2.5 h-2.5 rounded-full" style={`background-color: ${getStatusColor(selectedService.status)}`}></span>
							{selectedService.status?.toUpperCase() || "PENDING"}
						</div>
						<p class="text-xs text-slate-400">{selectedService.diagnosis || "Nominal operation across all vantage points."}</p>
					</div>

					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-1.5">
						<div class="text-[11px] uppercase font-bold text-slate-400">Vantage Points</div>
						<div class="text-base font-bold text-white">{selectedService.endpoints?.length || 0} active</div>
						<p class="text-xs text-slate-400">Probing across LAN, Tailnet, and Public WAN ingress scopes.</p>
					</div>

					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-1.5">
						<div class="text-[11px] uppercase font-bold text-slate-400">Primary Latency</div>
						<div class="text-base font-bold font-mono text-emerald-400">
							{selectedService.primary_latency_ms != null ? `${selectedService.primary_latency_ms}ms` : "—"}
						</div>
						<p class="text-xs text-slate-400">End-to-end response time on primary routing path.</p>
					</div>
				</div>

				<!-- Endpoints Detailed List -->
				<div class="space-y-2.5">
					<div class="flex items-center justify-between">
						<h4 class="text-xs font-bold uppercase tracking-wider text-slate-400">Configured Vantage Endpoints</h4>
						<span class="text-[11px] text-slate-500 font-mono">Differential matrix evaluates ingress failure domains</span>
					</div>

					{#if (selectedService.endpoints || []).length === 0}
						<div class="p-8 rounded-xl bg-slate-950/60 border border-dashed border-slate-800 text-center text-slate-500 text-xs">
							No endpoints configured yet. Click "+ Add Vantage Endpoint" to attach a LAN, Tailnet, or Public probe.
						</div>
					{:else}
						<div class="divide-y divide-slate-800/80 rounded-xl bg-slate-950 border border-slate-800 overflow-hidden">
							{#each selectedService.endpoints || [] as ep (ep.id)}
								<div class="p-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-900/40 transition-colors">
									<div class="flex items-start sm:items-center gap-3 min-w-0">
										<span class="w-2 h-2 rounded-full shrink-0 mt-1 sm:mt-0" style={`background-color: ${getStatusColor(ep.status)}`}></span>
										<div class="min-w-0">
											<div class="flex items-center gap-2 flex-wrap">
												<span class="font-bold text-sm text-white truncate">{ep.name}</span>
												{#if ep.is_primary}
													<span class="text-[9px] font-mono font-bold px-1.5 py-0.5 rounded bg-emerald-950/80 text-emerald-400 border border-emerald-800/60 uppercase">
														Primary
													</span>
												{/if}
												<span class="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-blue-400 uppercase font-semibold">
													{ep.scope_id}
												</span>
												<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 uppercase">
													{ep.target_type}
												</span>
											</div>
											<div class="text-xs font-mono text-slate-400 mt-1 truncate">
												{ep.config?.url || ep.config?.host || JSON.stringify(ep.config)}
											</div>
											{#if ep.last_message}
												<div class="text-[11px] text-rose-400 mt-1 truncate max-w-xl">
													{ep.last_message}
												</div>
											{/if}
										</div>
									</div>

									<div class="flex items-center gap-4 shrink-0 justify-between sm:justify-end">
										<div class="text-right">
											<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(ep.status)}`}>
												{ep.status?.toUpperCase() || "PENDING"}
											</span>
											{#if ep.last_latency_ms != null}
												<div class="text-[11px] font-mono text-emerald-400 font-semibold mt-1">
													{ep.last_latency_ms}ms
												</div>
											{/if}
										</div>

										<button
											type="button"
											onclick={() => handleDeleteEndpoint(ep.id)}
											disabled={deletingEndpointId === ep.id}
											class="p-1.5 text-slate-500 hover:text-rose-400 transition-colors rounded-lg hover:bg-slate-800"
											title="Delete endpoint"
											aria-label={`Delete endpoint ${ep.name}`}
										>
											<Trash2 class="w-4 h-4" />
										</button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
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

<style>
	@keyframes flowDash {
		to {
			stroke-dashoffset: -20;
		}
	}
	@keyframes flowPulseSlow {
		0%, 100% {
			stroke-opacity: 0.6;
		}
		50% {
			stroke-opacity: 1;
		}
	}
	.flow-line-animated {
		stroke-dasharray: 6 4;
		animation: flowDash 1.2s linear infinite;
	}
	.flow-line-pulse {
		stroke-dasharray: 4 4;
		animation: flowDash 2s linear infinite, flowPulseSlow 2s ease-in-out infinite;
	}
</style>
