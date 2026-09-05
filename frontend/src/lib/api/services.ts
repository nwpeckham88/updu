import { fetchAPI } from './client';

export interface ServiceEndpoint {
    id: string;
    service_id: string;
    name: string;
    scope_id: string; // 'lan' | 'tailnet' | 'public'
    target_type: string;
    config: Record<string, any>;
    interval_s: number;
    timeout_s: number;
    retries: number;
    is_primary: boolean;
    status?: 'up' | 'down' | 'degraded' | 'pending';
    last_latency_ms?: number;
    last_status_code?: number;
    last_message?: string;
    last_checked_at?: string;
}

export interface Service {
    id: string;
    name: string;
    type: 'web' | 'infra' | 'database' | 'host' | 'job' | string;
    zone_id: string;
    groups?: string[];
    tags?: string[];
    enabled: boolean;
    endpoints?: ServiceEndpoint[];
    status?: 'up' | 'down' | 'degraded' | 'pending';
    diagnosis?: string;
    primary_latency_ms?: number;
    last_check?: string;
    created_at?: string;
    updated_at?: string;
}

export interface ServiceDiagnosis {
    status: 'up' | 'down' | 'degraded' | 'pending';
    summary: string;
    probable_cause: string;
    action_hint: string;
    healthy_count: number;
    total_count: number;
    probe_breakdown: string[];
}

export interface Zone {
    id: string;
    name: string;
    description?: string;
}

export interface Scope {
    id: string;
    name: string;
    description?: string;
}

export interface TopologyNode {
    id: string;
    name: string;
    zone_id: string;
    scopes: string[];
    status: string;
    is_local: boolean;
    last_seen: string;
}

export interface TopologyEdge {
    id: string;
    node_id: string;
    service_id: string;
    endpoint_id: string;
    scope_id: string;
    status: 'up' | 'down' | 'degraded' | 'pending';
    latency_ms?: number;
    message?: string;
    checked_at: string;
}

export interface NetworkTopology {
    nodes: TopologyNode[];
    zones: Zone[];
    scopes: Scope[];
    services: Service[];
    edges: TopologyEdge[];
}

export async function listServices(): Promise<Service[]> {
    return fetchAPI<Service[]>('/api/v1/services');
}

export async function getService(id: string): Promise<{ service: Service; diagnosis: ServiceDiagnosis }> {
    return fetchAPI<{ service: Service; diagnosis: ServiceDiagnosis }>(`/api/v1/services/${id}`);
}

export async function createService(payload: Partial<Service>): Promise<Service> {
    return fetchAPI<Service>('/api/v1/services', {
        method: 'POST',
        body: JSON.stringify(payload)
    });
}

export async function updateService(id: string, payload: Partial<Service>): Promise<Service> {
    return fetchAPI<Service>(`/api/v1/services/${id}`, {
        method: 'PUT',
        body: JSON.stringify(payload)
    });
}

export async function deleteService(id: string): Promise<{ status: string }> {
    return fetchAPI<{ status: string }>(`/api/v1/services/${id}`, {
        method: 'DELETE'
    });
}

export async function probeService(id: string): Promise<{ service_id: string; diagnosis: ServiceDiagnosis; probes: Record<string, any> }> {
    return fetchAPI<{ service_id: string; diagnosis: ServiceDiagnosis; probes: Record<string, any> }>(`/api/v1/services/${id}/probe`, {
        method: 'POST'
    });
}

export async function getNetworkTopology(): Promise<NetworkTopology> {
    return fetchAPI<NetworkTopology>('/api/v1/topology');
}

export async function listZones(): Promise<Zone[]> {
    return fetchAPI<Zone[]>('/api/v1/zones');
}

export async function createZone(payload: Partial<Zone>): Promise<Zone> {
    return fetchAPI<Zone>('/api/v1/zones', {
        method: 'POST',
        body: JSON.stringify(payload)
    });
}

export async function listScopes(): Promise<Scope[]> {
    return fetchAPI<Scope[]>('/api/v1/scopes');
}
