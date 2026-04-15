import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const ssr = false; // Bypass SSR for client fetch compatibility

export const load: PageLoad = async ({ params, fetch }) => {
    const response = await fetch(`/api/v1/status-pages/${params.slug}`);
    const payload = await response.json().catch(() => null);

    if (response.ok) {
        return {
            slug: params.slug,
            ...(payload ?? {}),
        };
    }

    if (response.status === 403 && payload?.password_required) {
        return {
            slug: params.slug,
            locked: true,
        };
    }

    if (response.status === 403) {
        throw error(403, payload?.error || 'Forbidden');
    }

    throw error(404, payload?.error || 'Status page not found');
};
