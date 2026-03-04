import { fetchAPI } from '$lib/api/client';
import { goto } from '$app/navigation';

export type User = {
    id: string;
    username: string;
    role: 'admin' | 'viewer';
};

class AuthStore {
    user = $state<User | null>(null);
    loading = $state(true);
    initialized = $state(false);

    async init() {
        this.loading = true;
        try {
            const u = await fetchAPI('/api/v1/auth/session');
            this.user = u;
        } catch {
            this.user = null;
        } finally {
            this.loading = false;
            this.initialized = true;
        }
    }

    async logout() {
        try {
            await fetchAPI('/api/v1/auth/logout', { method: 'POST' });
        } finally {
            this.user = null;
            goto('/login');
        }
    }
}

export const authStore = new AuthStore();
