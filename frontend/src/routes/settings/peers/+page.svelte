<script lang="ts">
    import { onMount } from 'svelte';
    import {
        Activity,
        Check,
        Copy,
        Cpu,
        HardDrive,
        Network,
        Radio,
        RefreshCw,
        Server,
        Trash2,
        UserCheck,
        UserX,
        Wifi,
    } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import Skeleton from '$lib/components/ui/skeleton.svelte';
    import Badge from '$lib/components/ui/badge.svelte';
    import {
        approvePeer,
        connectPeer,
        deletePeer,
        getPeers,
        type DiscoveredPeer,
        type LocalIdentity,
        type Peer,
        type SurvivorTriage,
    } from '$lib/api/peers';

    let loading = $state(true);
    let local = $state<LocalIdentity | null>(null);
    let peers = $state<Peer[]>([]);
    let discovered = $state<DiscoveredPeer[]>([]);
    let triage = $state<SurvivorTriage[]>([]);

    let connectAddress = $state('');
    let connectName = $state('');
    let connectLoading = $state(false);
    let connectError = $state('');
    let connectSuccess = $state('');

    let actionLoading = $state<string | null>(null);
    let copiedKey = $state(false);
    let copiedID = $state(false);

    async function loadData() {
        try {
            loading = true;
            const res = await getPeers();
            local = res.local ?? null;
            peers = res.peers ?? [];
            discovered = res.discovered ?? [];
            triage = res.triage ?? [];
        } catch {
            // failed to load
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        loadData();
        const timer = setInterval(loadData, 8000);
        return () => clearInterval(timer);
    });

    async function handleConnect(e: Event) {
        e.preventDefault();
        if (!connectAddress.trim()) return;

        connectLoading = true;
        connectError = '';
        connectSuccess = '';

        try {
            const paired = await connectPeer(connectAddress.trim(), connectName.trim() || undefined);
            connectSuccess = `Successfully initiated pairing with ${paired.name} (${paired.id})`;
            connectAddress = '';
            connectName = '';
            await loadData();
        } catch (err: any) {
            connectError = err.message || 'Failed to connect to peer';
        } finally {
            connectLoading = false;
        }
    }

    async function handleQuickConnect(peer: DiscoveredPeer) {
        actionLoading = peer.node_id;
        try {
            await connectPeer(peer.address, peer.name);
            await loadData();
        } catch (err: any) {
            alert(err.message || 'Failed to connect');
        } finally {
            actionLoading = null;
        }
    }

    async function handleApprove(id: string) {
        actionLoading = id;
        try {
            await approvePeer(id);
            await loadData();
        } catch (err: any) {
            alert(err.message || 'Failed to approve peer');
        } finally {
            actionLoading = null;
        }
    }

    async function handleDelete(id: string) {
        if (!confirm('Are you sure you want to disconnect and delete this peer?')) return;
        actionLoading = id;
        try {
            await deletePeer(id);
            await loadData();
        } catch (err: any) {
            alert(err.message || 'Failed to delete peer');
        } finally {
            actionLoading = null;
        }
    }

    function copyToClipboard(text: string, type: 'id' | 'key') {
        navigator.clipboard.writeText(text);
        if (type === 'id') {
            copiedID = true;
            setTimeout(() => (copiedID = false), 2000);
        } else {
            copiedKey = true;
            setTimeout(() => (copiedKey = false), 2000);
        }
    }

    const pendingPeers = $derived(peers.filter((p) => p.status === 'pending'));
    const approvedPeers = $derived(peers.filter((p) => p.status !== 'pending'));
</script>

<svelte:head>
    <title>Settings / Peers – updu</title>
</svelte:head>

