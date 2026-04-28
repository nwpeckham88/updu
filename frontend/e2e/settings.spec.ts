import { expect, test, type Locator } from '@playwright/test';
import { createAuthenticatedRequestContext } from './helpers/auth';
import { fixtureBaseUrl } from './helpers/env';

interface NotificationChannelRecord {
    id: string;
    name: string;
}

interface APITokenRecord {
    id: string;
    name: string;
}

interface AdminUserRecord {
    id: string;
    username: string;
}

async function confirmDestructiveAction(
    dialog: Locator,
    buttonName: string,
): Promise<void> {
    const confirmButton = dialog.getByRole('button', { name: buttonName });
    await dialog.getByLabel('Type DELETE to confirm').fill('DELETE');
    await expect(confirmButton).toBeEnabled({ timeout: 5_000 });
    await confirmButton.click();
}

async function cleanupAPIToken(name: string) {
    const api = await createAuthenticatedRequestContext();

    try {
        const response = await api.get('/api/v1/admin/api-tokens');
        expect(response.ok()).toBeTruthy();

        const payload = (await response.json()) as APITokenRecord[] | null;
        const tokens = Array.isArray(payload) ? payload : [];

        for (const token of tokens.filter((candidate) => candidate.name === name)) {
            const deleteResponse = await api.delete(
                `/api/v1/admin/api-tokens/${token.id}`,
            );
            expect(deleteResponse.ok()).toBeTruthy();
        }
    } finally {
        await api.dispose();
    }
}

async function cleanupNotificationChannel(name: string) {
    const api = await createAuthenticatedRequestContext();

    try {
        const response = await api.get('/api/v1/notifications');
        expect(response.ok()).toBeTruthy();

        const payload = (await response.json()) as NotificationChannelRecord[] | null;
        const channels = Array.isArray(payload) ? payload : [];

        for (const channel of channels.filter((candidate) => candidate.name === name)) {
            const deleteResponse = await api.delete(
                `/api/v1/notifications/${channel.id}`,
            );
            expect(deleteResponse.ok()).toBeTruthy();
        }
    } finally {
        await api.dispose();
    }
}

async function cleanupUser(username: string) {
    const api = await createAuthenticatedRequestContext();

    try {
        const response = await api.get('/api/v1/admin/users');
        expect(response.ok()).toBeTruthy();

        const payload = (await response.json()) as AdminUserRecord[] | null;
        const users = Array.isArray(payload) ? payload : [];

        for (const user of users.filter((candidate) => candidate.username === username)) {
            const deleteResponse = await api.delete(`/api/v1/admin/users/${user.id}`);
            expect(deleteResponse.ok()).toBeTruthy();
        }
    } finally {
        await api.dispose();
    }
}

