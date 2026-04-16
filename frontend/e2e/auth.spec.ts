import { expect, test } from '@playwright/test';
import { adminUsername } from './helpers/env';
import { loginThroughUI } from './helpers/auth';

test.use({ storageState: { cookies: [], origins: [] } });

test('login, session persistence, and logout work', async ({ page }) => {
    await page.goto('/login');
    await expect(
        page.getByRole('heading', { name: /sign in to updu/i }),
    ).toBeVisible();

    await loginThroughUI(page);
    await expect(page.getByText(adminUsername)).toBeVisible();

    await page.getByRole('link', { name: 'Monitors' }).click();
    await expect(page).toHaveURL(/\/monitors$/);
    await expect(
        page.getByRole('heading', { name: 'Monitors' }),
    ).toBeVisible();

    await page.reload();
    await expect(page.getByText(adminUsername)).toBeVisible();

    await page.getByRole('button', { name: /sign out/i }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(
        page.getByRole('heading', { name: /sign in to updu/i }),
    ).toBeVisible();
});