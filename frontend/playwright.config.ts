import { defineConfig, devices } from '@playwright/test';

const authMode = process.env.UPDU_E2E_AUTH_MODE ?? 'password';
const isOIDCAuth = authMode === 'oidc';
const appHost = process.env.UPDU_E2E_HOST ?? '127.0.0.1';
const appPort = process.env.UPDU_E2E_PORT ?? '4010';
const appBaseUrl =
    process.env.UPDU_E2E_BASE_URL ?? `http://${appHost}:${appPort}`;
const fixturePort = process.env.UPDU_E2E_FIXTURE_PORT ?? '4011';
const fixtureBaseUrl =
    process.env.UPDU_E2E_FIXTURE_URL ?? `http://127.0.0.1:${fixturePort}`;
const oidcPort = process.env.UPDU_E2E_OIDC_PORT ?? '4012';
const oidcIssuer =
    process.env.UPDU_E2E_OIDC_ISSUER ?? `http://127.0.0.1:${oidcPort}`;
const oidcRedirectUrl =
    process.env.UPDU_E2E_OIDC_REDIRECT_URL ??
    `${appBaseUrl}/api/v1/auth/oidc/callback`;
const shouldStartMockOIDC = isOIDCAuth && !process.env.UPDU_E2E_OIDC_ISSUER;
const authStorageState = 'e2e/.auth/admin.json';

const webServers = [
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
];

if (shouldStartMockOIDC) {
    webServers.push({
        command: 'node ./e2e/scripts/mock-oidc-server.mjs',
        url: `${oidcIssuer}/healthz`,
        timeout: 30_000,
        reuseExistingServer: false,
        env: {
            ...process.env,
            UPDU_E2E_OIDC_PORT: oidcPort,
            UPDU_E2E_OIDC_ISSUER: oidcIssuer,
            UPDU_E2E_OIDC_REDIRECT_URL: oidcRedirectUrl,
            UPDU_E2E_OIDC_CLIENT_ID:
                process.env.UPDU_E2E_OIDC_CLIENT_ID ??
                'updu-playwright-client',
            UPDU_E2E_OIDC_CLIENT_SECRET:
                process.env.UPDU_E2E_OIDC_CLIENT_SECRET ??
                'updu-playwright-secret',
            UPDU_E2E_OIDC_USERNAME:
                process.env.UPDU_E2E_OIDC_USERNAME ??
                process.env.UPDU_E2E_ADMIN_USER ??
                'admin',
            UPDU_E2E_OIDC_EMAIL:
                process.env.UPDU_E2E_OIDC_EMAIL ?? 'admin@example.test',
            UPDU_E2E_OIDC_SUB:
                process.env.UPDU_E2E_OIDC_SUB ??
                'updu-playwright-oidc-sub',
        },
    });
}

webServers.push({
    command: 'bash ./e2e/scripts/start-updu.sh',
    url: `${appBaseUrl}/healthz`,
    timeout: 180_000,
    reuseExistingServer: false,
    env: {
        ...process.env,
        UPDU_E2E_AUTH_MODE: authMode,
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
        UPDU_E2E_LOG_LEVEL: process.env.UPDU_E2E_LOG_LEVEL ?? 'warn',
        UPDU_E2E_OIDC_PORT: oidcPort,
        UPDU_E2E_OIDC_ISSUER: oidcIssuer,
        UPDU_E2E_OIDC_REDIRECT_URL: oidcRedirectUrl,
        UPDU_E2E_OIDC_CLIENT_ID:
            process.env.UPDU_E2E_OIDC_CLIENT_ID ?? 'updu-playwright-client',
        UPDU_E2E_OIDC_CLIENT_SECRET:
            process.env.UPDU_E2E_OIDC_CLIENT_SECRET ??
            'updu-playwright-secret',
        UPDU_E2E_OIDC_USERNAME:
            process.env.UPDU_E2E_OIDC_USERNAME ??
            process.env.UPDU_E2E_ADMIN_USER ??
            'admin',
        UPDU_E2E_OIDC_EMAIL:
            process.env.UPDU_E2E_OIDC_EMAIL ?? 'admin@example.test',
        UPDU_E2E_OIDC_SUB:
            process.env.UPDU_E2E_OIDC_SUB ?? 'updu-playwright-oidc-sub',
    },
});

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
    webServer: webServers,
});