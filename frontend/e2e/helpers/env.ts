import path from 'node:path';

export type E2EAuthMode = 'password' | 'oidc';

const requestedAuthMode = process.env.UPDU_E2E_AUTH_MODE;

export const authMode: E2EAuthMode =
    requestedAuthMode === 'oidc' ? 'oidc' : 'password';
export const isOIDCAuth = authMode === 'oidc';

export const appHost = process.env.UPDU_E2E_HOST ?? '127.0.0.1';
export const appPort = process.env.UPDU_E2E_PORT ?? '4010';
export const appBaseUrl =
    process.env.UPDU_E2E_BASE_URL ?? `http://${appHost}:${appPort}`;
export const fixtureBaseUrl =
    process.env.UPDU_E2E_FIXTURE_URL ?? 'http://127.0.0.1:4011';
export const adminUsername = process.env.UPDU_E2E_ADMIN_USER ?? 'admin';
export const adminPassword =
    process.env.UPDU_E2E_ADMIN_PASSWORD ?? 'password123';
export const oidcPort = process.env.UPDU_E2E_OIDC_PORT ?? '4012';
export const oidcIssuer =
    process.env.UPDU_E2E_OIDC_ISSUER ?? `http://127.0.0.1:${oidcPort}`;
export const oidcClientId =
    process.env.UPDU_E2E_OIDC_CLIENT_ID ?? 'updu-playwright-client';
export const oidcClientSecret =
    process.env.UPDU_E2E_OIDC_CLIENT_SECRET ?? 'updu-playwright-secret';
export const oidcUsername =
    process.env.UPDU_E2E_OIDC_USERNAME ?? adminUsername;
export const expectedUsername = isOIDCAuth ? oidcUsername : adminUsername;
export const authStorageStatePath = path.join(
    process.cwd(),
    'e2e/.auth/admin.json',
);