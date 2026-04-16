import { expect, test, type Locator, type Page } from '@playwright/test';
import { loginThroughUI, createAuthenticatedRequestContext } from './helpers/auth';
import {
    clearIncidents,
    clearMonitors,
    clearStatusPages,
    createMonitor,
    setMonitorEnabled,
    waitForDashboardMonitors,
} from './helpers/api';
import { fixtureBaseUrl } from './helpers/env';

function monitorRows(page: Page): Locator {
    return page.locator('[data-testid^="monitor-row-"]');
}

async function monitorNames(page: Page): Promise<string[]> {
    return page
        .locator('[data-testid^="monitor-row-"] td:nth-child(2) a')
        .allTextContents();
}

test.describe('monitors', () => {
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

    test('monitor list search, sorting, and empty state are stable', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();

        const alphaFast = await createMonitor(api, {
            name: 'Alpha Fast',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });
        const bravoSlow = await createMonitor(api, {
            name: 'Bravo Slow',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/slow`,
                method: 'GET',
                expected_status: 200,
            },
        });
        await createMonitor(api, {
            name: 'Charlie Down',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/fail`,
                method: 'GET',
                expected_status: 200,
            },
        });

        await setMonitorEnabled(api, alphaFast.id, true);
        await setMonitorEnabled(api, bravoSlow.id, true);
        await waitForDashboardMonitors(api, {
            'Alpha Fast': { status: 'up', requireLatency: true },
            'Bravo Slow': { status: 'up', requireLatency: true },
            'Charlie Down': { status: 'down' },
        });
        await api.dispose();

        await loginThroughUI(page);
        await page.goto('/monitors');
        await expect(page.getByRole('heading', { name: 'Monitors' })).toBeVisible();
        await expect(monitorRows(page)).toHaveCount(3);

        await page.getByTestId('search-monitors').fill('bravo');
        await expect(monitorRows(page)).toHaveCount(1);
        await expect(monitorRows(page).first()).toContainText('Bravo Slow');

        await page.getByTestId('search-monitors').fill('missing-monitor');
        await expect(page.getByTestId('monitors-empty-state')).toContainText(
            'No monitors matching "missing-monitor"',
        );

        await page.getByTestId('search-monitors').fill('');
        await expect(monitorRows(page)).toHaveCount(3);

        await page.getByTestId('sort-name').click();
        await expect(await monitorNames(page)).toEqual([
            'Alpha Fast',
            'Bravo Slow',
            'Charlie Down',
        ]);

        await page.getByTestId('sort-name').click();
        await expect(await monitorNames(page)).toEqual([
            'Charlie Down',
            'Bravo Slow',
            'Alpha Fast',
        ]);

        await page.getByTestId('sort-status').click();
        await expect(monitorRows(page).first()).toContainText('Charlie Down');

        await page.getByTestId('sort-latency').click();
        const latencyAscending = await monitorNames(page);
        await expect(
            latencyAscending.indexOf('Alpha Fast'),
        ).toBeLessThan(latencyAscending.indexOf('Bravo Slow'));

        await page.getByTestId('sort-latency').click();
        const latencyDescending = await monitorNames(page);
        await expect(
            latencyDescending.indexOf('Alpha Fast'),
        ).toBeGreaterThan(latencyDescending.indexOf('Bravo Slow'));
    });

    test('monitor CRUD works through the UI', async ({ page }) => {
        await loginThroughUI(page);
        await page.goto('/monitors');

        await page.getByRole('button', { name: 'New Monitor' }).click();
        const createDialog = page.getByRole('dialog', {
            name: 'Add New Monitor',
        });

        await createDialog.locator('#cm-name').fill('UI HTTP Monitor');
        await createDialog.locator('#cm-host').fill(`${fixtureBaseUrl}/ok`);
        await createDialog.getByRole('button', { name: 'Create Monitor' }).click();

        const createdRow = monitorRows(page).filter({ hasText: 'UI HTTP Monitor' });
        await expect(createdRow).toHaveCount(1);

        await createdRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Edit' }).click();

        const editDialog = page.getByRole('dialog', { name: 'Edit Monitor' });
        await editDialog.locator('#em-name').fill('UI HTTP Monitor Updated');
        await editDialog.getByRole('button', { name: 'Update Monitor' }).click();

        const updatedRow = monitorRows(page).filter({
            hasText: 'UI HTTP Monitor Updated',
        });
        await expect(updatedRow).toHaveCount(1);

        await updatedRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Pause' }).click();
        await expect(updatedRow).toContainText(/paused/i);

        page.once('dialog', async (dialog) => dialog.accept());
        await updatedRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Delete' }).click();
        await expect(updatedRow).toHaveCount(0);
    });
});