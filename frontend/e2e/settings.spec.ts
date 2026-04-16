import { expect, test } from '@playwright/test';
import { loginThroughUI } from './helpers/auth';

test.describe('settings system tools', () => {
    test('admin can manage api tokens and browse audit logs', async ({ page }) => {
        await loginThroughUI(page);
        await page.goto('/settings');

        await page.getByRole('tab', { name: 'System' }).click();
        await expect(
            page.getByRole('heading', { name: 'API Tokens', exact: true }),
        ).toBeVisible();

        const tokenName = `Playwright token ${Date.now()}`;

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
        page.once('dialog', async (dialogEvent) => dialogEvent.accept());
        await tokenRow.getByRole('button', { name: 'Revoke token' }).click();
        const revokeResponse = await revokeResponsePromise;
        expect(revokeResponse.ok()).toBeTruthy();

        await expect(tokenRow).toContainText('revoked');
        await expect(auditList).toContainText('api_token.revoke');
    });
});