import { defineConfig, devices } from '@playwright/test';

const appHost = process.env.UPDU_E2E_HOST ?? '127.0.0.1';
const appPort = process.env.UPDU_E2E_PORT ?? '4010';
const appBaseUrl =
    process.env.UPDU_E2E_BASE_URL ?? `http://${appHost}:${appPort}`;
const fixturePort = process.env.UPDU_E2E_FIXTURE_PORT ?? '4011';
const fixtureBaseUrl =
    process.env.UPDU_E2E_FIXTURE_URL ?? `http://127.0.0.1:${fixturePort}`;
const authStorageState = 'e2e/.auth/admin.json';

export default defineConfig({
    testDir: './e2e',
    testIgnore: ['**/helpers/**', '**/scripts/**'],
    fullyParallel: false,
    forbidOnly: Boolean(process.env.CI),
    retries: process.env.CI ? 1 : 0,
    workers: 1,
    reporter: [
        ['list'],
        ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ],
    outputDir: 'test-results',
    use: {
        baseURL: appBaseUrl,
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
    },
    projects: [
        {
            name: 'setup',
            testMatch: /auth\.setup\.ts/,
        },
        {
            name: 'chromium',
            use: {
                ...devices['Desktop Chrome'],
                storageState: authStorageState,
            },
            dependencies: ['setup'],
        },
    ],
    webServer: [
        {
            command: 'node ./e2e/scripts/target-server.mjs',
            url: `${fixtureBaseUrl}/healthz`,
            timeout: 30_000,
            reuseExistingServer: false,
            env: {
                ...process.env,
                UPDU_E2E_FIXTURE_PORT: fixturePort,
            },
        },
        {
            command: 'bash ./e2e/scripts/start-updu.sh',
            url: `${appBaseUrl}/healthz`,
            timeout: 180_000,
            reuseExistingServer: false,
            env: {
                ...process.env,
                UPDU_E2E_HOST: appHost,
                UPDU_E2E_PORT: appPort,
                UPDU_E2E_BASE_URL: appBaseUrl,
                UPDU_E2E_FIXTURE_URL: fixtureBaseUrl,
                UPDU_E2E_AUTH_SECRET:
                    process.env.UPDU_E2E_AUTH_SECRET ??
                    'updu-playwright-auth-secret',
                UPDU_E2E_ADMIN_USER:
                    process.env.UPDU_E2E_ADMIN_USER ?? 'admin',
                UPDU_E2E_ADMIN_PASSWORD:
                    process.env.UPDU_E2E_ADMIN_PASSWORD ?? 'password123',
                UPDU_E2E_LOG_LEVEL:
                    process.env.UPDU_E2E_LOG_LEVEL ?? 'warn',
            },
        },
    ],
});