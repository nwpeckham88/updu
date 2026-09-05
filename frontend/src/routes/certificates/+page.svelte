<script lang="ts">
	import { onMount } from "svelte";
	import {
		listCertificates,
		testCertificate,
		type TLSCertificate,
	} from "$lib/api/certificates";
	import {
		ShieldCheck,
		ShieldAlert,
		Lock,
		RefreshCw,
		Calendar,
		Globe,
		Search,
		CheckCircle2,
		AlertTriangle,
		ExternalLink,
		SlidersHorizontal,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Spinner from "$lib/components/ui/spinner.svelte";
	import { Dialog } from "bits-ui";

	let certs = $state<TLSCertificate[]>([]);
	let loading = $state(true);
	let error = $state("");
	let searchQuery = $state("");

	// Test Tool state
	let testDialogOpen = $state(false);
	let testHost = $state("");
	let testPort = $state(443);
	let testLoading = $state(false);
	let testResult = $state<TLSCertificate | null>(null);
	let testError = $state("");

	async function loadCerts() {
		try {
			loading = true;
			error = "";
			certs = await listCertificates();
		} catch (e: any) {
			error = e?.message || "Failed to load TLS certificates";
		} finally {
			loading = false;
		}
	}

	async function runTestHandshake() {
		if (!testHost.trim()) return;
		try {
			testLoading = true;
			testError = "";
			testResult = null;
			testResult = await testCertificate(testHost.trim(), testPort);
		} catch (e: any) {
			testError = e?.message || "Handshake failed";
		} finally {
			testLoading = false;
		}
	}

	onMount(() => {
		void loadCerts();
	});

	const filteredCerts = $derived(
		certs.filter((c) => {
			if (!searchQuery) return true;
			const q = searchQuery.toLowerCase();
			return (
				c.domain.toLowerCase().includes(q) ||
				c.issuer.toLowerCase().includes(q) ||
				(c.service_name && c.service_name.toLowerCase().includes(q))
			);
		})
	);

	function getExpiryBadge(days: number): { label: string; class: string; icon: any } {
		if (days <= 0) {
			return {
				label: "Expired",
				class: "bg-rose-500/10 text-rose-400 border-rose-500/20",
				icon: ShieldAlert,
			};
		}
		if (days <= 14) {
			return {
				label: `${days}d (Critical)`,
				class: "bg-rose-500/10 text-rose-400 border-rose-500/20",
				icon: AlertTriangle,
			};
		}
		if (days <= 30) {
			return {
				label: `${days}d (Expiring Soon)`,
				class: "bg-amber-500/10 text-amber-400 border-amber-500/20",
				icon: AlertTriangle,
			};
		}
		return {
			label: `${days} days`,
			class: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
			icon: CheckCircle2,
		};
	}

	function formatDate(iso: string): string {
		try {
			return new Date(iso).toLocaleDateString("en-US", {
				month: "short",
				day: "numeric",
				year: "numeric",
			});
		} catch {
			return iso;
		}
	}
</script>

<svelte:head>
	<title>TLS Certificates Dashboard – updu</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
				<Lock class="w-7 h-7 text-emerald-400" />
				TLS Certificates
			</h1>
			<p class="text-sm text-slate-400 mt-1">
				Real-time SSL/TLS certificate monitoring, expiration alerts, and handshake diagnostics.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<Button
				variant="secondary"
				class="gap-1.5 text-xs font-semibold text-emerald-400 border-emerald-500/20 bg-emerald-500/10 hover:bg-emerald-500/20"
				onclick={() => {
					testHost = "";
					testPort = 443;
					testResult = null;
					testError = "";
					testDialogOpen = true;
				}}
			>
				<ShieldCheck class="w-4 h-4" />
				Test TLS Handshake
			</Button>

			<Button variant="secondary" onclick={loadCerts} disabled={loading} class="gap-1.5 text-xs">
				<RefreshCw class={`w-3.5 h-3.5 ${loading ? "animate-spin text-emerald-400" : ""}`} />
				Refresh
			</Button>
		</div>
	</div>

	<!-- Summary Cards -->
	<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
		<div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
			<div class="text-xs uppercase font-bold text-slate-400">Audited Certificates</div>
			<div class="text-2xl font-bold text-white mt-1">{certs.length}</div>
			<div class="text-xs text-slate-500 mt-1">Discovered across all HTTPS endpoints</div>
		</div>

		<div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
			<div class="text-xs uppercase font-bold text-slate-400">Expiring in &le; 30 Days</div>
			<div class="text-2xl font-bold text-amber-400 mt-1">
				{certs.filter((c) => c.days_remaining <= 30 && c.days_remaining > 0).length}
			</div>
			<div class="text-xs text-slate-500 mt-1">Renewal required soon</div>
		</div>

		<div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
			<div class="text-xs uppercase font-bold text-slate-400">Expired or Critical</div>
			<div class="text-2xl font-bold text-rose-400 mt-1">
				{certs.filter((c) => c.days_remaining <= 0).length}
			</div>
			<div class="text-xs text-slate-500 mt-1">Outage / Security Alert</div>
		</div>
	</div>

	<!-- Filter Search Bar -->
	<div class="relative max-w-md">
		<Search class="w-4 h-4 text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
		<input
			type="text"
			placeholder="Filter by domain, issuer, or service..."
			bind:value={searchQuery}
			class="w-full pl-9 pr-4 py-2 rounded-xl bg-slate-900/80 border border-slate-800 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500"
		/>
	</div>

	<!-- Certificates Grid -->
	{#if loading && certs.length === 0}
		<div class="flex flex-col items-center justify-center min-h-[300px] border border-slate-800 rounded-xl bg-slate-900/40">
			<Spinner class="w-8 h-8 text-emerald-400" />
			<p class="text-sm text-slate-400 mt-3">Scanning certificate authority chains...</p>
		</div>
	{:else if error}
		<div class="p-6 rounded-xl border border-rose-900/50 bg-rose-950/20 text-rose-300">
			<p class="font-medium">Failed to load certificates</p>
			<p class="text-sm text-rose-400 mt-1">{error}</p>
		</div>
	{:else if filteredCerts.length === 0}
		<div class="p-12 text-center rounded-2xl border border-slate-800 bg-slate-900/40">
			<Lock class="w-10 h-10 text-slate-600 mx-auto mb-3" />
			<h3 class="text-base font-semibold text-white">No certificates discovered yet</h3>
			<p class="text-sm text-slate-400 max-w-sm mx-auto mt-1">
				Run a probe on an HTTP/HTTPS endpoint or click "Test TLS Handshake" above to audit a domain.
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each filteredCerts as cert}
				{@const badge = getExpiryBadge(cert.days_remaining)}
				<div class="p-4 rounded-2xl border border-slate-800 bg-slate-900/80 hover:border-slate-700 transition-all flex flex-col justify-between shadow-sm">
					<div>
						<!-- Top row: Domain & Badge -->
						<div class="flex items-start justify-between gap-2">
							<div class="flex items-center gap-2">
								<Globe class="w-4 h-4 text-emerald-400 shrink-0" />
								<span class="font-bold text-sm text-white truncate max-w-[180px]">{cert.domain}</span>
							</div>
							<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${badge.class} shrink-0`}>
								{badge.label}
							</span>
						</div>

						{#if cert.service_name}
							<div class="text-[11px] text-slate-400 mt-1 flex items-center gap-1">
								<span>Service:</span>
								<span class="font-medium text-slate-300">{cert.service_name}</span>
							</div>
						{/if}

						<!-- Details -->
						<div class="mt-3.5 space-y-2 text-xs border-t border-slate-800/80 pt-3">
							<div class="flex justify-between text-slate-400">
								<span>Issuer:</span>
								<span class="text-slate-200 font-medium truncate max-w-[160px]">{cert.issuer}</span>
							</div>
							<div class="flex justify-between text-slate-400">
								<span>Valid Until:</span>
								<span class="text-slate-200 font-medium">{formatDate(cert.valid_until)}</span>
							</div>

							<!-- SANs tags -->
							{#if cert.sans && cert.sans.length > 0}
								<div class="pt-1 flex flex-wrap gap-1">
									{#each cert.sans.slice(0, 3) as san}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800/80 text-slate-300">
											{san}
										</span>
									{/each}
									{#if cert.sans.length > 3}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800/80 text-slate-400">
											+{cert.sans.length - 3} more
										</span>
									{/if}
								</div>
							{/if}
						</div>
					</div>

					<!-- Days countdown bar -->
					<div class="mt-4 pt-3 border-t border-slate-800/60">
						<div class="w-full bg-slate-800 rounded-full h-1.5 overflow-hidden">
							<div
								class={`h-full rounded-full transition-all ${
									cert.days_remaining <= 14
										? "bg-rose-500"
										: cert.days_remaining <= 30
										? "bg-amber-500"
										: "bg-emerald-500"
								}`}
								style={`width: ${Math.min(100, Math.max(5, (cert.days_remaining / 90) * 100))}%`}
							></div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Test Handshake Dialog -->
	<Dialog.Root bind:open={testDialogOpen}>
		<Dialog.Portal>
			<Dialog.Overlay class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 animate-fade-in" />
			<Dialog.Content class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 z-50 shadow-2xl space-y-4">
				<div class="flex items-center justify-between border-b border-slate-800 pb-3">
					<div class="flex items-center gap-2">
						<ShieldCheck class="w-5 h-5 text-emerald-400" />
						<h3 class="font-bold text-base text-white">Audit Live TLS Handshake</h3>
					</div>
					<Dialog.Close class="text-slate-400 hover:text-white">✕</Dialog.Close>
				</div>

				<div class="space-y-3">
					<div>
						<label for="test-host-input" class="text-xs font-semibold text-slate-300">Hostname / FQDN</label>
						<input
							id="test-host-input"
							type="text"
							placeholder="e.g. vault.kn8design.com or google.com"
							bind:value={testHost}
							class="w-full mt-1 px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-emerald-500"
						/>
					</div>
					<div>
						<label for="test-port-input" class="text-xs font-semibold text-slate-300">Port (Default 443)</label>
						<input
							id="test-port-input"
							type="number"
							bind:value={testPort}
							class="w-full mt-1 px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-sm text-white focus:outline-none focus:border-emerald-500"
						/>
					</div>
					<Button
						variant="primary"
						class="w-full mt-2"
						disabled={testLoading || !testHost.trim()}
						onclick={runTestHandshake}
					>
						{#if testLoading}
							<Spinner class="w-4 h-4 text-slate-950 mr-2" />
							Auditing Handshake...
						{:else}
							Perform TLS Audit
						{/if}
					</Button>
				</div>

				{#if testError}
					<div class="p-3 rounded-xl bg-rose-950/40 border border-rose-800 text-rose-300 text-xs">
						{testError}
					</div>
				{/if}

				{#if testResult}
					{@const testBadge = getExpiryBadge(testResult.days_remaining)}
					<div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2.5 text-xs animate-fade-in">
						<div class="flex items-center justify-between">
							<span class="font-bold text-sm text-white">{testResult.domain}</span>
							<span class={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${testBadge.class}`}>
								{testBadge.label}
							</span>
						</div>
						<div class="flex justify-between text-slate-400">
							<span>Issuer:</span>
							<span class="text-slate-200 font-semibold">{testResult.issuer}</span>
						</div>
						<div class="flex justify-between text-slate-400">
							<span>Valid Until:</span>
							<span class="text-slate-200 font-semibold">{formatDate(testResult.valid_until)}</span>
						</div>
						<div class="flex justify-between text-slate-400">
							<span>Signature Algorithm:</span>
							<span class="text-slate-200 font-mono">{testResult.signature_algorithm}</span>
						</div>
						{#if testResult.sans && testResult.sans.length > 0}
							<div>
								<span class="text-slate-400 block mb-1">SANs:</span>
								<div class="flex flex-wrap gap-1">
									{#each testResult.sans as san}
										<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">{san}</span>
									{/each}
								</div>
							</div>
						{/if}
					</div>
				{/if}
			</Dialog.Content>
		</Dialog.Portal>
	</Dialog.Root>
</div>
