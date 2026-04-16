import { expect, request, type APIRequestContext, type Page } from '@playwright/test';
import { authStorageStatePath, adminPassword, adminUsername, appBaseUrl } from './env';

export async function loginThroughUI(page: Page): Promise<void> {
    await page.goto('/login');
    await expect(
        page.getByRole('heading', { name: /sign in to updu/i }),
    ).toBeVisible();
    await page.getByLabel('Username').fill(adminUsername);
    await page.getByLabel('Password').fill(adminPassword);

    await Promise.all([
        page.waitForURL((url) => !url.pathname.endsWith('/login')),
        page.getByRole('button', { name: /^sign in$/i }).click(),
    ]);

    await expect(
        page.getByRole('button', { name: /sign out/i }),
    ).toBeVisible({ timeout: 10000 });
}

export async function createAuthenticatedRequestContext(): Promise<APIRequestContext> {
    return request.newContext({
        baseURL: appBaseUrl,
        storageState: authStorageStatePath,
    });
}