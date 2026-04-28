import { expect, test, type Page } from '@playwright/test';
import { createAuthenticatedRequestContext } from './helpers/auth';
import {
    clearIncidents,
    clearMonitors,
    clearStatusPages,
} from './helpers/api';

interface DashboardMonitorFixture {
    id: string;
    name: string;
    type: string;
    groups: string[];
    enabled: boolean;
    interval_s: number;
    status: 'up' | 'down' | 'degraded' | 'pending';
    last_latency_ms?: number;
    uptime_24h?: number;
    recent_checks: { status: string; latency_ms?: number; checked_at: string }[];
}

async function resetFixtures(): Promise<void> {
    const api = await createAuthenticatedRequestContext();
    await clearStatusPages(api);
    await clearIncidents(api);
    await clearMonitors(api);
    await api.dispose();
}

function monitorFixture(
    overrides: Partial<DashboardMonitorFixture> & Pick<DashboardMonitorFixture, 'id' | 'name' | 'status'>,
): DashboardMonitorFixture {
    const now = Date.now();
    return {
        type: 'http',
        groups: ['Core'],
        enabled: true,
        interval_s: 30,
        last_latency_ms: overrides.status === 'down' ? undefined : 124,
        uptime_24h: overrides.status === 'down' ? 92.5 : 100,
        recent_checks: Array.from({ length: 6 }, (_, index) => ({
            status: overrides.status,
            latency_ms: overrides.status === 'down' ? undefined : 124,
            checked_at: new Date(now - index * 30_000).toISOString(),
        })),
        ...overrides,
    };
}

async function mockDashboard(
    page: Page,
    monitors: DashboardMonitorFixture[],
): Promise<void> {
    await page.route('**/api/v1/dashboard', async (route) => {
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ monitors, sse_clients: 1 }),
        });
    });
}

async function documentOverflow(page: Page): Promise<number> {
    return page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
}

test.describe('dashboard situational awareness', () => {
    test.describe.configure({ timeout: 90_000 });

    test.beforeEach(async () => {
        await resetFixtures();
    });

    test.afterEach(async () => {
        await resetFixtures();
    });

    test('summarizes an operational fleet with a verdict-first health region', async ({ page }) => {
        await mockDashboard(page, [
            monitorFixture({
                id: 'dashboard-healthy-monitor',
                name: 'Dashboard Healthy Monitor',
                status: 'up',
            }),
        ]);

        await page.setViewportSize({ width: 375, height: 667 });
        await page.goto('/');

        const healthRegion = page.getByRole('region', {
            name: /system health: all systems operational/i,
        });
        await expect(healthRegion).toBeVisible();
        await expect(healthRegion).toContainText('All systems operational');
        await expect(healthRegion).toContainText('Down');
        await expect(healthRegion).toContainText('0/1');
        // Chromium can report a few sub-pixels of overflow from scrollbar/font rounding.
        await expect(await documentOverflow(page)).toBeLessThanOrEqual(5);
    });

    test('surfaces outages with non-color status text on the dashboard', async ({ page }) => {
        await mockDashboard(page, [
            monitorFixture({
                id: 'dashboard-down-monitor',
                name: 'Dashboard Down Monitor',
                status: 'down',
            }),
        ]);

        await page.goto('/');

        const healthRegion = page.getByRole('region', {
            name: /system health: service outage/i,
        });
        await expect(healthRegion).toBeVisible();
        await expect(healthRegion).toContainText('Service outage');
        await expect(healthRegion).toContainText('Down');
        await expect(healthRegion).toContainText('1/1');
        const downMonitorCard = page.getByRole('link', {
            name: /Dashboard Down Monitor Status: Down/i,
        });
        await expect(downMonitorCard).toContainText('Down');
        await expect(downMonitorCard.locator('svg').first()).toBeVisible();
    });

    test('does not treat paused monitors with stale down status as outages', async ({ page }) => {
        await mockDashboard(page, [
            monitorFixture({
                id: 'dashboard-paused-monitor',
                name: 'Dashboard Paused Monitor',
                status: 'down',
                enabled: false,
            }),
        ]);

        await page.goto('/');

        await expect(
            page.getByRole('region', {
                name: /system health: no active monitors/i,
            }),
        ).toBeVisible();
        await expect(page.getByRole('region', { name: /system health/i })).not.toContainText(
            'Service outage',
        );
        await expect(
            page.getByRole('link', {
                name: /Dashboard Paused Monitor Status: Paused/i,
            }),
        ).toContainText('Paused');

        await expect(
            page.getByRole('img', {
                name: /Dashboard Paused Monitor heartbeat history.*0 failed checks.*6 paused periods/i,
            }),
        ).toBeVisible();
    });
});