import { expect, test } from '@playwright/test';
import { createAuthenticatedRequestContext } from './helpers/auth';
import {
    clearIncidents,
    clearMonitors,
    clearStatusPages,
    createMonitor,
    createStatusPage,
    waitForDashboardMonitors,
} from './helpers/api';
import { fixtureBaseUrl } from './helpers/env';

test.describe('smoke', () => {
    test.beforeEach(async () => {
        const api = await createAuthenticatedRequestContext();
        await clearStatusPages(api);
        await clearIncidents(api);
        await clearMonitors(api);
        await api.dispose();
    });

    test.afterEach(async () => {
        const api = await createAuthenticatedRequestContext();
        await clearStatusPages(api);
        await clearIncidents(api);
        await clearMonitors(api);
        await api.dispose();
    });

    test('settings and incidents pages reach a ready state', async ({ page }) => {
        await page.goto('/settings');
        await expect(page).toHaveURL(/\/settings\/general$/);
        await expect(
            page.getByRole('heading', { name: 'Settings' }),
        ).toBeVisible({ timeout: 10000 });
        await expect(
            page.getByRole('heading', { name: 'Instance Profile' }),
        ).toBeVisible();

        await page.goto('/incidents');
        await expect(page).toHaveURL(/\/incidents$/);
        await expect(
            page.getByRole('heading', { name: 'Incidents', level: 1 }),
        ).toBeVisible();
        await expect(page.getByPlaceholder('Search incidents...')).toBeVisible();
    });

    test('public status page renders for a seeded page', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();
        const monitor = await createMonitor(api, {
            name: 'Status Fixture Monitor',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });

        await waitForDashboardMonitors(api, {
            'Status Fixture Monitor': { status: 'up', requireLatency: true },
        });

        await createStatusPage(api, {
            name: 'Playwright Public Status',
            slug: 'playwright-public-status',
            description: 'Smoke test status page',
            is_public: true,
            groups: [{ name: '', monitor_ids: [monitor.id] }],
            clear_password: false,
        });
        await api.dispose();

        await page.goto('/status/playwright-public-status');
        await expect(
            page.getByRole('heading', { name: 'Playwright Public Status' }),
        ).toBeVisible();
        await expect(page.getByText('Status Fixture Monitor')).toBeVisible();
    });
});