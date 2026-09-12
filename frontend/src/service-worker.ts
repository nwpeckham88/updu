/// <reference types="@sveltejs/kit" />
import { build, files, version } from '$service-worker';

const CACHE = `updu-cache-${version}`;
const ASSETS = [...build, ...files];

self.addEventListener('install', (event: any) => {
	async function addFilesToCache() {
		const cache = await caches.open(CACHE);
		await cache.addAll(ASSETS);
	}
	event.waitUntil(addFilesToCache());
});

self.addEventListener('activate', (event: any) => {
	async function deleteOldCaches() {
		for (const key of await caches.keys()) {
			if (key !== CACHE) await caches.delete(key);
		}
	}
	event.waitUntil(deleteOldCaches());
});

self.addEventListener('fetch', (event: any) => {
	if (event.request.method !== 'GET') return;

	const url = new URL(event.request.url);

	// Never cache API routes, realtime SSE streams, health checks, or metadata
	if (
		url.pathname.startsWith('/api/') ||
		url.pathname.startsWith('/heartbeat/') ||
		url.pathname.startsWith('/.well-known/') ||
		url.pathname === '/healthz' ||
		url.pathname === '/llms.txt'
	) {
		return;
	}

	async function respond() {
		const cache = await caches.open(CACHE);

		// For static assets and Vite chunks, serve from cache first
		if (ASSETS.includes(url.pathname)) {
			const cachedResponse = await cache.match(url.pathname);
			if (cachedResponse) return cachedResponse;
		}

		// For navigation requests (HTML page), try network first, then cache
		try {
			const response = await fetch(event.request);
			if (response.status === 200) {
				cache.put(event.request, response.clone());
			}
			return response;
		} catch {
			const cached = await cache.match(event.request);
			if (cached) return cached;
			// Fallback to cached root shell
			const root = await cache.match('/');
			if (root) return root;
			return new Response('Offline - updu is temporarily unreachable', {
				status: 503,
				headers: { 'Content-Type': 'text/plain' },
			});
		}
	}

	event.respondWith(respond());
});
