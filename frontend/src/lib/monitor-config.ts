export interface MonitorConfigTarget {
    id?: string;
    type: string;
    config: unknown;
    interval_s?: number;
}


export interface MonitorDisplayField {
    label: string;
    value: string;
    href?: string;
    monospace?: boolean;
    multiline?: boolean;
    testId?: string;
}

export interface MonitorDisplaySection {
    title: string;
    rows: MonitorDisplayField[];
}

export interface MonitorDisplayStep {
    id: string;
    title: string;
    summary: string;
    rows: MonitorDisplayField[];
}

export interface MonitorCheckDescription {
    typeLabel: string;
    basicItems: MonitorDisplayField[];
    summaryItems: MonitorDisplayField[];
    runtimeSections: MonitorDisplaySection[];
    detailSections: MonitorDisplaySection[];
    steps: MonitorDisplayStep[];
}

export interface MonitorCheckResult {
    status?: string;
    latency_ms?: number;
    status_code?: number;
    message?: string;
    metadata?: unknown;
    checked_at?: string;
}

const typeLabels: Record<string, string> = {
    http: 'HTTP',
    tcp: 'TCP',
    ping: 'Ping',
    dns: 'DNS',
    ssl: 'SSL',
    ssh: 'SSH',
    json: 'JSON API',
    push: 'Push',
    websocket: 'WebSocket',
    smtp: 'SMTP',
    udp: 'UDP',
    redis: 'Redis',
    postgres: 'PostgreSQL',
    mysql: 'MySQL',
    mongo: 'MongoDB',
    https: 'HTTPS',
    composite: 'Composite',
    transaction: 'Transaction',
    dns_http: 'DNS+HTTP',
};

export function formatMonitorTypeLabel(type: string): string {
    return (
        typeLabels[type] ??
        type
            .replace(/_/g, ' ')
            .replace(/\b\w/g, (char) => char.toUpperCase())
    );
}

export function parseMonitorConfig(config: unknown): Record<string, any> {
    if (typeof config === 'string') {
        try {
            return JSON.parse(config) as Record<string, any>;
        } catch (error) {
            console.warn('Failed to parse monitor config', error);
            return {};
        }
    }

    return typeof config === 'object' && config !== null && !Array.isArray(config)
        ? (config as Record<string, any>)
        : {};
}

export function parseCheckMetadata(metadata: unknown): Record<string, any> {
    return parseMonitorConfig(metadata);
}

function isNonEmptyString(value: unknown): value is string {
    return typeof value === 'string' && value.trim().length > 0;
}

function readString(config: Record<string, any>, key: string): string | undefined {
    const value = config[key];
    return isNonEmptyString(value) ? value.trim() : undefined;
}

function readNumber(config: Record<string, any>, key: string): number | undefined {
    const value = config[key];
    return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

function readBoolean(config: Record<string, any>, key: string): boolean | undefined {
    const value = config[key];
    return typeof value === 'boolean' ? value : undefined;
}

function readStringArray(config: Record<string, any>, key: string): string[] {
    const value = config[key];
    if (!Array.isArray(value)) {
        return [];
    }

    return value
        .filter((entry): entry is string => isNonEmptyString(entry))
        .map((entry) => entry.trim());
}

function readRecord(config: Record<string, any>, key: string): Record<string, any> {
    const value = config[key];
    return typeof value === 'object' && value !== null && !Array.isArray(value)
        ? (value as Record<string, any>)
        : {};
}

function readStringRecord(config: Record<string, any>, key: string): Record<string, string> {
    const value = readRecord(config, key);
    const entries = Object.entries(value)
        .filter(([, entry]) => entry !== undefined && entry !== null && `${entry}`.trim() !== '')
        .map(([entryKey, entryValue]) => [entryKey, `${entryValue}`.trim()]);
    return Object.fromEntries(entries);
}

function humanizeKey(key: string): string {
    return key
        .replace(/_/g, ' ')
        .replace(/\b\w/g, (char) => char.toUpperCase());
}

function addField(
    rows: MonitorDisplayField[],
    label: string,
    value: string | number | undefined,
    options: Partial<MonitorDisplayField> = {},
) {
    if (value === undefined) {
        return;
    }

    const text = typeof value === 'string' ? value.trim() : `${value}`;
    if (text.length === 0) {
        return;
    }

    rows.push({
        label,
        value: text,
        href: options.href,
        monospace: options.monospace,
        multiline: options.multiline,
        testId: options.testId,
    });
}

function addCadence(summaryItems: MonitorDisplayField[], intervalS?: number) {
    if (typeof intervalS !== 'number' || !Number.isFinite(intervalS) || intervalS <= 0) {
        return;
    }

    addField(summaryItems, 'Cadence', `Every ${intervalS}s`);
}

function formatEndpoint(host?: string, port?: number): string | undefined {
    if (!host) {
        return undefined;
    }

    return typeof port === 'number' && Number.isFinite(port) && port > 0
        ? `${host}:${port}`
        : host;
}

function formatHeaders(headers: Record<string, string>): string | undefined {
    const entries = Object.entries(headers);
    if (entries.length === 0) {
        return undefined;
    }

    return entries
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, value]) => `${key}: ${value}`)
        .join('\n');
}

