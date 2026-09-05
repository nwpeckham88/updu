import { fetchAPI } from '$lib/api/client';

export interface LocalIdentity {
    node_id: string;
    name: string;
    public_key: string;
}

export interface PeerTelemetry {
    tailnet_ip?: string;
    public_ip?: string;
    mem_pct: number;
    cpu_pct: number;
    uptime_s: number;
}

export interface TracerouteHop {
    hop: number;
    address: string;
    rtt_ms: number;
    loss_pct: number;
}

export interface SurvivorTriage {
    peer_id: string;
    peer_name: string;
    discovered_at: string;
    probable_cause: string;
    tailnet_reachable: boolean;
    public_wan_reachable: boolean;
    tailnet_ip?: string;
    public_ip?: string;
    last_telemetry?: PeerTelemetry;
    traceroute_hops?: TracerouteHop[];
    notes?: string;
}

export interface Peer {
    id: string;
    name: string;
    address: string;
    public_key: string;
    role: string;
    status: 'pending' | 'approved' | 'rejected' | 'disconnected';
    last_seen?: string;
    metadata?: PeerTelemetry;
    created_at: string;
    latency_ms?: number;
    triage?: SurvivorTriage;
}

export interface DiscoveredPeer {
    node_id: string;
    name: string;
    address: string;
    last_seen: string;
}

export interface PeersResponse {
    local?: LocalIdentity;
    peers: Peer[];
    discovered: DiscoveredPeer[];
    triage: SurvivorTriage[];
}

export async function getPeers(): Promise<PeersResponse> {
    return fetchAPI<PeersResponse>('/api/v1/admin/peers');
}

export async function approvePeer(id: string): Promise<void> {
    await fetchAPI('/api/v1/admin/peers/approve', {
        method: 'POST',
        body: JSON.stringify({ id }),
    });
}

export async function connectPeer(address: string, name?: string): Promise<Peer> {
    return fetchAPI<Peer>('/api/v1/admin/peers/connect', {
        method: 'POST',
        body: JSON.stringify({ address, name }),
    });
}

export async function deletePeer(id: string): Promise<void> {
    await fetchAPI(`/api/v1/admin/peers/${encodeURIComponent(id)}`, {
        method: 'DELETE',
    });
}
