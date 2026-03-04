import { writable, get } from 'svelte/store';
import { fetchAPI } from '$lib/api/client';

export type MonitorStatus = 'up' | 'down' | 'degraded' | 'pending' | 'paused';

export interface Monitor {
    id: string;
    name: string;
    type: string;
    group_name: string;
    enabled: boolean;
    interval_s: number;
    status: MonitorStatus;
    last_latency_ms?: number;
    last_check?: string;
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
                const idx = this.monitors.findIndex(m => m.id === data.id);
                if (idx >= 0) {
                    this.monitors[idx] = { ...this.monitors[idx], ...data };
                }
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
}

export const monitorsStore = new MonitorsStore();