function formatExtractRules(extract: Record<string, string>): string | undefined {
    const entries = Object.entries(extract);
    if (entries.length === 0) {
        return undefined;
    }

    return entries
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, value]) => `${key} <- ${value}`)
        .join('\n');
}

function formatCompositeMode(mode: string, quorum?: number): string {
    if (mode === 'any_up') {
        return 'Any child monitor can be up';
    }

    if (mode === 'quorum') {
        return `At least ${quorum ?? 1} child monitor${(quorum ?? 1) === 1 ? '' : 's'} must be up`;
    }

    return 'All child monitors must be up';
}

function summarizeExpectation(parts: Array<string | undefined>): string | undefined {
    const filtered = parts.filter((part): part is string => isNonEmptyString(part));
    return filtered.length > 0 ? filtered.join(', ') : undefined;
}

function parseConnectionTarget(connectionString: string): string {
    try {
        const parsed = new URL(connectionString);
        const host = parsed.port
            ? `${parsed.hostname}:${parsed.port}`
            : parsed.hostname;
        const path = parsed.pathname.replace(/^\/+/, '');
        return path ? `${host}/${path}` : host;
    } catch {
        return connectionString;
    }
}

function buildGenericRows(config: Record<string, any>): MonitorDisplayField[] {
    return Object.entries(config)
        .sort(([left], [right]) => left.localeCompare(right))
        .flatMap(([key, value]) => {
            if (value === undefined || value === null || value === '') {
                return [];
            }

            if (Array.isArray(value)) {
                if (value.length === 0) {
                    return [];
                }

                return [
                    {
                        label: humanizeKey(key),
                        value: value
                            .map((entry) =>
                                typeof entry === 'object'
                                    ? JSON.stringify(entry, null, 2)
                                    : `${entry}`,
                            )
                            .join('\n'),
                        monospace: true,
                        multiline: true,
                    },
                ];
            }

            if (typeof value === 'object') {
                return [
                    {
                        label: humanizeKey(key),
                        value: JSON.stringify(value, null, 2),
                        monospace: true,
                        multiline: true,
                    },
                ];
            }

            return [
                {
                    label: humanizeKey(key),
                    value: typeof value === 'boolean' ? (value ? 'Yes' : 'No') : `${value}`,
                    monospace: typeof value === 'string' && value.includes('://'),
                },
            ];
        });
}

function formatISODate(value: string | undefined): string | undefined {
    if (!value) {
        return undefined;
    }

    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }

    return parsed.toISOString().slice(0, 10);
}

function formatTimestamp(value: string | undefined): string | undefined {
    if (!value) {
        return undefined;
    }

    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }

    return `${parsed.toISOString().slice(0, 16).replace('T', ' ')} UTC`;
}

