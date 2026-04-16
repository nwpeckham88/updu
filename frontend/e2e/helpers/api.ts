import type { APIRequestContext, APIResponse } from '@playwright/test';

export interface MonitorRecord {
    id: string;
    name: string;
    type: string;
    groups?: string[];
    interval_s: number;
    enabled: boolean;
    status?: string;
    last_latency_ms?: number;
    config: Record<string, unknown>;
}

interface DashboardResponse {
    monitors?: MonitorRecord[];
}

interface StatusPageRecord {
    id: string;
    name: string;
    slug: string;
}

interface IncidentRecord {
    id: string;
    title: string;
}

interface StatusPageGroup {
    name: string;
    monitor_ids: string[];
}

interface WaitRule {
    status?: string;
    requireLatency?: boolean;
}

async function assertOk(response: APIResponse): Promise<void> {
    if (response.ok()) {
        return;
    }

    const body = await response.text();
    throw new Error(
        `Request failed with ${response.status()} ${response.statusText()}: ${body}`,
    );
}

async function readJson<T>(response: APIResponse): Promise<T> {
    await assertOk(response);
    return (await response.json()) as T;
}

function sleep(milliseconds: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

export async function listMonitors(
    api: APIRequestContext,
): Promise<MonitorRecord[]> {
    return readJson<MonitorRecord[]>(await api.get('/api/v1/monitors'));
}

export async function createMonitor(
    api: APIRequestContext,
    payload: Partial<MonitorRecord> & {
        name: string;
        type: string;
        config: Record<string, unknown>;
    },
): Promise<MonitorRecord> {
    return readJson<MonitorRecord>(
        await api.post('/api/v1/monitors', {
            data: {
                groups: payload.groups ?? ['Core'],
                interval_s: payload.interval_s ?? 10,
                ...payload,
            },
        }),
    );
}

export async function getMonitor(
    api: APIRequestContext,
    id: string,
): Promise<MonitorRecord> {
    return readJson<MonitorRecord>(await api.get(`/api/v1/monitors/${id}`));
}

export async function setMonitorEnabled(
    api: APIRequestContext,
    id: string,
    enabled: boolean,
): Promise<void> {
    const monitor = await getMonitor(api, id);
    await assertOk(
        await api.put(`/api/v1/monitors/${id}`, {
            data: {
                ...monitor,
                enabled,
            },
        }),
    );
}

export async function clearMonitors(api: APIRequestContext): Promise<void> {
    const monitors = await listMonitors(api);

    for (const monitor of monitors) {
        await assertOk(await api.delete(`/api/v1/monitors/${monitor.id}`));
    }
}

export async function clearIncidents(api: APIRequestContext): Promise<void> {
    const incidents = await readJson<IncidentRecord[]>(
        await api.get('/api/v1/incidents'),
    );

    for (const incident of incidents) {
        await assertOk(await api.delete(`/api/v1/incidents/${incident.id}`));
    }
}

export async function clearStatusPages(api: APIRequestContext): Promise<void> {
    const pages = await readJson<StatusPageRecord[]>(
        await api.get('/api/v1/status-pages'),
    );

    for (const page of pages) {
        await assertOk(await api.delete(`/api/v1/status-pages/${page.id}`));
    }
}

export async function createStatusPage(
    api: APIRequestContext,
    payload: {
        name: string;
        slug: string;
        description?: string;
        is_public: boolean;
        groups: StatusPageGroup[];
        password?: string;
        clear_password?: boolean;
    },
): Promise<StatusPageRecord> {
    return readJson<StatusPageRecord>(
        await api.post('/api/v1/status-pages', { data: payload }),
    );
}

export async function waitForDashboardMonitors(
    api: APIRequestContext,
    rules: Record<string, WaitRule>,
    timeoutMs = 30_000,
): Promise<MonitorRecord[]> {
    const deadline = Date.now() + timeoutMs;

    while (Date.now() < deadline) {
        const dashboard = await readJson<DashboardResponse>(
            await api.get('/api/v1/dashboard'),
        );
        const monitors = dashboard.monitors ?? [];

        const ready = Object.entries(rules).every(([name, rule]) => {
            const monitor = monitors.find((candidate) => candidate.name === name);

            if (!monitor) {
                return false;
            }

            if (rule.status && monitor.status !== rule.status) {
                return false;
            }

            if (
                rule.requireLatency &&
                typeof monitor.last_latency_ms !== 'number'
            ) {
                return false;
            }

            return true;
        });

        if (ready) {
            return monitors;
        }

        await sleep(1_000);
    }

    throw new Error('Timed out waiting for monitor checks to stabilize');
}