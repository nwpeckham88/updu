import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { fetchAPI } from '$lib/api/client';

export const ssr = false; // Bypass SSR for client fetch compatibility

export const load: PageLoad = async ({ params }) => {
    try {
        const res = await fetchAPI(`/api/v1/status-pages/${params.slug}`);
        if (!res) {
            throw error(404, 'Status page not found');
        }
        return res;
    } catch (e: any) {
        throw error(404, e.message || 'Status page not found');
    }
};
