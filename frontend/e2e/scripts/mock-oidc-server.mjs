import http from 'node:http';
import { createSign, generateKeyPairSync, randomBytes } from 'node:crypto';

const port = Number.parseInt(process.env.UPDU_E2E_OIDC_PORT ?? '4012', 10);
const issuerBaseUrl =
    process.env.UPDU_E2E_OIDC_ISSUER ?? `http://127.0.0.1:${port}`;
const clientId =
    process.env.UPDU_E2E_OIDC_CLIENT_ID ?? 'updu-playwright-client';
const clientSecret =
    process.env.UPDU_E2E_OIDC_CLIENT_SECRET ?? 'updu-playwright-secret';
const expectedRedirectUrl =
    process.env.UPDU_E2E_OIDC_REDIRECT_URL ??
    'http://127.0.0.1:4010/api/v1/auth/oidc/callback';
const username =
    process.env.UPDU_E2E_OIDC_USERNAME ??
    process.env.UPDU_E2E_ADMIN_USER ??
    'admin';
const email =
    process.env.UPDU_E2E_OIDC_EMAIL ?? `${username}@example.test`;
const subject =
    process.env.UPDU_E2E_OIDC_SUB ?? 'updu-playwright-oidc-sub';
const displayName = process.env.UPDU_E2E_OIDC_NAME ?? username;
const keyId = 'updu-playwright-oidc-key';

const { privateKey, publicKey } = generateKeyPairSync('rsa', {
    modulusLength: 2048,
});

const publicJwk = publicKey.export({ format: 'jwk' });
publicJwk.alg = 'RS256';
publicJwk.kid = keyId;
publicJwk.use = 'sig';

const authorizationCodes = new Map();

function includesScope(scopeList, expectedScope) {
    return scopeList.split(/\s+/).includes(expectedScope);
}

function hasRequiredScopes(scopeList) {
    const scopes = scopeList.split(/\s+/).filter(Boolean);
    const expectedScopes = ['openid', 'profile', 'email'];

    return (
        scopes.length === expectedScopes.length &&
        expectedScopes.every((scope) => includesScope(scopeList, scope))
    );
}

function encodeBase64Url(value) {
    return Buffer.from(value).toString('base64url');
}

function signJwt(claims) {
    const header = { alg: 'RS256', kid: keyId, typ: 'JWT' };
    const encodedHeader = encodeBase64Url(JSON.stringify(header));
    const encodedPayload = encodeBase64Url(JSON.stringify(claims));
    const signer = createSign('RSA-SHA256');
    signer.update(`${encodedHeader}.${encodedPayload}`);
    signer.end();
    const signature = signer.sign(privateKey).toString('base64url');
    return `${encodedHeader}.${encodedPayload}.${signature}`;
}

function writeJson(response, statusCode, payload) {
    response.writeHead(statusCode, {
        'content-type': 'application/json; charset=utf-8',
    });
    response.end(JSON.stringify(payload));
}

function writeText(response, statusCode, body) {
    response.writeHead(statusCode, {
        'content-type': 'text/plain; charset=utf-8',
    });
    response.end(body);
}

async function readRequestBody(request) {
    const chunks = [];
    for await (const chunk of request) {
        chunks.push(chunk);
    }
    return Buffer.concat(chunks).toString('utf8');
}

function getClientCredentials(request, form) {
    const authorization = request.headers.authorization;
    if (authorization?.startsWith('Basic ')) {
        const decoded = Buffer.from(
            authorization.slice('Basic '.length),
            'base64',
        ).toString('utf8');
        const separator = decoded.indexOf(':');
        if (separator >= 0) {
            return {
                clientId: decoded.slice(0, separator),
                clientSecret: decoded.slice(separator + 1),
            };
        }
    }

    return {
        clientId: form.get('client_id') ?? '',
        clientSecret: form.get('client_secret') ?? '',
    };
}