<div class="space-y-6">
    <!-- Local Identity Card -->
    <div class="rounded-lg border border-border bg-card p-6">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
                <div class="flex items-center gap-2">
                    <Radio class="size-5 text-primary animate-pulse" />
                    <h2 class="text-lg font-semibold text-text">Local Node Identity</h2>
                    <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-500">
                        <span class="size-1.5 rounded-full bg-emerald-500"></span>
                        UDP 3001 Active
                    </span>
                </div>
                <p class="mt-1 text-sm text-text-muted">
                    This node advertises itself to nearby updu instances and authenticates peering proposals using Ed25519.
                </p>
            </div>
            <Button variant="outline" size="sm" onclick={loadData} disabled={loading}>
                <RefreshCw class="size-4 mr-1.5 {loading ? 'animate-spin' : ''}" />
                Refresh
            </Button>
        </div>

        {#if loading && !local}
            <div class="mt-4 grid gap-4 sm:grid-cols-3">
                <Skeleton class="h-16 w-full" />
                <Skeleton class="h-16 w-full" />
                <Skeleton class="h-16 w-full" />
            </div>
        {:else if local}
            <div class="mt-4 grid gap-4 sm:grid-cols-3">
                <div class="rounded-md border border-border/50 bg-background/50 p-3">
                    <p class="text-xs font-medium uppercase tracking-wider text-text-muted">Node ID</p>
                    <div class="mt-1 flex items-center justify-between">
                        <span class="font-mono text-base font-bold text-text">{local.node_id}</span>
                        <button
                            type="button"
                            class="text-text-muted hover:text-text transition-colors"
                            onclick={() => copyToClipboard(local?.node_id ?? '', 'id')}
                            title="Copy Node ID"
                        >
                            {#if copiedID}
                                <Check class="size-4 text-emerald-500" />
                            {:else}
                                <Copy class="size-4" />
                            {/if}
                        </button>
                    </div>
                </div>

                <div class="rounded-md border border-border/50 bg-background/50 p-3">
                    <p class="text-xs font-medium uppercase tracking-wider text-text-muted">Node Name</p>
                    <div class="mt-1 flex items-center justify-between">
                        <span class="text-base font-semibold text-text truncate">{local.name}</span>
                        <Server class="size-4 text-text-muted shrink-0" />
                    </div>
                </div>

                <div class="rounded-md border border-border/50 bg-background/50 p-3">
                    <p class="text-xs font-medium uppercase tracking-wider text-text-muted">Public Key</p>
                    <div class="mt-1 flex items-center justify-between">
                        <span class="font-mono text-xs text-text-muted truncate max-w-[180px]">
                            {local.public_key.slice(0, 16)}...{local.public_key.slice(-8)}
                        </span>
                        <button
                            type="button"
                            class="text-text-muted hover:text-text transition-colors"
                            onclick={() => copyToClipboard(local?.public_key ?? '', 'key')}
                            title="Copy Full Public Key"
                        >
                            {#if copiedKey}
                                <Check class="size-4 text-emerald-500" />
                            {:else}
                                <Copy class="size-4" />
                            {/if}
                        </button>
                    </div>
                </div>
            </div>
        {/if}
    </div>

    <!-- Pending Pairing Proposals (if any) -->
    {#if pendingPeers.length > 0}
        <div class="rounded-lg border border-amber-500/30 bg-amber-500/5 p-6">
            <div class="flex items-center gap-2 mb-4">
                <span class="size-2 rounded-full bg-amber-500 animate-ping"></span>
                <h3 class="text-base font-semibold text-text">Pending Pairing Proposals ({pendingPeers.length})</h3>
            </div>
            <p class="text-sm text-text-muted mb-4">
                These nodes have requested to federate with your updu instance. Approving them will establish a mutual monitoring feed.
            </p>

            <div class="grid gap-3">
                {#each pendingPeers as peer (peer.id)}
                    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-md border border-border bg-card p-4">
                        <div>
                            <div class="flex items-center gap-2">
                                <span class="font-semibold text-text">{peer.name}</span>
                                <Badge dot={false} icon={false} class="font-mono text-xs">{peer.id}</Badge>
                                <span class="text-xs text-text-muted">({peer.address})</span>
                            </div>
                            <p class="mt-1 font-mono text-xs text-text-muted truncate max-w-md">
                                Public Key: {peer.public_key}
                            </p>
                        </div>
                        <div class="flex items-center gap-2">
                            <Button
                                size="sm"
                                class="bg-emerald-600 hover:bg-emerald-700 text-white"
                                onclick={() => handleApprove(peer.id)}
                                disabled={actionLoading === peer.id}
                            >
                                <UserCheck class="size-4 mr-1.5" />
                                Approve
                            </Button>
                            <Button
                                size="sm"
                                variant="outline"
                                class="text-rose-500 hover:bg-rose-500/10"
                                onclick={() => handleDelete(peer.id)}
                                disabled={actionLoading === peer.id}
                            >
                                <UserX class="size-4 mr-1.5" />
                                Reject
                            </Button>
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    {/if}

    <!-- Connect Remote / Tailnet Peer Form -->
    <div class="rounded-lg border border-border bg-card p-6">
        <div class="flex items-center gap-2 mb-1">
            <Network class="size-5 text-primary" />
            <h3 class="text-base font-semibold text-text">Connect Remote / Tailnet Peer</h3>
        </div>
        <p class="text-sm text-text-muted mb-4">
            Connect to a peer node via Tailscale mesh IP (e.g. <code class="text-xs bg-muted px-1 py-0.5 rounded">100.x.y.z:3000</code>), private LAN, or public host. Tailnet and private LAN nodes communicate via direct HTTP (secured transparently by Tailscale WireGuard).
        </p>

        <form onsubmit={handleConnect} class="grid gap-4 sm:grid-cols-3 items-end">
            <div class="sm:col-span-1">
                <label for="peer-address" class="block text-xs font-medium text-text-muted mb-1">Peer Address *</label>
                <input
                    id="peer-address"
                    type="text"
                    bind:value={connectAddress}
                    placeholder="100.64.0.1:3000"
                    class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-primary"
                    required
                />
            </div>
            <div class="sm:col-span-1">
                <label for="peer-name" class="block text-xs font-medium text-text-muted mb-1">Display Name (Optional)</label>
                <input
                    id="peer-name"
                    type="text"
                    bind:value={connectName}
                    placeholder="Offsite-Agent"
                    class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-text placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-primary"
                />
            </div>
            <div class="sm:col-span-1">
                <Button type="submit" class="w-full" disabled={connectLoading}>
                    {#if connectLoading}
                        <RefreshCw class="size-4 mr-1.5 animate-spin" />
                        Connecting...
                    {:else}
                        <Network class="size-4 mr-1.5" />
                        Connect Peer
                    {/if}
                </Button>
            </div>
        </form>

        {#if connectError}
            <p class="mt-3 text-xs text-rose-500">{connectError}</p>
        {/if}
        {#if connectSuccess}
            <p class="mt-3 text-xs text-emerald-500">{connectSuccess}</p>
        {/if}
    </div>

    <!-- Discovered LAN / Tailnet Peers (UDP Beacon 3001) -->
    <div class="rounded-lg border border-border bg-card p-6">
        <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
                <Wifi class="size-5 text-primary" />
                <h3 class="text-base font-semibold text-text">Discovered LAN & Tailnet Peers</h3>
                <span class="rounded-full bg-muted px-2 py-0.5 text-xs text-text-muted">{discovered.length}</span>
            </div>
        </div>

        {#if discovered.length === 0}
            <div class="rounded-md border border-dashed border-border/70 p-6 text-center text-sm text-text-muted">
                Listening for UDP beacons on port 3001... Other updu instances on this LAN or Tailnet will automatically appear here.
            </div>
        {:else}
            <div class="grid gap-3">
                {#each discovered as d (d.node_id)}
                    <div class="flex items-center justify-between rounded-md border border-border bg-background/50 p-4">
                        <div>
                            <div class="flex items-center gap-2">
                                <span class="font-semibold text-text">{d.name}</span>
                                <Badge dot={false} icon={false} class="font-mono text-xs">{d.node_id}</Badge>
                            </div>
                            <p class="mt-0.5 text-xs text-text-muted">{d.address}</p>
                        </div>
                        <Button
                            size="sm"
                            variant="outline"
                            onclick={() => handleQuickConnect(d)}
                            disabled={actionLoading === d.node_id}
                        >
                            {#if actionLoading === d.node_id}
                                <RefreshCw class="size-3.5 mr-1 animate-spin" />
                            {/if}
                            Pair
                        </Button>
                    </div>
                {/each}
            </div>
        {/if}
    </div>

    <!-- Approved & Paired Peers -->
    <div class="rounded-lg border border-border bg-card p-6">
        <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
                <Server class="size-5 text-primary" />
                <h3 class="text-base font-semibold text-text">Paired & Federated Nodes</h3>
                <span class="rounded-full bg-muted px-2 py-0.5 text-xs text-text-muted">{approvedPeers.length}</span>
            </div>
        </div>

        {#if approvedPeers.length === 0}
            <div class="rounded-md border border-dashed border-border/70 p-8 text-center text-sm text-text-muted">
                No paired peers configured yet. Connect a Tailnet host or pair with a discovered node above to begin federating monitor status.
            </div>
        {:else}
            <div class="grid gap-4">
                {#each approvedPeers as peer (peer.id)}
                    <div class="rounded-md border border-border bg-background/50 p-4">
                        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                            <div>
                                <div class="flex items-center gap-2">
                                    <span class="font-bold text-text">{peer.name}</span>
                                    <Badge dot={false} icon={false} class="font-mono text-xs">{peer.id}</Badge>
                                    {#if peer.status === 'disconnected'}
                                        <span class="inline-flex items-center gap-1 rounded-full bg-rose-500/10 px-2 py-0.5 text-xs font-medium text-rose-500">
                                            Disconnected
                                        </span>
                                    {:else}
                                        <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-500">
                                            <span class="size-1.5 rounded-full bg-emerald-500"></span>
                                            Online
                                        </span>
                                    {/if}
                                </div>
                                <p class="mt-0.5 text-xs text-text-muted">{peer.address}</p>
                            </div>

                            <div class="flex items-center gap-2">
                                <Button
                                    size="sm"
                                    variant="outline"
                                    class="text-rose-500 hover:bg-rose-500/10"
                                    onclick={() => handleDelete(peer.id)}
                                    disabled={actionLoading === peer.id}
                                >
                                    <Trash2 class="size-3.5 mr-1" />
                                    Unpair
                                </Button>
                            </div>
                        </div>

                        <!-- Telemetry details if present -->
                        {#if peer.metadata}
                            <div class="mt-3 grid grid-cols-2 sm:grid-cols-4 gap-2 pt-3 border-t border-border/40 text-xs">
                                {#if peer.metadata.tailnet_ip}
                                    <div>
                                        <span class="text-text-muted">Tailnet:</span>
                                        <span class="font-mono ml-1 text-text">{peer.metadata.tailnet_ip}</span>
                                    </div>
                                {/if}
                                {#if peer.metadata.public_ip}
                                    <div>
                                        <span class="text-text-muted">WAN:</span>
                                        <span class="font-mono ml-1 text-text">{peer.metadata.public_ip}</span>
                                    </div>
                                {/if}
                                <div>
                                    <span class="text-text-muted">RAM:</span>
                                    <span class="font-mono ml-1 text-text">{peer.metadata.mem_pct?.toFixed(1)}%</span>
                                </div>
                                <div>
                                    <span class="text-text-muted">CPU:</span>
                                    <span class="font-mono ml-1 text-text">{peer.metadata.cpu_pct?.toFixed(1)}%</span>
                                </div>
                            </div>
                        {/if}

                        <!-- Triage badge/notes if disconnected -->
                        {#if peer.triage}
                            <div class="mt-3 rounded bg-rose-500/10 border border-rose-500/20 p-2.5 text-xs">
                                <div class="font-semibold text-rose-500">
                                    Survivor Triage: {peer.triage.probable_cause}
                                </div>
                                {#if peer.triage.notes}
                                    <p class="mt-1 text-rose-400/90">{peer.triage.notes}</p>
                                {/if}
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>
