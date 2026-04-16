import path from 'node:path';

export const appHost = process.env.UPDU_E2E_HOST ?? '127.0.0.1';
export const appPort = process.env.UPDU_E2E_PORT ?? '4010';
export const appBaseUrl =
    process.env.UPDU_E2E_BASE_URL ?? `http://${appHost}:${appPort}`;
export const fixtureBaseUrl =
    process.env.UPDU_E2E_FIXTURE_URL ?? 'http://127.0.0.1:4011';
export const adminUsername = process.env.UPDU_E2E_ADMIN_USER ?? 'admin';
export const adminPassword =
    process.env.UPDU_E2E_ADMIN_PASSWORD ?? 'password123';
export const authStorageStatePath = path.join(
    process.cwd(),
    'e2e/.auth/admin.json',
);