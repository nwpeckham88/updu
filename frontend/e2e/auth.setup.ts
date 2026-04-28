import fs from 'node:fs/promises';
import path from 'node:path';
import { request, test as setup } from '@playwright/test';
import { loginThroughUI } from './helpers/auth';
import { appBaseUrl, authStorageStatePath } from './helpers/env';

async function hasReusableStorageState(): Promise<boolean> {
    try {
        await fs.access(authStorageStatePath);
    } catch {
        return false;
    }

    const api = await request.newContext({
        baseURL: appBaseUrl,
        storageState: authStorageStatePath,
    });

    try {
        const response = await api.get('/api/v1/auth/session');
        return response.ok();
    } finally {
        await api.dispose();
    }
}

setup('store authenticated admin session', async ({ page }) => {
    await fs.mkdir(path.dirname(authStorageStatePath), { recursive: true });

    if (await hasReusableStorageState()) {
        return;
    }

    await loginThroughUI(page);
    await page.context().storageState({ path: authStorageStatePath });
});