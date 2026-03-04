import { dev } from '$app/environment';

// In development, the SvelteKit dev server proxies /api calls to the Go backend.
// In production, the single binary serves both the static frontend and the API from the same host/port.
export const API_BASE = dev ? '' : '';

export async function fetchAPI(endpoint: string, options: RequestInit = {}) {
    const url = `${API_BASE}${endpoint}`;
    const headers = new Headers(options.headers || {});

    if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
        headers.set('Content-Type', 'application/json');
    }

    const res = await fetch(url, {
        ...options,
        headers,
        // Ensure cookies (session) are sent with every request
        credentials: 'same-origin'
    });

    if (!res.ok) {
        let message = 'An error occurred';
        try {
            const data = await res.json();
            message = data.error || message;
        } catch {
            // fallback
        }
        throw new Error(message);
    }

    return res.json();
}
