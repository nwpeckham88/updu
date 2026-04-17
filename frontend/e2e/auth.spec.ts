import { expect, test } from '@playwright/test';
import { authMode, expectedUsername } from './helpers/env';
import { loginThroughUI } from './helpers/auth';

test.use({ storageState: { cookies: [], origins: [] } });

test('login, session persistence, and logout work', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    if (authMode === 'oidc') {
        await expect(
            page.getByRole('link', { name: /single sign-on \(oidc\)/i }),
        ).toBeVisible();
    } else {
        await expect(
            page.getByRole('heading', { name: /sign in to updu/i }),
        ).toBeVisible();
    }

    await loginThroughUI(page);
    await expect(page.getByText(expectedUsername, { exact: true })).toBeVisible();

    await page.goto('/monitors');
    await expect(page).toHaveURL(/\/monitors$/);
    await expect(
        page.getByRole('heading', { name: 'Monitors', level: 1 }),
    ).toBeVisible();

    await page.reload();
    await expect(page.getByText(expectedUsername, { exact: true })).toBeVisible();

    await page.getByRole('button', { name: /sign out/i }).click();
    await expect(page).toHaveURL(/\/login$/);
    await expect(
        page.getByRole('heading', { name: /sign in to updu/i }),
    ).toBeVisible();
    if (authMode === 'oidc') {
        await expect(
            page.getByRole('link', { name: /single sign-on \(oidc\)/i }),
        ).toBeVisible();
    }
});