import { expect, test, type Page } from '@playwright/test';
import { createAuthenticatedRequestContext } from './helpers/auth';
import {
    clearIncidents,
    clearMonitors,
    clearStatusPages,
    createMonitor,
} from './helpers/api';
import { fixtureBaseUrl } from './helpers/env';

const scrollableMonitorCount = 14;
const layoutWidths = [320, 768, 1440, 1920];
const statsTabs = ['Overview', 'Performance', 'Monitors', 'Incidents'];

async function seedScrollableStats(): Promise<void> {
    const api = await createAuthenticatedRequestContext();

    for (let index = 1; index <= scrollableMonitorCount; index += 1) {
        await createMonitor(api, {
            name: `Stats Layout Monitor ${index}`,
            type: 'http',
            config: {
                url: `${fixtureBaseUrl}/ok`,
                method: 'GET',
                expected_status: 200,
            },
        });
    }

    await api.dispose();
}

async function clearFixtures(): Promise<void> {
    const api = await createAuthenticatedRequestContext();
    await clearStatusPages(api);
    await clearIncidents(api);
    await clearMonitors(api);
    await api.dispose();
}

async function documentOverflow(page: Page): Promise<number> {
    return page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
}

test.describe('stats page layout', () => {
    test.beforeEach(async () => {
        await clearFixtures();
        await seedScrollableStats();
    });

    test.afterEach(async () => {
        await clearFixtures();
    });

    test('keeps analytics tabs reachable after scrolling on mobile', async ({ page }) => {
        await page.setViewportSize({ width: 375, height: 667 });
        await page.goto('/stats');

        const tablist = page.getByRole('tablist', {
            name: 'Analytics sections',
        });
        await expect(tablist).toBeVisible();

        await page.getByRole('tab', { name: /Monitors/ }).click();
        await expect(
            page.getByRole('tab', { name: /Monitors/, selected: true }),
        ).toBeVisible();

        await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));

        await expect(tablist).toBeInViewport();
        await page.getByRole('tab', { name: /Performance/ }).click();
        await expect(
            page.getByRole('tab', { name: /Performance/, selected: true }),
        ).toBeVisible();
    });

    test('does not create document-level horizontal overflow', async ({ page }) => {
        for (const width of layoutWidths) {
            await page.setViewportSize({ width, height: 800 });
            await page.goto('/stats');

            for (const tabName of statsTabs) {
                await page.getByRole('tab', { name: new RegExp(tabName) }).click();
                await expect(await documentOverflow(page)).toBeLessThanOrEqual(1);
            }
        }
    });
});