const server = http.createServer(async (request, response) => {
    const requestUrl = new URL(request.url ?? '/', issuerBaseUrl);

    if (requestUrl.pathname === '/healthz') {
        writeText(response, 200, 'ok');
        return;
    }

    if (requestUrl.pathname === '/.well-known/openid-configuration') {
        writeJson(response, 200, {
            issuer: issuerBaseUrl,
            authorization_endpoint: `${issuerBaseUrl}/authorize`,
            token_endpoint: `${issuerBaseUrl}/token`,
            jwks_uri: `${issuerBaseUrl}/jwks`,
            response_types_supported: ['code'],
            subject_types_supported: ['public'],
            id_token_signing_alg_values_supported: ['RS256'],
            token_endpoint_auth_methods_supported: [
                'client_secret_basic',
                'client_secret_post',
            ],
            scopes_supported: ['openid', 'profile', 'email'],
        });
        return;
    }

    if (requestUrl.pathname === '/authorize') {
        if (request.method !== 'GET') {
            writeText(response, 405, 'method not allowed');
            return;
        }

        const redirectUri = requestUrl.searchParams.get('redirect_uri');
        const state = requestUrl.searchParams.get('state');
        const nonce = requestUrl.searchParams.get('nonce');
        const requestedClientId = requestUrl.searchParams.get('client_id');
        const responseType = requestUrl.searchParams.get('response_type');
        const scope = requestUrl.searchParams.get('scope') ?? '';

        if (requestedClientId !== clientId) {
            writeText(response, 401, 'invalid client_id');
            return;
        }
        if (responseType !== 'code') {
            writeText(response, 400, 'invalid response_type');
            return;
        }

        if (!redirectUri || !state || !nonce) {
            writeText(response, 400, 'missing authorize parameters');
            return;
        }
        if (redirectUri !== expectedRedirectUrl) {
            writeText(response, 400, 'invalid redirect_uri');
            return;
        }
        if (!hasRequiredScopes(scope)) {
            writeText(response, 400, 'missing required scopes');
            return;
        }

        let redirectTarget;
        try {
            redirectTarget = new URL(redirectUri);
        } catch {
            writeText(response, 400, 'invalid redirect_uri');
            return;
        }

        const authorizationCode = randomBytes(18).toString('hex');
        authorizationCodes.set(authorizationCode, {
            nonce,
            redirectUri,
        });

        redirectTarget.searchParams.set('code', authorizationCode);
        redirectTarget.searchParams.set('state', state);

        response.writeHead(302, { location: redirectTarget.toString() });
        response.end();
        return;
    }

    if (requestUrl.pathname === '/token') {
        if (request.method !== 'POST') {
            writeText(response, 405, 'method not allowed');
            return;
        }

        const requestBody = await readRequestBody(request);
        const form = new URLSearchParams(requestBody);
        if (form.get('grant_type') !== 'authorization_code') {
            writeText(response, 400, 'invalid grant_type');
            return;
        }

        const presentedCredentials = getClientCredentials(request, form);
        if (
            presentedCredentials.clientId !== clientId ||
            presentedCredentials.clientSecret !== clientSecret
        ) {
            writeText(response, 401, 'invalid client credentials');
            return;
        }

        const authorizationCode = form.get('code');
        if (!authorizationCode) {
            writeText(response, 400, 'missing authorization code');
            return;
        }

        const authorization = authorizationCodes.get(authorizationCode);
        if (!authorization) {
            writeText(response, 400, 'unknown authorization code');
            return;
        }
        authorizationCodes.delete(authorizationCode);
        if (form.get('redirect_uri') !== authorization.redirectUri) {
            writeText(response, 400, 'invalid redirect_uri');
            return;
        }

        const now = Math.floor(Date.now() / 1000);
        const idToken = signJwt({
            iss: issuerBaseUrl,
            aud: clientId,
            sub: subject,
            email,
            email_verified: true,
            preferred_username: username,
            name: displayName,
            nonce: authorization.nonce,
            iat: now,
            exp: now + 300,
        });

        writeJson(response, 200, {
            access_token: 'mock-access-token',
            token_type: 'Bearer',
            expires_in: 300,
            id_token: idToken,
        });
        return;
    }

    if (requestUrl.pathname === '/jwks') {
        writeJson(response, 200, { keys: [publicJwk] });
        return;
    }

    writeText(response, 404, 'not found');
});

server.listen(port, '127.0.0.1', () => {
    process.stdout.write(
        `mock OIDC server listening on ${issuerBaseUrl}\n`,
    );
});

for (const signal of ['SIGINT', 'SIGTERM']) {
    process.on(signal, () => {
        server.close(() => process.exit(0));
    });
}