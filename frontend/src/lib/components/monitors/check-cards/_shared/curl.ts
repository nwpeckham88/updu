/**
 * Build a curl(1) command string for an HTTP request.
 * Quotes appropriately for shell paste-ability.
 */
export function buildCurlCommand(options: {
    method?: string;
    url?: string;
    headers?: Record<string, string>;
    body?: string;
    insecure?: boolean;
}): string {
    if (!options.url) return "";

    const parts: string[] = ["curl", "-i"];
    if (options.insecure) parts.push("-k");
    const method = (options.method ?? "GET").toUpperCase();
    if (method !== "GET") parts.push("-X", method);

    if (options.headers) {
        for (const [key, value] of Object.entries(options.headers)) {
            if (!value) continue;
            parts.push("-H", `'${key}: ${escapeSingle(value)}'`);
        }
    }

    if (options.body) {
        parts.push("--data", `'${escapeSingle(options.body)}'`);
    }

    parts.push(`'${escapeSingle(options.url)}'`);
    return parts.join(" ");
}

function escapeSingle(value: string): string {
    return value.replace(/'/g, "'\\''");
}
