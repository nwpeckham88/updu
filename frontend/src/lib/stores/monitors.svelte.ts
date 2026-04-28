import { fetchAPI } from '$lib/api/client';

export type MonitorStatus = 'up' | 'down' | 'degraded' | 'pending' | 'paused';

export interface MonitorInvestigation {
    monitor_id: string;
    active: boolean;
    updated_by?: string;
    updated_at: string;
}

export interface Monitor {
    id: string;
    name: string;
    type: string;
    groups: string[];
    enabled: boolean;
    interval_s: number;
    status: MonitorStatus;
    last_latency_ms?: number;
    last_check?: string;
    investigation?: MonitorInvestigation;
    recent_checks?: { status: string; latency_ms?: number; checked_at: string }[];
    uptime_24h?: number;
}

class MonitorsStore {
    monitors = $state<Monitor[]>([]);
    loading = $state(false);
    #eventSource: EventSource | null = null;
    #reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    async init() {
        this.loading = true;
        try {
            const data = await fetchAPI('/api/v1/dashboard');
            this.monitors = data?.monitors ?? [];
            this.#connectSSE();
        } catch {
            this.monitors = [];
        } finally {
            this.loading = false;
        }
    }

    #connectSSE() {
        if (this.#eventSource) return;
        this.#eventSource = new EventSource('/api/v1/events');

        this.#eventSource.addEventListener('monitor:status', (e: MessageEvent) => {
            try {
                const data = JSON.parse(e.data);
                this.#patchMonitor(data.id, data);
            } catch (err) {
                console.error('SSE parse error', err);
            }
        });

        this.#eventSource.addEventListener('monitor:investigation', (e: MessageEvent) => {
            try {
                const data = JSON.parse(e.data) as MonitorInvestigation;
                this.monitors = this.monitors.map((monitor) => {
                    if (monitor.id !== data.monitor_id) return monitor;
                    if (data.active) return { ...monitor, investigation: data };

                    const { investigation: removedInvestigation, ...next } = monitor;
                    void removedInvestigation;
                    return next;
                });
            } catch (err) {
                console.error('SSE parse error', err);
            }
        });

        this.#eventSource.onerror = () => {
            this.#eventSource?.close();
            this.#eventSource = null;
            this.#reconnectTimer = setTimeout(() => this.#connectSSE(), 5000);
        };
    }

    destroy() {
        this.#eventSource?.close();
        this.#eventSource = null;
        if (this.#reconnectTimer) clearTimeout(this.#reconnectTimer);
    }

    #patchMonitor(id: string, patch: Partial<Monitor>) {
        this.monitors = this.monitors.map((monitor) =>
            monitor.id === id ? { ...monitor, ...patch } : monitor,
        );
    }
}

export const monitorsStore = new MonitorsStore();