test.describe('settings system tools', () => {
    test('admin can manage api tokens and browse audit logs', async ({ page }) => {
        const tokenName = `Playwright token ${Date.now()}`;

        try {
            await page.goto('/settings/system');
            await expect(
                page.getByRole('heading', { name: 'System Update' }),
            ).toBeVisible();
            await expect(
                page.getByRole('heading', { name: 'API Tokens', exact: true }),
            ).toBeVisible({ timeout: 10000 });

            await page.getByTestId('create-api-token').click();

            const dialog = page.getByRole('dialog', {
                name: 'Create API Token',
            });
            await dialog.getByLabel('Token Name').fill(tokenName);
            await dialog.getByLabel('Access Scope').selectOption('write');

            const createResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/admin/api-tokens') &&
                    response.request().method() === 'POST',
            );
            await dialog.getByRole('button', { name: 'Create Token' }).click();
            const createResponse = await createResponsePromise;
            expect(createResponse.ok()).toBeTruthy();

            await expect(page.getByTestId('api-token-secret')).toContainText('updu_');

            const tokenRow = page
                .getByTestId('api-token-list')
                .locator('[data-testid="api-token-row"]')
                .filter({ hasText: tokenName });
            await expect(tokenRow).toContainText('write');

            const auditList = page.getByTestId('audit-log-list');
            await expect(auditList).toContainText('api_token.create');
            await expect(auditList).toContainText(tokenName);

            const revokeResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/admin/api-tokens/') &&
                    response.request().method() === 'DELETE',
            );
            await tokenRow.getByRole('button', { name: 'Revoke token' }).click();
            const revokeDialog = page.getByRole('dialog', {
                name: 'Revoke API Token',
            });
            await expect(revokeDialog).toContainText(tokenName);
            await confirmDestructiveAction(revokeDialog, 'Revoke Token');
            const revokeResponse = await revokeResponsePromise;
            expect(revokeResponse.ok()).toBeTruthy();

            await expect(tokenRow).toContainText('revoked');
            await expect(auditList).toContainText('api_token.revoke');
        } finally {
            await cleanupAPIToken(tokenName);
        }
    });

    test('backup import stages a review dialog before uploading', async ({ page }) => {
        await page.goto('/settings/backup');
        await expect(
            page.getByRole('heading', { name: 'Backups & Export' }),
        ).toBeVisible();

        await page.locator('input[type="file"]').setInputFiles({
            name: 'updu-backup.json',
            mimeType: 'application/json',
            buffer: Buffer.from('{"settings":{}}'),
        });

        const importDialog = page.getByRole('dialog', {
            name: 'Import Configuration Backup',
        });
        await expect(importDialog).toContainText('updu-backup.json');
        await expect(importDialog).toContainText('Import Backup');
        await importDialog.getByRole('button', { name: 'Cancel' }).click();
    });

    test('admin can manage notification channels with confirmation dialogs', async ({ page }) => {
        await page.goto('/settings/notifications');
        await expect(
            page.getByRole('heading', {
                name: 'Notification Channels',
                exact: true,
            }),
        ).toBeVisible();

        const channelName = `Playwright channel ${Date.now()}`;
        const channelUrl = `${fixtureBaseUrl}/ok?channel=${Date.now()}`;

        try {
            await page.getByRole('button', { name: 'New Channel' }).click();

            const dialog = page.getByRole('dialog', {
                name: 'New Channel',
            });
            await dialog.getByLabel('Name').fill(channelName);
            await dialog.getByLabel('URL').fill(channelUrl);

            const createResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/notifications') &&
                    response.request().method() === 'POST',
            );
            await dialog.getByRole('button', { name: 'Create Channel' }).click();
            const createResponse = await createResponsePromise;
            expect(createResponse.ok()).toBeTruthy();

            const channelRow = page
                .getByTestId('notification-channel-row')
                .filter({ hasText: channelName });
            await expect(channelRow).toContainText('127.0.0.1:4011');

            await channelRow.getByRole('button', { name: 'Delete' }).click();
            const deleteDialog = page.getByRole('dialog', {
                name: 'Delete Notification Channel',
            });
            await expect(deleteDialog).toContainText(channelName);
            await expect(deleteDialog).toContainText('127.0.0.1:4011');
            await deleteDialog.getByRole('button', { name: 'Cancel' }).click();
            await expect(channelRow).toBeVisible();

            const deleteResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/notifications/') &&
                    response.request().method() === 'DELETE',
            );
            await channelRow.getByRole('button', { name: 'Delete' }).click();
            await confirmDestructiveAction(
                page.getByRole('dialog', { name: 'Delete Notification Channel' }),
                'Delete Channel',
            );
            const deleteResponse = await deleteResponsePromise;
            expect(deleteResponse.ok()).toBeTruthy();

            await expect(channelRow).toHaveCount(0);
        } finally {
            await cleanupNotificationChannel(channelName);
        }
    });

    test('admin can manage users with confirmation dialogs', async ({ page }) => {
        await page.goto('/settings/users');
        await expect(
            page.getByRole('heading', { name: 'Users & Roles' }),
        ).toBeVisible();

        const username = `playwright-user-${Date.now()}`;
        const password = 'password123';

        try {
            await page.getByRole('button', { name: 'Invite User' }).click();

            const dialog = page.getByRole('dialog', {
                name: 'Invite User',
            });
            await dialog.getByLabel('Username').fill(username);
            await dialog.getByLabel('Password').fill(password);

            const createResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/auth/register') &&
                    response.request().method() === 'POST',
            );
            await dialog.getByRole('button', { name: 'Create User' }).click();
            const createResponse = await createResponsePromise;
            expect(createResponse.ok()).toBeTruthy();

            const userRow = page.getByTestId('user-row').filter({ hasText: username });
            await expect(userRow).toContainText('Viewer');

            await userRow.getByRole('button', { name: 'Make Admin' }).click();
            const roleDialog = page.getByRole('dialog', {
                name: 'Change User Role',
            });
            await expect(roleDialog).toContainText(username);
            await expect(roleDialog).toContainText('admin');
            await roleDialog.getByRole('button', { name: 'Cancel' }).click();
            await expect(userRow).toContainText('Viewer');

            const roleResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/admin/users/') &&
                    response.url().includes('/role') &&
                    response.request().method() === 'PUT',
            );
            await userRow.getByRole('button', { name: 'Make Admin' }).click();
            await page
                .getByRole('dialog', { name: 'Change User Role' })
                .getByRole('button', { name: 'Apply Role Change' })
                .click();
            const roleResponse = await roleResponsePromise;
            expect(roleResponse.ok()).toBeTruthy();

            await expect(userRow).toContainText('Admin', { timeout: 5000 });
            await expect(
                userRow.getByRole('button', { name: 'Make Viewer' }),
            ).toBeVisible();

            const deleteResponsePromise = page.waitForResponse(
                (response) =>
                    response.url().includes('/api/v1/admin/users/') &&
                    !response.url().includes('/role') &&
                    response.request().method() === 'DELETE',
            );
            await userRow.getByRole('button', { name: 'Delete' }).click();
            const deleteDialog = page.getByRole('dialog', {
                name: 'Delete User',
            });
            await expect(deleteDialog).toContainText(username);
            await confirmDestructiveAction(deleteDialog, 'Delete User');
            const deleteResponse = await deleteResponsePromise;
            expect(deleteResponse.ok()).toBeTruthy();

            await expect(userRow).toHaveCount(0);
        } finally {
            await cleanupUser(username);
        }
    });
});