function formatDaysRemaining(days: number | undefined): string | undefined {
    if (days === undefined) {
        return undefined;
    }

    if (days < 0) {
        return `Expired ${Math.abs(days)} day${Math.abs(days) === 1 ? '' : 's'} ago`;
    }

    if (days === 0) {
        return 'Less than 1 day';
    }

    return `${days} day${days === 1 ? '' : 's'}`;
}

function summarizeList(values: string[], limit = 2): string | undefined {
    if (values.length === 0) {
        return undefined;
    }

    if (values.length <= limit) {
        return values.join(', ');
    }

    return `${values.slice(0, limit).join(', ')} (+${values.length - limit} more)`;
}

function buildLatestRuntime(
    monitor: MonitorConfigTarget,
    latestCheck?: MonitorCheckResult | null,
): {
    basicItems: MonitorDisplayField[];
    runtimeSections: MonitorDisplaySection[];
} {
    const basicItems: MonitorDisplayField[] = [];
    const runtimeSections: MonitorDisplaySection[] = [];

    if (!latestCheck) {
        return { basicItems, runtimeSections };
    }

    const config = parseMonitorConfig(monitor.config);
    const metadata = parseCheckMetadata(latestCheck.metadata);
    const latestRows: MonitorDisplayField[] = [];

    addField(latestRows, 'Status', latestCheck.status ? humanizeKey(latestCheck.status) : undefined);
    addField(latestRows, 'Status Code', latestCheck.status_code);
    addField(latestRows, 'Message', latestCheck.message, { multiline: true });
    addField(latestRows, 'Checked At', formatTimestamp(latestCheck.checked_at));

    if (
        typeof latestCheck.status_code === 'number' &&
        ['http', 'https', 'json', 'dns_http'].includes(monitor.type)
    ) {
        addField(basicItems, 'Last Status Code', latestCheck.status_code);
    }

    switch (monitor.type) {
        case 'https':
        case 'ssl': {
            const certRows: MonitorDisplayField[] = [];
            const certNotAfter = readString(metadata, 'cert_not_after');
            const certDaysRemaining = readNumber(metadata, 'cert_days_remaining');
            const certWarnDays =
                readNumber(metadata, 'cert_warn_days') ??
                (monitor.type === 'https'
                    ? readNumber(config, 'warn_days') ?? 14
                    : readNumber(config, 'days_before_expiry') ?? 7);

            addField(basicItems, 'Certificate Expires', formatISODate(certNotAfter), {
                testId: 'monitor-basic-certificate-expires',
            });
            addField(basicItems, 'Days Left', formatDaysRemaining(certDaysRemaining), {
                testId: 'monitor-basic-days-left',
            });

            addField(certRows, 'Certificate Expires', formatTimestamp(certNotAfter));
            addField(certRows, 'Days Left', formatDaysRemaining(certDaysRemaining));
            addField(certRows, 'Warning Threshold', formatDaysRemaining(certWarnDays));
            addField(certRows, 'Valid From', formatTimestamp(readString(metadata, 'cert_not_before')));
            addField(certRows, 'Subject', readString(metadata, 'cert_subject'), {
                monospace: true,
            });
            addField(certRows, 'Issuer', readString(metadata, 'cert_issuer'), {
                monospace: true,
            });

            if (certRows.length > 0) {
                runtimeSections.push({ title: 'Latest Certificate', rows: certRows });
            }
            break;
        }

        case 'dns': {
            const answers = readStringArray(metadata, 'answers');
            addField(basicItems, 'Latest Answers', summarizeList(answers));
            break;
        }

        case 'dns_http': {
            const resolvedIPs = readStringArray(metadata, 'resolved_ips');
            addField(basicItems, 'Resolved IPs', summarizeList(resolvedIPs));
            addField(basicItems, 'Hostname', readString(metadata, 'hostname'), {
                monospace: true,
            });
            break;
        }

        case 'composite': {
            const upCount = readNumber(metadata, 'up_count');
            const total = readNumber(metadata, 'total');
            addField(
                basicItems,
                'Members Up',
                upCount !== undefined && total !== undefined ? `${upCount}/${total}` : undefined,
            );
            break;
        }

        case 'transaction': {
            const transactionSteps = Array.isArray(metadata.steps) ? metadata.steps : [];
            addField(basicItems, 'Latest Run Steps', transactionSteps.length);
            break;
        }
    }

    if (Object.keys(metadata).length > 0 && runtimeSections.length === 0) {
        const metadataRows = buildGenericRows(metadata);
        if (metadataRows.length > 0) {
            runtimeSections.push({ title: 'Latest Monitor Data', rows: metadataRows });
        }
    }

    if (latestRows.length > 0) {
        runtimeSections.push({ title: 'Latest Check', rows: latestRows });
    }

    return { basicItems, runtimeSections };
}

