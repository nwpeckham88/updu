<script lang="ts">
	import type { NetworkTopology, Service, ServiceEndpoint, HopTrace } from "$lib/api/services";
	import {
		Activity,
		RefreshCw,
		Plus,
		Trash2,
		ShieldAlert,
		CheckCircle2,
		AlertTriangle,
		XCircle,
		ArrowRight,
		Server,
		Radio,
		Network,
		Lock,
		Cpu,
		Globe,
		GitBranch,
		Layers,
		Check,
		Clock,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";

	interface Props {
		topology: NetworkTopology;
		service: Service | null;
		focusedScopeId?: string | null;
		probingId: string | null;
		onSelectService: (service: Service) => void;
		onRunProbe: (serviceId: string) => void;
		onAddEndpoint: (service: Service) => void;
		onDeleteEndpoint: (serviceId: string, endpointId: string) => void;
		onDeleteService: (serviceId: string) => void;
		onClose?: () => void;
	}

	let {
		topology,
		service,
		focusedScopeId = null,
		probingId,
		onSelectService,
		onRunProbe,
		onAddEndpoint,
		onDeleteEndpoint,
		onDeleteService,
		onClose,
	}: Props = $props();

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

	function getScopeBadge(scopeId: string): { label: string; class: string } {
		switch (scopeId) {
			case "lan":
				return { label: "Local LAN (RFC1918)", class: "text-emerald-400 bg-emerald-950/60 border-emerald-800/60" };
			case "tailnet":
				return { label: "Tailscale Overlay (WireGuard)", class: "text-indigo-400 bg-indigo-950/60 border-indigo-800/60" };
			case "public":
				return { label: "Public WAN (Internet)", class: "text-sky-400 bg-sky-950/60 border-sky-800/60" };
			default:
				return { label: scopeId.toUpperCase(), class: "text-slate-300 bg-slate-900 border-slate-800" };
		}
	}

	function getEndpointTrace(ep: ServiceEndpoint): HopTrace | null {
		if (!ep.last_metadata) return null;
		const meta = ep.last_metadata as any;
		return (meta?.trace as HopTrace) || null;
	}

	function getHAPeers(svc: Service): Service[] {
		if (!svc.ha_group) return [];
		return (topology?.services || []).filter(
			(s) => s.id !== svc.id && s.ha_group === svc.ha_group
		);
	}
</script>

<div class="space-y-6">
	<!-- Top Bar & Service Selector -->
	<div class="rounded-2xl border border-slate-800 bg-slate-950/90 p-4 sm:p-5 shadow-xl">
		<div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
			<div class="flex items-center gap-3">
				<div class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 shrink-0">
					<Activity class="w-6 h-6" />
				</div>
				<div>
					<div class="flex items-center gap-2 flex-wrap">
						<span class="text-xs text-slate-400 font-semibold uppercase tracking-wider">Troubleshooting & Hop Inspector</span>
						{#if service?.ha_group}
							<span class="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded bg-indigo-950/80 text-indigo-300 border border-indigo-800/60">
								<GitBranch class="w-3 h-3 text-indigo-400" />
								HA: {service.ha_group}
							</span>
						{/if}
					</div>

					<!-- Service Selection Dropdown -->
					<div class="mt-1 flex items-center gap-3">
						<select
							class="bg-slate-900 border border-slate-700 text-white font-bold text-base rounded-xl px-3 py-1.5 focus:outline-none focus:border-emerald-500"
							value={service?.id || ""}
							onchange={(e) => {
								const targetId = (e.target as HTMLSelectElement).value;
								const found = (topology?.services || []).find((s) => s.id === targetId);
								if (found) onSelectService(found);
							}}
						>
							{#each topology?.services || [] as s}
								<option value={s.id}>
									{s.name} ({s.zone_id || "default"}) [{s.status?.toUpperCase() || "PENDING"}]
								</option>
							{/each}
						</select>

						{#if service}
							<span class={`text-xs font-bold px-2.5 py-1 rounded-full border ${getStatusBadgeClass(service.status)}`}>
								{service.status?.toUpperCase() || "PENDING"}
							</span>
						{/if}
					</div>
				</div>
			</div>

			<!-- Actions -->
			{#if service}
				<div class="flex items-center gap-2 flex-wrap">
					<Button
						variant="default"
						onclick={() => onRunProbe(service!.id)}
						disabled={probingId === service.id}
						class="gap-1.5 text-xs bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold"
					>
						<RefreshCw class={`w-3.5 h-3.5 ${probingId === service.id ? "animate-spin" : ""}`} />
						Run Probe Now
					</Button>

					<Button
						variant="secondary"
						onclick={() => onAddEndpoint(service!)}
						class="gap-1 text-xs bg-slate-800 hover:bg-slate-700 text-white"
					>
						<Plus class="w-3.5 h-3.5 text-emerald-400" />
						Add Scope Endpoint
					</Button>

					<Button
						variant="outline"
						onclick={() => onDeleteService(service!.id)}
						class="gap-1 text-xs text-rose-400 border-rose-900/40 hover:bg-rose-950/40"
					>
						<Trash2 class="w-3.5 h-3.5" />
						Delete
					</Button>

					{#if onClose}
						<button
							type="button"
							onclick={onClose}
							class="text-slate-400 hover:text-white px-2 py-1 text-sm rounded-lg hover:bg-slate-800 transition-colors"
						>
							✕ Close
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	{#if !service}
		<div class="p-12 text-center rounded-2xl border border-slate-800 bg-slate-900/30 text-slate-500 text-sm">
			Select a service above to inspect its hop-by-hop reachability pipelines.
		</div>
	{:else}
		<!-- Diagnostic Autopsy Summary Banner -->
		{#if service.status === "degraded" || service.status === "down" || service.diagnosis}
			<div class="rounded-2xl border border-amber-500/30 bg-amber-950/20 p-5 shadow-lg space-y-3 animate-fade-in">
				<div class="flex items-start justify-between gap-3">
					<div class="flex items-start gap-3">
						<div class="p-2 rounded-xl bg-amber-500/20 text-amber-400 shrink-0 mt-0.5">
							<ShieldAlert class="w-6 h-6" />
						</div>
						<div>
							<h3 class="font-bold text-white text-base">Diagnostic Autopsy: {service.name}</h3>
							<p class="text-xs text-amber-300 font-medium mt-0.5">
								{service.diagnosis || "Probe discrepancy detected across reachability scopes."}
							</p>
						</div>
					</div>
					<span class="text-xs font-mono px-2 py-0.5 rounded bg-amber-950/80 border border-amber-800/80 text-amber-400">
						Differential Root-Cause
					</span>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs pt-1">
					<div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 space-y-1">
						<span class="text-[11px] font-semibold text-slate-400 uppercase tracking-wide">Probable Cause</span>
						<p class="text-slate-300">
							Backend responded on some scopes (e.g. LAN/Tailnet), but external or local routing failed on others.
						</p>
					</div>

					<div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 space-y-1">
						<span class="text-[11px] font-semibold text-emerald-400 uppercase tracking-wide">Actionable Recommendation</span>
						<p class="text-slate-300">
							Check reverse proxy (Caddy / Traefik), Cloudflare tunnel / WAN port forward, or host firewall bindings.
						</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- HA Cluster Peers Summary (if in HA Group) -->
		{@const haPeers = getHAPeers(service)}
		{#if service.ha_group && haPeers.length > 0}
			<div class="rounded-2xl border border-indigo-900/50 bg-indigo-950/20 p-4 space-y-2">
				<div class="flex items-center justify-between text-xs">
					<div class="flex items-center gap-2 font-bold text-white">
						<GitBranch class="w-4 h-4 text-indigo-400" />
						<span>High Availability Cluster: {service.ha_group}</span>
					</div>
					<span class="text-[11px] font-mono text-indigo-300">
						{haPeers.length + 1} Replicas across zones
					</span>
				</div>

				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 text-xs pt-1">
					<!-- Current Service Replica -->
					<div class="p-3 rounded-xl bg-slate-950/80 border border-indigo-500/40 space-y-1">
						<div class="flex items-center justify-between">
							<span class="font-bold text-white">{service.name} (Active View)</span>
							<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(service.status)}`}>
								{service.status?.toUpperCase()}
							</span>
						</div>
						<div class="text-[11px] text-slate-400 font-mono">Zone: {service.zone_id || "default"}</div>
					</div>

					<!-- Peer Replicas -->
					{#each haPeers as peer}
						<div
							role="button"
							tabindex="0"
							onclick={() => onSelectService(peer)}
							onkeydown={(e) => {
								if (e.key === "Enter" || e.key === " ") onSelectService(peer);
							}}
							class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 hover:border-indigo-500 transition-all cursor-pointer space-y-1"
						>
							<div class="flex items-center justify-between">
								<span class="font-bold text-slate-200 hover:text-indigo-300">{peer.name}</span>
								<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${getStatusBadgeClass(peer.status)}`}>
									{peer.status?.toUpperCase()}
								</span>
							</div>
							<div class="text-[11px] text-slate-400 font-mono">Zone: {peer.zone_id || "default"} (Click to inspect)</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Multi-Vantage Hop Pipelines -->
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<h3 class="font-bold text-base text-white flex items-center gap-2">
					<Network class="w-5 h-5 text-emerald-400" />
					Multi-Vantage Reachability Pipelines ({service.endpoints?.length || 0} Scopes)
				</h3>
				<span class="text-xs text-slate-400">
					Sequential Hop-by-Hop Stage Attribution
				</span>
			</div>

			{#if !service.endpoints || service.endpoints.length === 0}
				<div class="p-8 text-center rounded-xl border border-slate-800 bg-slate-900/40 text-slate-400 text-xs">
					No endpoints configured for this service. Click "Add Scope Endpoint" to add a LAN, Tailnet, or Public WAN ingress check.
				</div>
			{:else}
				{#each service.endpoints as ep (ep.id)}
					{@const trace = getEndpointTrace(ep)}
					{@const scopeMeta = getScopeBadge(ep.scope_id)}
					{@const isFailing = ep.status === "down" || ep.status === "degraded"}
					{@const failingHop = trace?.failing_hop || (isFailing ? "app" : null)}
					{@const isFocused = focusedScopeId === ep.scope_id}
					{@const isDnsFailing = failingHop === "dns"}
					{@const isIngressFailing = failingHop === "tcp" || failingHop === "tls" || failingHop === "ingress"}
					{@const isAppFailing = failingHop === "app"}

					<div
						class={`rounded-2xl border transition-all p-5 space-y-4 shadow-md ${
							isFocused
								? "border-emerald-500 bg-slate-900/95 ring-1 ring-emerald-500/20"
								: "border-slate-800 bg-slate-950/80"
						}`}
					>
						<!-- Scope Endpoint Header -->
						<div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800/80 pb-3">
							<div class="flex items-center gap-2.5">
								<span class={`text-xs font-bold font-mono px-2.5 py-1 rounded-lg border ${scopeMeta.class}`}>
									{ep.scope_id.toUpperCase()}
								</span>
								<div>
									<h4 class="font-bold text-sm text-white flex items-center gap-2">
										{ep.name}
										{#if ep.is_primary}
											<span class="text-[9px] font-mono uppercase px-1.5 py-0.2 rounded bg-emerald-950 text-emerald-400 border border-emerald-800 font-semibold">
												Primary
											</span>
										{/if}
									</h4>
									<div class="text-[11px] text-slate-400 font-mono mt-0.5">
										Target: {ep.target_type} • Config: {JSON.stringify(ep.config)}
									</div>
								</div>
							</div>

							<div class="flex items-center gap-3">
								{#if ep.last_latency_ms != null}
									<div class="text-right">
										<div class="text-sm font-bold text-white font-mono">{ep.last_latency_ms}ms</div>
										<div class="text-[10px] text-slate-400">Total Latency</div>
									</div>
								{/if}

								<span class={`text-[11px] font-bold px-3 py-1 rounded-full border ${getStatusBadgeClass(ep.status)}`}>
									{ep.status?.toUpperCase() || "PENDING"}
								</span>

								<button
									type="button"
									onclick={() => onDeleteEndpoint(service.id, ep.id)}
									class="text-slate-500 hover:text-rose-400 p-1 rounded hover:bg-slate-800 transition-colors"
									title="Delete Endpoint"
									aria-label="Delete Endpoint"
								>
									<Trash2 class="w-3.5 h-3.5" />
								</button>
							</div>
						</div>

						<!-- 4-Stage Sequential Hop Pipeline Graphic -->
						<div class="grid grid-cols-1 md:grid-cols-4 gap-3">
							<!-- Hop 1: Prober Node -->
							<div
								class="p-3.5 rounded-xl border bg-slate-900/90 border-slate-800 flex flex-col justify-between space-y-2 relative"
							>
								<div class="flex items-center justify-between">
									<span class="text-[10px] font-mono font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
										<Radio class="w-3 h-3 text-emerald-400" />
										Hop 1: Origin
									</span>
									<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
								</div>
								<div>
									<div class="font-bold text-xs text-white">Local Prober Node</div>
									<div class="text-[11px] text-slate-400 mt-0.5">Source prober engine</div>
								</div>
								<div class="text-[10px] font-mono text-emerald-400 bg-emerald-950/60 px-2 py-0.5 rounded border border-emerald-800/40 inline-block self-start">
									PASS • 0ms
								</div>
							</div>

							<!-- Hop 2: Network Fabric & DNS -->
							<div
								class={`p-3.5 rounded-xl border flex flex-col justify-between space-y-2 relative ${
									isDnsFailing
										? "bg-rose-950/30 border-rose-500/60 shadow-lg shadow-rose-500/10"
										: "bg-slate-900/90 border-slate-800"
								}`}
							>
								<div class="flex items-center justify-between">
									<span class="text-[10px] font-mono font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
										<Globe class="w-3 h-3 text-sky-400" />
										Hop 2: Fabric & DNS
									</span>
									{#if isDnsFailing}
										<span class="w-2 h-2 rounded-full bg-rose-500 animate-ping"></span>
									{:else}
										<span class="w-2 h-2 rounded-full bg-emerald-400"></span>
									{/if}
								</div>
								<div>
									<div class="font-bold text-xs text-white">
										{ep.scope_id.toUpperCase()} Transport
									</div>
									<div class="text-[11px] text-slate-400 mt-0.5 truncate font-mono">
										{#if trace?.resolved_ip}
											IP: {trace.resolved_ip}
										{:else if ep.scope_id === "lan"}
											Direct RFC1918 Route
										{:else if ep.scope_id === "tailnet"}
											WireGuard Mesh Peer
										{:else}
											DNS Resolution
										{/if}
									</div>
								</div>
								<div class={`text-[10px] font-mono px-2 py-0.5 rounded border inline-block self-start ${
									isDnsFailing
										? 'text-rose-400 bg-rose-950 border-rose-800'
										: 'text-sky-400 bg-sky-950/60 border-sky-800/40'
								}`}>
									{#if isDnsFailing}
										FAIL • DNS Unreachable
									{:else if trace?.dns_lookup_ms != null}
										PASS • {trace.dns_lookup_ms}ms DNS
									{:else}
										PASS • Direct
									{/if}
								</div>
							</div>

							<!-- Hop 3: Ingress Gateway & TLS Handshake -->
							<div
								class={`p-3.5 rounded-xl border flex flex-col justify-between space-y-2 relative ${
									isIngressFailing
										? "bg-rose-950/30 border-rose-500/60 shadow-lg shadow-rose-500/10"
										: "bg-slate-900/90 border-slate-800"
								}`}
							>
								<div class="flex items-center justify-between">
									<span class="text-[10px] font-mono font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
										<Lock class="w-3 h-3 text-indigo-400" />
										Hop 3: Ingress & TLS
									</span>
									{#if isIngressFailing}
										<span class="w-2 h-2 rounded-full bg-rose-500 animate-ping"></span>
									{:else}
										<span class="w-2 h-2 rounded-full bg-emerald-400"></span>
									{/if}
								</div>
								<div>
									<div class="font-bold text-xs text-white truncate">
										{#if trace?.connected_addr}
											{trace.connected_addr}
										{:else}
											Edge Gateway / Proxy
										{/if}
									</div>
									<div class="text-[11px] text-slate-400 mt-0.5 truncate font-mono">
										{#if trace?.tls_version}
											{trace.tls_version} ({trace.tls_cipher || "Encrypted"})
										{:else}
											TCP Handshake & Certs
										{/if}
									</div>
								</div>
								<div class={`text-[10px] font-mono px-2 py-0.5 rounded border inline-block self-start ${
									isIngressFailing
										? 'text-rose-400 bg-rose-950 border-rose-800'
										: 'text-indigo-400 bg-indigo-950/60 border-indigo-800/40'
								}`}>
									{#if isIngressFailing}
										FAIL • {failingHop?.toUpperCase()} Error
									{:else if trace?.tls_handshake_ms != null || trace?.tcp_connect_ms != null}
										PASS • {(trace.tcp_connect_ms || 0) + (trace.tls_handshake_ms || 0)}ms Handshake
									{:else}
										PASS • Gateway OK
									{/if}
								</div>
							</div>

							<!-- Hop 4: Target Application Response -->
							<div
								class={`p-3.5 rounded-xl border flex flex-col justify-between space-y-2 relative ${
									isAppFailing
										? "bg-rose-950/30 border-rose-500/60 shadow-lg shadow-rose-500/10"
										: "bg-slate-900/90 border-slate-800"
								}`}
							>
								<div class="flex items-center justify-between">
									<span class="text-[10px] font-mono font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
										<Server class="w-3 h-3 text-emerald-400" />
										Hop 4: Target App
									</span>
									{#if isAppFailing}
										<span class="w-2 h-2 rounded-full bg-rose-500 animate-ping"></span>
									{:else if ep.status === "up"}
										<span class="w-2 h-2 rounded-full bg-emerald-400"></span>
									{:else}
										<span class="w-2 h-2 rounded-full bg-slate-500"></span>
									{/if}
								</div>
								<div>
									<div class="font-bold text-xs text-white">
										{#if ep.last_status_code}
											HTTP {ep.last_status_code}
										{:else}
											Target Service
										{/if}
									</div>
									<div class="text-[11px] text-slate-400 mt-0.5 truncate font-mono">
										{#if trace?.ttfb_ms != null}
											TTFB: {trace.ttfb_ms}ms
										{:else if ep.last_message}
											{ep.last_message}
										{:else}
											Container Response
										{/if}
									</div>
								</div>
								<div class={`text-[10px] font-mono px-2 py-0.5 rounded border inline-block self-start ${
									isAppFailing
										? 'text-rose-400 bg-rose-950 border-rose-800'
										: 'text-emerald-400 bg-emerald-950/60 border-emerald-800/40'
								}`}>
									{#if isAppFailing}
										FAIL • {ep.last_message || "Non-200 Status"}
									{:else if ep.status === "up"}
										PASS • Verified
									{:else}
										Pending
									{/if}
								</div>
							</div>
						</div>

						<!-- Failure Diagnostic Hint Bar -->
						{#if isFailing && ep.last_message}
							<div class="p-3 rounded-xl bg-rose-950/30 border border-rose-900/50 flex items-center justify-between gap-3 text-xs">
								<div class="flex items-center gap-2 text-rose-300">
									<XCircle class="w-4 h-4 text-rose-400 shrink-0" />
									<span><strong class="font-semibold">Failing Point:</strong> {ep.last_message}</span>
								</div>
								{#if failingHop}
									<span class="text-[10px] font-mono px-2 py-0.5 rounded bg-rose-900/60 text-rose-200 uppercase font-bold">
										Hop: {failingHop}
									</span>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			{/if}
		</div>
	{/if}
</div>
