import { expect, test } from '@playwright/test';

test.describe('settings section routing', () => {
    test('redirects /settings to the general section', async ({ page }) => {
        await page.goto('/settings');

        await expect(page).toHaveURL(/\/settings\/general$/);
        await expect(
            page.getByRole('heading', { name: 'Instance Profile' }),
        ).toBeVisible();
    });

    test('supports deep linking and section navigation', async ({ page }) => {
        await page.goto('/settings/general');

        await expect(page).toHaveURL(/\/settings\/general$/);
        await expect(
            page.getByRole('heading', { name: 'Instance Profile' }),
        ).toBeVisible();

        await page.getByRole('link', { name: 'System' }).click();
        await expect(page).toHaveURL(/\/settings\/system$/);
        await expect(
            page.getByRole('heading', { name: 'System Update' }),
        ).toBeVisible();

        await page.getByRole('link', { name: 'Notifications' }).click();
        await expect(page).toHaveURL(/\/settings\/notifications$/);
        await expect(
            page.getByRole('button', { name: 'New Channel' }),
        ).toBeVisible();
    });
});