export function describeMonitorCheck(
    monitor: MonitorConfigTarget,
    latestCheck?: MonitorCheckResult | null,
): MonitorCheckDescription {
    const config = parseMonitorConfig(monitor.config);
    const typeLabel = formatMonitorTypeLabel(monitor.type);
    const { basicItems, runtimeSections } = buildLatestRuntime(monitor, latestCheck);
    const summaryItems: MonitorDisplayField[] = [];
    const detailSections: MonitorDisplaySection[] = [];
    const steps: MonitorDisplayStep[] = [];
    const rows: MonitorDisplayField[] = [];

    switch (monitor.type) {
        case 'http': {
            const method = readString(config, 'method') ?? 'GET';
            const url = readString(config, 'url');
            const expectedStatus = readNumber(config, 'expected_status') ?? 200;
            const expectedBody = readString(config, 'expected_body');
            const headers = readStringRecord(config, 'headers');
            const body = readString(config, 'body');
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');

            addField(summaryItems, 'Request', url ? `${method} ${url}` : method, {
                href: url,
            });
            addField(
                summaryItems,
                'Expectation',
                summarizeExpectation([
                    `HTTP ${expectedStatus}`,
                    expectedBody ? `body contains "${expectedBody}"` : undefined,
                ]),
            );

            addField(rows, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Method', method);
            addField(rows, 'Expected Status', expectedStatus);
            addField(rows, 'Expected Body', expectedBody, { multiline: true });
            addField(rows, 'Headers', formatHeaders(headers), {
                monospace: true,
                multiline: true,
            });
            addField(rows, 'Request Body', body, { multiline: true, monospace: true });
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : undefined);
            break;
        }

        case 'https': {
            const method = readString(config, 'method') ?? 'GET';
            const url = readString(config, 'url');
            const expectedStatus = readNumber(config, 'expected_status') ?? 200;
            const expectedBody = readString(config, 'expected_body');
            const headers = readStringRecord(config, 'headers');
            const body = readString(config, 'body');
            const warnDays = readNumber(config, 'warn_days') ?? 14;
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');

            addField(summaryItems, 'Request', url ? `${method} ${url}` : method, {
                href: url,
            });
            addField(
                summaryItems,
                'Expectation',
                summarizeExpectation([
                    `HTTP ${expectedStatus}`,
                    `TLS expires in more than ${warnDays} days`,
                    expectedBody ? `body contains "${expectedBody}"` : undefined,
                ]),
            );

            addField(rows, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Method', method);
            addField(rows, 'Expected Status', expectedStatus);
            addField(rows, 'Expected Body', expectedBody, { multiline: true });
            addField(rows, 'TLS Warning Threshold', `${warnDays} days`);
            addField(rows, 'Headers', formatHeaders(headers), {
                monospace: true,
                multiline: true,
            });
            addField(rows, 'Request Body', body, { multiline: true, monospace: true });
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : undefined);
            break;
        }

        case 'tcp': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port');
            const endpoint = formatEndpoint(host, port);
            addField(summaryItems, 'Check', 'Open a TCP connection');
            addField(summaryItems, 'Target', endpoint, { monospace: true });
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            break;
        }

        case 'ping': {
            const host = readString(config, 'host');
            const count = readNumber(config, 'count');
            addField(summaryItems, 'Check', 'Send ICMP pings');
            addField(summaryItems, 'Target', host, { monospace: true });
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Ping Count', count);
            break;
        }

        case 'dns': {
            const host = readString(config, 'host');
            const recordType = readString(config, 'record_type') ?? 'A';
            const resolver = readString(config, 'resolver');
            const expected = readString(config, 'expected');
            addField(summaryItems, 'Check', `Resolve ${recordType} records`);
            addField(summaryItems, 'Target', host, { monospace: true });
            addField(summaryItems, 'Expectation', expected ? `Answer includes ${expected}` : undefined);
            addField(rows, 'Domain', host, { monospace: true });
            addField(rows, 'Record Type', recordType);
            addField(rows, 'Resolver', resolver, { monospace: true });
            addField(rows, 'Expected Answer', expected, { monospace: true });
            break;
        }

        case 'ssl': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port') ?? 443;
            const daysBeforeExpiry = readNumber(config, 'days_before_expiry') ?? 7;
            addField(summaryItems, 'Check', 'Inspect TLS certificate expiry');
            addField(summaryItems, 'Target', formatEndpoint(host, port), {
                monospace: true,
            });
            addField(summaryItems, 'Expectation', `More than ${daysBeforeExpiry} days remaining`);
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            addField(rows, 'Expiry Threshold', `${daysBeforeExpiry} days`);
            break;
        }

        case 'ssh': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port') ?? 22;
            addField(summaryItems, 'Check', 'Open an SSH connection');
            addField(summaryItems, 'Target', formatEndpoint(host, port), {
                monospace: true,
            });
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            break;
        }

        case 'json': {
            const method = readString(config, 'method') ?? 'GET';
            const url = readString(config, 'url');
            const field = readString(config, 'field');
            const expectedValue = readString(config, 'expected_value');
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');
            addField(summaryItems, 'Request', url ? `${method} ${url}` : method, {
                href: url,
            });
            addField(
                summaryItems,
                'Expectation',
                field && expectedValue
                    ? `${field} equals ${expectedValue}`
                    : field
                      ? `Check JSON field ${field}`
                      : undefined,
            );
            addField(rows, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Method', method);
            addField(rows, 'JSON Field', field, { monospace: true });
            addField(rows, 'Expected Value', expectedValue, { monospace: true });
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : undefined);
            break;
        }

        case 'push': {
            const token = readString(config, 'token');
            const endpointPath = monitor.id ? `/api/v1/heartbeat/${monitor.id}` : undefined;
            addField(summaryItems, 'Check', 'Wait for heartbeat pings');
            addField(summaryItems, 'Endpoint', endpointPath, { monospace: true });
            addField(summaryItems, 'Token', token, { monospace: true });
            addField(rows, 'Heartbeat Endpoint', endpointPath, { monospace: true });
            addField(rows, 'Token', token, { monospace: true });
            break;
        }

        case 'websocket': {
            const url = readString(config, 'url');
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');
            addField(summaryItems, 'Check', 'Open a WebSocket connection');
            addField(summaryItems, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : undefined);
            break;
        }

        case 'smtp': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port');
            const requireTLS = readBoolean(config, 'require_tls');
            addField(summaryItems, 'Check', 'Open an SMTP connection');
            addField(summaryItems, 'Target', formatEndpoint(host, port), {
                monospace: true,
            });
            addField(summaryItems, 'TLS', requireTLS ? 'Required' : 'Optional');
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            addField(rows, 'Require TLS', requireTLS === undefined ? undefined : requireTLS ? 'Yes' : 'No');
            break;
        }

        case 'udp': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port');
            const sendPayload = readString(config, 'send_payload');
            const expectedResponse = readString(config, 'expected_response');
            addField(summaryItems, 'Check', 'Send a UDP packet');
            addField(summaryItems, 'Target', formatEndpoint(host, port), {
                monospace: true,
            });
            addField(summaryItems, 'Expectation', expectedResponse ? `Response contains ${expectedResponse}` : undefined);
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            addField(rows, 'Send Payload', sendPayload, { multiline: true, monospace: true });
            addField(rows, 'Expected Response', expectedResponse, {
                multiline: true,
                monospace: true,
            });
            break;
        }

        case 'redis': {
            const host = readString(config, 'host');
            const port = readNumber(config, 'port');
            const database = readNumber(config, 'database') ?? 0;
            const password = readString(config, 'password');
            addField(summaryItems, 'Check', 'Connect to Redis');
            addField(summaryItems, 'Target', formatEndpoint(host, port), {
                monospace: true,
            });
            addField(summaryItems, 'Database', database);
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            addField(rows, 'Database', database);
            addField(rows, 'Password', password, { monospace: true });
            break;
        }

        case 'postgres':
        case 'mysql':
        case 'mongo': {
            const connectionString = readString(config, 'connection_string');
            const host = readString(config, 'host');
            const port = readNumber(config, 'port');
            const database = readString(config, 'database');
            const user = readString(config, 'user');
            const sslMode = readString(config, 'ssl_mode');
            const target = connectionString
                ? parseConnectionTarget(connectionString)
                : formatEndpoint(host, port);
            addField(summaryItems, 'Check', `Connect to ${typeLabel}`);
            addField(summaryItems, 'Target', target, { monospace: true });
            addField(summaryItems, 'Database', database, { monospace: true });
            addField(rows, 'Target', target, { monospace: true });
            addField(rows, 'Connection String', connectionString, {
                monospace: true,
                multiline: true,
            });
            addField(rows, 'Host', host, { monospace: true });
            addField(rows, 'Port', port);
            addField(rows, 'User', user, { monospace: true });
            addField(rows, 'Database', database, { monospace: true });
            addField(rows, 'SSL Mode', sslMode);
            break;
        }

        case 'composite': {
            const monitorIDs = readStringArray(config, 'monitor_ids');
            const mode = readString(config, 'mode') ?? 'all_up';
            const quorum = readNumber(config, 'quorum');
            addField(summaryItems, 'Check', 'Evaluate child monitor states');
            addField(summaryItems, 'Members', `${monitorIDs.length} monitor${monitorIDs.length === 1 ? '' : 's'}`);
            addField(summaryItems, 'Mode', formatCompositeMode(mode, quorum));
            addField(rows, 'Mode', formatCompositeMode(mode, quorum));
            addField(rows, 'Quorum', quorum);
            addField(rows, 'Monitor IDs', monitorIDs.join('\n'), {
                monospace: true,
                multiline: monitorIDs.length > 1,
            });
            break;
        }

        case 'transaction': {
            const transactionSteps = Array.isArray(config.steps) ? config.steps : [];
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');
            const firstStep =
                transactionSteps.length > 0 &&
                typeof transactionSteps[0] === 'object' &&
                transactionSteps[0] !== null
                    ? (transactionSteps[0] as Record<string, any>)
                    : undefined;
            const firstMethod = firstStep && isNonEmptyString(firstStep.method)
                ? firstStep.method.trim()
                : 'GET';
            const firstURL = firstStep && isNonEmptyString(firstStep.url)
                ? firstStep.url.trim()
                : undefined;

            addField(summaryItems, 'Flow', `${transactionSteps.length} step${transactionSteps.length === 1 ? '' : 's'}`);
            addField(summaryItems, 'Starts With', firstURL ? `${firstMethod} ${firstURL}` : undefined, {
                href: firstURL,
            });
            addField(summaryItems, 'TLS', skipTLSVerify ? 'Verification skipped' : 'Verification enforced');

            addField(rows, 'Step Count', transactionSteps.length);
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : 'No');

            steps.push(
                ...transactionSteps.flatMap((step, index) => {
                    if (typeof step !== 'object' || step === null || Array.isArray(step)) {
                        return [];
                    }

                    const record = step as Record<string, any>;
                    const method = isNonEmptyString(record.method)
                        ? record.method.trim()
                        : 'GET';
                    const url = isNonEmptyString(record.url) ? record.url.trim() : undefined;
                    const expectedStatus = typeof record.expected_status === 'number'
                        ? record.expected_status
                        : undefined;
                    const expectedBody = isNonEmptyString(record.expected_body)
                        ? record.expected_body.trim()
                        : undefined;
                    const headers = Object.fromEntries(
                        Object.entries(readRecord(record, 'headers')).map(([key, value]) => [key, `${value}`]),
                    );
                    const extract = Object.fromEntries(
                        Object.entries(readRecord(record, 'extract')).map(([key, value]) => [key, `${value}`]),
                    );
                    const body = isNonEmptyString(record.body) ? record.body.trim() : undefined;
                    const stepRows: MonitorDisplayField[] = [];

                    addField(stepRows, 'Method', method);
                    addField(stepRows, 'Target', url, { href: url, monospace: true });
                    addField(stepRows, 'Expected Status', expectedStatus);
                    addField(stepRows, 'Expected Body', expectedBody, { multiline: true });
                    addField(stepRows, 'Headers', formatHeaders(headers), {
                        monospace: true,
                        multiline: true,
                    });
                    addField(stepRows, 'Request Body', body, {
                        monospace: true,
                        multiline: true,
                    });
                    addField(stepRows, 'Extract', formatExtractRules(extract), {
                        monospace: true,
                        multiline: true,
                    });

                    return [
                        {
                            id: `monitor-detail-transaction-step-${index + 1}`,
                            title: `Step ${index + 1}`,
                            summary: url ? `${method} ${url}` : method,
                            rows: stepRows,
                        },
                    ];
                }),
            );
            break;
        }

        case 'dns_http': {
            const url = readString(config, 'url');
            const expectedStatus = readNumber(config, 'expected_status') ?? 200;
            const expectedIPPrefix = readString(config, 'expected_ip_prefix');
            const expectedCNAME = readString(config, 'expected_cname');
            const expectedBody = readString(config, 'expected_body');
            const skipTLSVerify = readBoolean(config, 'skip_tls_verify');
            addField(summaryItems, 'Request', url ? `Resolve and fetch ${url}` : undefined, {
                href: url,
            });
            addField(
                summaryItems,
                'Expectation',
                summarizeExpectation([
                    `HTTP ${expectedStatus}`,
                    expectedIPPrefix ? `IP starts with ${expectedIPPrefix}` : undefined,
                    expectedCNAME ? `CNAME matches ${expectedCNAME}` : undefined,
                    expectedBody ? `body contains "${expectedBody}"` : undefined,
                ]),
            );
            addField(rows, 'Target', url, { href: url, monospace: true });
            addField(rows, 'Expected Status', expectedStatus);
            addField(rows, 'Expected IP Prefix', expectedIPPrefix, { monospace: true });
            addField(rows, 'Expected CNAME', expectedCNAME, { monospace: true });
            addField(rows, 'Expected Body', expectedBody, { multiline: true });
            addField(rows, 'Skip TLS Verification', skipTLSVerify ? 'Yes' : undefined);
            break;
        }

        default: {
            addField(summaryItems, 'Type', typeLabel);
            break;
        }
    }

    addCadence(summaryItems, monitor.interval_s);

    if (rows.length > 0) {
        detailSections.push({ title: 'Configuration', rows });
    }

    if (detailSections.length === 0) {
        const genericRows = buildGenericRows(config);
        if (genericRows.length > 0) {
            detailSections.push({ title: 'Configuration', rows: genericRows });
        }
    }

    if (summaryItems.length === 0) {
        addField(summaryItems, 'Type', typeLabel);
        addCadence(summaryItems, monitor.interval_s);
    }

    return {
        typeLabel,
        basicItems,
        summaryItems,
        runtimeSections,
        detailSections,
        steps,
    };
}