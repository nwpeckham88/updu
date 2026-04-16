import fs from 'node:fs/promises';
import path from 'node:path';
import { test as setup } from '@playwright/test';
import { loginThroughUI } from './helpers/auth';
import { authStorageStatePath } from './helpers/env';

setup('store authenticated admin session', async ({ page }) => {
    await fs.mkdir(path.dirname(authStorageStatePath), { recursive: true });
    await loginThroughUI(page);
    await page.context().storageState({ path: authStorageStatePath });
});