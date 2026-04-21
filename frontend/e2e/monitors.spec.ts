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

    test('monitor details explain what each monitor actually checks', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();
        const httpMonitor = await createMonitor(api, {
            name: 'HTTP Detail Monitor',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });
        const transactionMonitor = await createMonitor(api, {
            name: 'Transaction Detail Monitor',
            type: 'transaction',
            config: {
                steps: [
                    {
                        url: `${fixtureBaseUrl}/ok`,
                        method: 'GET',
                        expected_status: 200,
                    },
                    {
                        url: `${fixtureBaseUrl}/slow`,
                        method: 'GET',
                        expected_status: 200,
                    },
                ],
            },
        });
        await api.dispose();

        await loginThroughUI(page);

        await page.goto(`/monitors/${httpMonitor.id}`);
        await expect(
            page.getByRole('heading', { name: 'HTTP Detail Monitor', level: 1 }),
        ).toBeVisible();
        const httpSummary = page.getByTestId('monitor-check-summary');
        await expect(httpSummary).toContainText('GET');
        await expect(httpSummary).toContainText(`${fixtureBaseUrl}/ok`);
        await expect(httpSummary).toContainText('200');

        await page.getByRole('button', { name: /detailed config/i }).click();
        const httpDetails = page.getByTestId('monitor-check-details');
        await expect(httpDetails).toContainText('Target');
        await expect(httpDetails).toContainText(`${fixtureBaseUrl}/ok`);

        await page.goto(`/monitors/${transactionMonitor.id}`);
        await expect(
            page.getByRole('heading', {
                name: 'Transaction Detail Monitor',
                level: 1,
            }),
        ).toBeVisible();
        const transactionSummary = page.getByTestId('monitor-check-summary');
        await expect(transactionSummary).toContainText('2 steps');
        await expect(transactionSummary).toContainText(`${fixtureBaseUrl}/ok`);

        await page.getByRole('button', { name: /detailed config/i }).click();
        await expect(page.getByTestId('monitor-detail-transaction-step-1')).toContainText(
            `${fixtureBaseUrl}/ok`,
        );
        await expect(page.getByTestId('monitor-detail-transaction-step-2')).toContainText(
            `${fixtureBaseUrl}/slow`,
        );
    });

    test('monitor details surface latest HTTPS certificate basics', async ({ page }) => {
        const monitorId = 'https-runtime-monitor';
        const monitorUrl = `${appBaseUrl}/api/v1/monitors/${monitorId}`;
        const checksUrl = `${appBaseUrl}/api/v1/monitors/${monitorId}/checks`;
        const uptimeUrl = `${appBaseUrl}/api/v1/monitors/${monitorId}/uptime`;
        const eventsUrl = `${appBaseUrl}/api/v1/monitors/${monitorId}/events?limit=5`;

        await loginThroughUI(page);

        await page.route(monitorUrl, async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    id: monitorId,
                    name: 'HTTPS Runtime Monitor',
                    type: 'https',
                    config: {
                        url: 'https://secure.example.test/health',
                        method: 'GET',
                        expected_status: 200,
                        warn_days: 14,
                    },
                    groups: ['Core'],
                    interval_s: 60,
                    enabled: true,
                    status: 'degraded',
                    last_check: '2026-04-18T12:00:00Z',
                    last_latency_ms: 182,
                }),
            });
        });

        await page.route(checksUrl, async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([
                    {
                        id: 1,
                        monitor_id: monitorId,
                        status: 'degraded',
                        latency_ms: 182,
                        status_code: 200,
                        message: 'TLS certificate expires in 9 day(s)',
                        metadata: {
                            cert_not_after: '2026-04-27T00:00:00Z',
                            cert_not_before: '2026-01-01T00:00:00Z',
                            cert_days_remaining: 9,
                            cert_subject: 'CN=secure.example.test',
                            cert_issuer: 'CN=Acme Root',
                            cert_warn_days: 14,
                            cert_serial_number: '01A2B3C4',
                            cert_fingerprint_sha256:
                                '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
                            cert_signature_algorithm: 'SHA256-RSA',
                            cert_public_key_algorithm: 'RSA',
                            cert_public_key_bits: 2048,
                            cert_dns_names: ['secure.example.test', 'api.example.test'],
                            cert_ip_addresses: ['127.0.0.1'],
                            cert_tls_verification_mode: 'skipped',
                            cert_tls_verified: false,
                            cert_chain_length: 2,
                            cert_chain_summary: [
                                'CN=secure.example.test',
                                'CN=Acme Root',
                            ],
                        },
                        checked_at: '2026-04-18T12:00:00Z',
                    },
                ]),
            });
        });

        await page.route(uptimeUrl, async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    '24h': 99.98,
                    '7d': 99.91,
                    '30d': 99.87,
                }),
            });
        });

        await page.route(eventsUrl, async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([]),
            });
        });

        await page.goto(`/monitors/${monitorId}`);
        await expect(
            page.getByRole('heading', { name: 'HTTPS Runtime Monitor', level: 1 }),
        ).toBeVisible();

        const basics = page.getByTestId('monitor-current-basics');
        await expect(basics).toContainText('Certificate Expires');
        await expect(page.getByTestId('monitor-basic-certificate-expires')).toContainText(
            '2026-04-27',
        );
        await expect(page.getByTestId('monitor-basic-days-left')).toContainText('9');
        await expect(basics).toContainText('Verification');
        await expect(basics).toContainText('Skipped');

        await page.getByRole('button', { name: /detailed config/i }).click();
        const details = page.getByTestId('monitor-check-details');
        await expect(details).toContainText('Latest Certificate');
        await expect(details).toContainText('CN=secure.example.test');
        await expect(details).toContainText('CN=Acme Root');
        await expect(details).toContainText('RSA');
        await expect(details).toContainText('2048');
        await expect(details).toContainText('secure.example.test');
        await expect(details).toContainText('api.example.test');
    });

    test('push monitor exposes copyable ping endpoint', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();
        const pushMonitor = await createMonitor(api, {
            name: 'Push Detail Monitor',
            type: 'push',
            config: {
                token: 'e2e-push-token-abc123',
                grace_period_s: 60,
            },
        });
        await api.dispose();

        await loginThroughUI(page);
        await page.goto(`/monitors/${pushMonitor.id}`);
        await expect(
            page.getByRole('heading', { name: 'Push Detail Monitor', level: 1 }),
        ).toBeVisible();

        const pingUrl = page.getByTestId('monitor-push-url');
        await expect(pingUrl).toBeVisible();
        await expect(pingUrl).toContainText('/heartbeat/e2e-push-token-abc123');

        const curlSnippet = page.getByTestId('monitor-push-curl');
        await expect(curlSnippet).toContainText('/heartbeat/e2e-push-token-abc123');
        await expect(curlSnippet).not.toContainText(`/api/v1/heartbeat/${pushMonitor.id}`);

        await page.getByRole('button', { name: /detailed config/i }).click();
        const details = page.getByTestId('monitor-check-details');
        await expect(details).toContainText('Slug Endpoint');
        await expect(details).toContainText('POST only');

        await expect(
            page.getByRole('button', { name: /copy.*heartbeat url/i }).first(),
        ).toBeVisible();
    });

    test('composite monitor links to child monitors', async ({ page }) => {
        const api = await createAuthenticatedRequestContext();
        const child = await createMonitor(api, {
            name: 'Composite Child Monitor',
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });
        const composite = await createMonitor(api, {
            name: 'Composite Detail Monitor',
            type: 'composite',
            config: {
                mode: 'all_up',
                monitor_ids: [child.id],
            },
        });
        await api.dispose();

        await loginThroughUI(page);
        await page.goto(`/monitors/${composite.id}`);
        await expect(
            page.getByRole('heading', { name: 'Composite Detail Monitor', level: 1 }),
        ).toBeVisible();

        const members = page.getByTestId('monitor-composite-members');
        await expect(members).toBeVisible();
        await expect(
            members.getByRole('link', { name: child.id }),
        ).toHaveAttribute('href', `/monitors/${child.id}`);
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