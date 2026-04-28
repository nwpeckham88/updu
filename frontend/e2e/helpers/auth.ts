import { expect, request, type APIRequestContext, type Page } from '@playwright/test';
import {
    authMode,
    authStorageStatePath,
    adminPassword,
    adminUsername,
    appBaseUrl,
} from './env';

const sessionCookieName = 'updu_session';

export async function loginThroughUI(page: Page): Promise<void> {
    if (await reuseExistingSession(page)) {
        return;
    }

    if (authMode === 'oidc') {
        await loginThroughOIDC(page);
        return;
    }

    await loginThroughPassword(page);
}

async function reuseExistingSession(page: Page): Promise<boolean> {
    const cookies = await page.context().cookies(appBaseUrl);
    const hasSessionCookie = cookies.some(
        (cookie) => cookie.name === sessionCookieName && cookie.value.length > 0,
    );

    if (!hasSessionCookie) {
        return false;
    }

    await page.goto('/');

    try {
        await page
            .getByRole('button', { name: /sign out/i })
            .waitFor({ state: 'visible', timeout: 10_000 });
        return true;
    } catch {
        return false;
    }
}

export async function loginThroughPassword(page: Page): Promise<void> {
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

export async function loginThroughOIDC(page: Page): Promise<void> {
    await page.goto('/login');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

    const oidcButton = page.getByRole('link', {
        name: /single sign-on \(oidc\)/i,
    });
    await expect(oidcButton).toBeVisible();

    await oidcButton.click();

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