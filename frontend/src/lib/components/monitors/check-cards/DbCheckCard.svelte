<script lang="ts">
    import { Database } from "lucide-svelte";
    import {
        parseMonitorConfig,
        readNumber,
        readString,
    } from "$lib/monitor-config";
    import CheckCardShell from "./_shared/CheckCardShell.svelte";
    import CopyButton from "./_shared/CopyButton.svelte";
    import DetailSection from "./_shared/DetailSection.svelte";
    import FieldTile from "./_shared/FieldTile.svelte";
    import type { CheckCardProps } from "./_shared/types.ts";

    let { monitor }: CheckCardProps = $props();

    const config = $derived(parseMonitorConfig(monitor.config));

    const typeLabel = $derived.by(() => {
        switch (monitor.type) {
            case "postgres":
                return "PostgreSQL";
            case "mysql":
                return "MySQL";
            case "mongo":
                return "MongoDB";
            case "redis":
                return "Redis";
            default:
                return "Database";
        }
    });

    const defaultPort = $derived.by(() => {
        switch (monitor.type) {
            case "postgres":
                return 5432;
            case "mysql":
                return 3306;
            case "mongo":
                return 27017;
            case "redis":
                return 6379;
            default:
                return undefined;
        }
    });

    const connectionString = $derived(readString(config, "connection_string"));
    const host = $derived(readString(config, "host"));
    const port = $derived(readNumber(config, "port") ?? defaultPort);
    const database = $derived(
        readString(config, "database") ??
            (monitor.type === "redis"
                ? readNumber(config, "database")?.toString()
                : undefined),
    );
    const user = $derived(readString(config, "user"));
    const sslMode = $derived(readString(config, "ssl_mode"));
    const password = $derived(readString(config, "password"));
    const cadence = $derived(monitor.interval_s);

    // Build a synthetic connection string when only discrete fields are set
    // so the user can copy a working URI. We never render the password
    // literal, but it is included in the copyable string.
    const syntheticUri = $derived.by(() => {
        if (!host) return undefined;
        const scheme =
            monitor.type === "postgres"
                ? "postgresql"
                : monitor.type === "mysql"
                  ? "mysql"
                  : monitor.type === "mongo"
                    ? "mongodb"
                    : monitor.type === "redis"
                      ? "redis"
                      : monitor.type;
        const auth = user
            ? password
                ? `${encodeURIComponent(user)}:${encodeURIComponent(password)}@`
                : `${encodeURIComponent(user)}@`
            : "";
        const portPart = port ? `:${port}` : "";
        const dbPart = database ? `/${database}` : "";
        return `${scheme}://${auth}${host}${portPart}${dbPart}`;
    });

    const fullConnectionString = $derived(connectionString ?? syntheticUri);

    // Mask any password segment for display: scheme://user:****@host/db
    const displayConnectionString = $derived.by(() => {
        if (!fullConnectionString) return undefined;
        return fullConnectionString.replace(
            /(\/\/[^:/@]+:)([^@]+)(@)/,
            "$1••••$3",
        );
    });

    const endpoint = $derived(host ? `${host}${port ? `:${port}` : ""}` : undefined);
</script>

<CheckCardShell
    typeLabel={typeLabel}
    description={`updu opens a ${typeLabel} connection and runs a lightweight health probe.`}
    hasDetails
>
    {#snippet basics()}
        <FieldTile label="Endpoint" value={endpoint} monospace />
        <FieldTile label="Database" value={database} monospace />
        <FieldTile label="User" value={user} monospace />
        <FieldTile
            label="Cadence"
            value={cadence ? `Every ${cadence}s` : undefined}
        />
    {/snippet}

    {#snippet hero()}
        <div
            class="rounded-2xl border border-primary/30 bg-primary/5 p-4 sm:p-5 space-y-3"
        >
            <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                    <Database class="size-4 text-primary" />
                    <p
                        class="text-[11px] font-semibold uppercase tracking-[0.18em] text-primary"
                    >
                        {typeLabel} Connection
                    </p>
                </div>
                {#if fullConnectionString}
                    <CopyButton
                        value={fullConnectionString}
                        label="Copy connection string"
                        successMessage="Connection string copied"
                        size="sm"
                        testId="monitor-db-copy-connection"
                    />
                {/if}
            </div>
            {#if displayConnectionString}
                <p
                    data-testid="monitor-db-connection"
                    class="font-mono text-sm sm:text-base break-all text-text"
                >
                    {displayConnectionString}
                </p>
                {#if password}
                    <p class="text-[11px] text-text-subtle">
                        Password is hidden in the UI but included when copied.
                    </p>
                {/if}
            {:else}
                <p class="text-sm text-text-subtle italic">
                    No connection target configured.
                </p>
            {/if}
        </div>
    {/snippet}

    {#snippet details()}
        <DetailSection title="Endpoint">
            <FieldTile
                label="Host"
                value={host}
                monospace
                copyable={Boolean(host)}
            />
            <FieldTile label="Port" value={port} />
            <FieldTile
                label="Database"
                value={database}
                monospace
                copyable={Boolean(database)}
            />
            <FieldTile
                label="User"
                value={user}
                monospace
                copyable={Boolean(user)}
            />
            {#if sslMode}
                <FieldTile label="SSL Mode" value={sslMode} />
            {/if}
            {#if password}
                <FieldTile
                    label="Password"
                    value="•••• (configured)"
                    copyable
                    copyValue={password}
                    copyLabel="Copy password"
                />
            {/if}
        </DetailSection>

        {#if connectionString}
            <DetailSection title="Connection String">
                <FieldTile
                    label="Configured URI"
                    value={displayConnectionString}
                    monospace
                    multiline
                    copyable
                    copyValue={connectionString}
                    copyLabel="Copy connection string"
                />
            </DetailSection>
        {/if}
    {/snippet}
</CheckCardShell>
