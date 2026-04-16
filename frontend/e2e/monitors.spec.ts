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
import { appBaseUrl, fixtureBaseUrl } from './helpers/env';

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
        await expect(
            page.getByRole('heading', { name: 'Monitors', level: 1 }),
        ).toBeVisible();
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
        const createResponsePromise = page.waitForResponse(
            (response) =>
                response.url().includes('/api/v1/monitors') &&
                response.request().method() === 'POST',
        );
        await createDialog.locator('form').evaluate((form) => {
            (form as HTMLFormElement).requestSubmit();
        });
        const createResponse = await createResponsePromise;
        expect(createResponse.ok()).toBeTruthy();
        const createdMonitor = (await createResponse.json()) as { id: string };
        const monitorUrlSuffix = `/api/v1/monitors/${createdMonitor.id}`;
        await expect(createDialog).toBeHidden();

        const createdRow = monitorRows(page).filter({ hasText: 'UI HTTP Monitor' });
        await expect(createdRow).toHaveCount(1);

        await createdRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Edit' }).click();

        const editDialog = page.getByRole('dialog', { name: 'Edit Monitor' });
        await editDialog.locator('#em-name').fill('UI HTTP Monitor Updated');
        await editDialog.locator('#em-host').fill(`${fixtureBaseUrl}/ok`);
        const updateResponsePromise = page.waitForResponse(
            (response) =>
                response.url().endsWith(monitorUrlSuffix) &&
                response.request().method() === 'PUT',
        );
        await editDialog.locator('form').evaluate((form) => {
            (form as HTMLFormElement).requestSubmit();
        });
        const updateResponse = await updateResponsePromise;
        expect(updateResponse.ok()).toBeTruthy();
        await expect(editDialog).toBeHidden();

        const updatedRow = monitorRows(page).filter({
            hasText: 'UI HTTP Monitor Updated',
        });
        await expect(updatedRow).toHaveCount(1);

        const pauseResponsePromise = page.waitForResponse(
            (response) =>
                response.url().endsWith(monitorUrlSuffix) &&
                response.request().method() === 'PUT',
        );
        await updatedRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Pause' }).click();
        const pauseResponse = await pauseResponsePromise;
        expect(pauseResponse.ok()).toBeTruthy();
        await expect(updatedRow).toContainText(/paused/i, { timeout: 10000 });

        const deleteResponsePromise = page.waitForResponse(
            (response) =>
                response.url().endsWith(monitorUrlSuffix) &&
                response.request().method() === 'DELETE',
        );
        page.once('dialog', async (dialog) => dialog.accept());
        await updatedRow.locator('[data-testid^="monitor-actions-"]').click();
        await page.getByRole('menuitem', { name: 'Delete' }).click();
        const deleteResponse = await deleteResponsePromise;
        expect(deleteResponse.ok()).toBeTruthy();
        await expect(updatedRow).toHaveCount(0);
    });

    test('edit failures surface a user-visible error', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();
        const monitor = await createMonitor(api, {
            name: 'Edit Failure Monitor',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });
        await api.dispose();

        await loginThroughUI(page);
        await page.goto('/monitors');

        const monitorDetailUrl = `${appBaseUrl}/api/v1/monitors/${monitor.id}`;
        await page.route(monitorDetailUrl, async (route) => {
            await route.fulfill({
                status: 500,
                contentType: 'application/json',
                body: JSON.stringify({ error: 'Failed to load monitor details' }),
            });
        });

        const monitorRow = monitorRows(page).filter({
            hasText: 'Edit Failure Monitor',
        });
        await expect(monitorRow).toHaveCount(1, { timeout: 10000 });

        await monitorRow.locator('[data-testid^="monitor-actions-"]').click();
        const failedResponsePromise = page.waitForResponse(
            (response) =>
                response.url() === monitorDetailUrl &&
                response.request().method() === 'GET',
        );
        page.once('dialog', async (dialog) => {
            expect(dialog.message()).toContain('Failed to load monitor details');
            await dialog.accept();
        });
        await page.getByRole('menuitem', { name: 'Edit' }).click();
        const failedResponse = await failedResponsePromise;
        expect(failedResponse.status()).toBe(500);
        await expect(page.getByRole('dialog', { name: 'Edit Monitor' })).toHaveCount(0);
    });
});