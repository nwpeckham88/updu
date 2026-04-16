<script lang="ts">
    import { onMount } from "svelte";
    import { formatDistanceToNow } from "date-fns";
    import { Dialog } from "bits-ui";
    import { fetchAPI } from "$lib/api/client";
    import Badge from "$lib/components/ui/badge.svelte";
    import Button from "$lib/components/ui/button.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import { KeyRound, Plus, Shield, Trash2, X } from "lucide-svelte";

    type APIToken = {
        id: string;
        name: string;
        prefix: string;
        scope: "read" | "write";
        created_at: string;
        last_used_at?: string | null;
        revoked_at?: string | null;
    };

    type CreatedAPIToken = APIToken & {
        token: string;
    };

    interface Props {
        onAuditRefresh?: (() => void) | undefined;
    }

    let { onAuditRefresh }: Props = $props();

    let tokens = $state<APIToken[]>([]);
    let loading = $state(true);
    let error = $state("");
    let dialogOpen = $state(false);
    let tokenName = $state("");
    let tokenScope = $state<"read" | "write">("read");
    let saving = $state(false);
    let saveError = $state("");
    let latestCreatedToken = $state<CreatedAPIToken | null>(null);
    let actionTokenID = $state<string | null>(null);

    function formatTimestamp(value?: string | null): string {
        if (!value) {
            return "Never";
        }

        return formatDistanceToNow(new Date(value), { addSuffix: true });
    }

    async function loadTokens() {
        loading = true;
        error = "";

        try {
            tokens = (await fetchAPI("/api/v1/admin/api-tokens")) || [];
        } catch (e: any) {
            error = e.message || "Failed to load API tokens";
            tokens = [];
        } finally {
            loading = false;
        }
    }

    function openCreateDialog() {
        tokenName = "";
        tokenScope = "read";
        saveError = "";
        dialogOpen = true;
    }

    async function createToken() {
        if (!tokenName.trim()) {
            saveError = "Token name is required";
            return;
        }

        saving = true;
        saveError = "";

        try {
            latestCreatedToken = await fetchAPI("/api/v1/admin/api-tokens", {
                method: "POST",
                body: JSON.stringify({
                    name: tokenName.trim(),
                    scope: tokenScope,
                }),
            });
            dialogOpen = false;
            await loadTokens();
            onAuditRefresh?.();
        } catch (e: any) {
            saveError = e.message || "Failed to create API token";
        } finally {
            saving = false;
        }
    }

    async function revokeToken(id: string) {
        if (!confirm("Revoke this API token? Existing clients will stop working immediately.")) {
            return;
        }

        actionTokenID = id;
        try {
            await fetchAPI(`/api/v1/admin/api-tokens/${id}`, {
                method: "DELETE",
            });
            await loadTokens();
            onAuditRefresh?.();
        } catch (e: any) {
            error = e.message || "Failed to revoke API token";
        } finally {
            actionTokenID = null;
        }
    }

    async function copyLatestToken() {
        if (!latestCreatedToken) {
            return;
        }

        try {
            await navigator.clipboard.writeText(latestCreatedToken.token);
        } catch {
            error = "Copy failed. Select the token value manually.";
        }
    }

    function clearLatestToken() {
        latestCreatedToken = null;
    }

    onMount(() => {
        void loadTokens();
    });
</script>

<div class="card space-y-5">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <KeyRound class="size-4 text-primary" />
            </div>
            <div>
                <h3 class="text-sm font-semibold text-text">API Tokens</h3>
                <p class="text-[11px] text-text-subtle mt-0.5 max-w-2xl">
                    Create session-independent credentials for automation and integrations. Tokens only reveal their full secret once.
                </p>
            </div>
        </div>

        <Button
            size="sm"
            variant="outline"
            onclick={openCreateDialog}
            data-testid="create-api-token"
        >
            <Plus class="size-4" />
            Create Token
        </Button>
    </div>

    {#if latestCreatedToken}
        <div
            class="rounded-2xl border border-primary/20 bg-primary/8 p-4 space-y-3"
            data-testid="api-token-secret"
        >
            <div class="flex items-start justify-between gap-3">
                <div>
                    <p class="text-sm font-semibold text-text">Store this token now</p>
                    <p class="text-xs text-text-muted mt-1">
                        This is the only time the full token value is shown for <span class="font-medium text-text">{latestCreatedToken.name}</span>.
                    </p>
                </div>
                <div class="flex items-center gap-2">
                    <Badge status={latestCreatedToken.scope} dot={false}>
                        {latestCreatedToken.scope}
                    </Badge>
                    <button
                        type="button"
                        class="size-8 rounded-lg border border-border/60 text-text-muted hover:text-text hover:bg-surface-elevated transition-colors"
                        onclick={clearLatestToken}
                        aria-label="Hide token"
                    >
                        <X class="size-4 mx-auto" />
                    </button>
                </div>
            </div>
            <div class="rounded-xl border border-border/70 bg-background/80 px-3 py-2 font-mono text-xs text-text break-all">
                {latestCreatedToken.token}
            </div>
            <div class="flex justify-end">
                <Button size="sm" variant="outline" onclick={copyLatestToken}>
                    Copy Token
                </Button>
            </div>
        </div>
    {/if}

    {#if error}
        <div class="p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm">
            {error}
        </div>
    {/if}

    <div class="space-y-3" data-testid="api-token-list">
        {#if loading}
            <div class="space-y-3">
                {#each { length: 3 } as _}
                    <div class="rounded-2xl border border-border/60 p-4 space-y-3">
                        <Skeleton height="h-4" width="w-1/3" />
                        <Skeleton height="h-3" width="w-1/2" />
                        <Skeleton height="h-8" width="w-24" />
                    </div>
                {/each}
            </div>
        {:else if tokens.length === 0}
            <EmptyState
                icon={Shield}
                title="No API tokens yet"
                description="Create a read or write token for CI jobs, scripts, or external automation."
            />
        {:else}
            {#each tokens as token (token.id)}
                <article
                    class="rounded-2xl border border-border/60 bg-surface/20 p-4 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
                    data-testid="api-token-row"
                >
                    <div class="space-y-2 min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                            <h4 class="text-sm font-semibold text-text">{token.name}</h4>
                            <Badge status={token.scope} dot={false}>{token.scope}</Badge>
                            {#if token.revoked_at}
                                <span class="inline-flex items-center rounded-full border border-danger/20 bg-danger/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-danger">
                                    revoked
                                </span>
                            {/if}
                        </div>

                        <div class="grid gap-2 text-xs text-text-muted sm:grid-cols-3">
                            <div>
                                <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Prefix</span>
                                <span class="font-mono text-text">{token.prefix}</span>
                            </div>
                            <div>
                                <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Created</span>
                                <span class="text-text">{formatTimestamp(token.created_at)}</span>
                            </div>
                            <div>
                                <span class="block text-[10px] uppercase tracking-wider text-text-subtle font-semibold">Last used</span>
                                <span class="text-text">{formatTimestamp(token.last_used_at)}</span>
                            </div>
                        </div>
                    </div>

                    <div class="flex items-center gap-2 shrink-0">
                        <Button
                            size="sm"
                            variant="outline"
                            onclick={() => revokeToken(token.id)}
                            disabled={Boolean(token.revoked_at)}
                            loading={actionTokenID === token.id}
                            aria-label="Revoke token"
                        >
                            <Trash2 class="size-4" />
                            {token.revoked_at ? "Revoked" : "Revoke"}
                        </Button>
                    </div>
                </article>
            {/each}
        {/if}
    </div>
</div>

<Dialog.Root bind:open={dialogOpen}>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-center justify-between mb-5">
                <div>
                    <Dialog.Title class="text-base font-semibold text-text">
                        Create API Token
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        Pick the smallest scope that fits the integration.
                    </Dialog.Description>
                </div>
                <Dialog.Close
                    class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                >
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if saveError}
                <div class="mb-4 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm">
                    {saveError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label for="api-token-name" class="text-sm font-medium text-text-muted">
                        Token Name
                    </label>
                    <input
                        id="api-token-name"
                        bind:value={tokenName}
                        class="input-base w-full"
                        placeholder="GitHub Actions"
                    />
                </div>

                <div class="space-y-1.5">
                    <label for="api-token-scope" class="text-sm font-medium text-text-muted">
                        Access Scope
                    </label>
                    <select
                        id="api-token-scope"
                        bind:value={tokenScope}
                        class="input-base w-full"
                    >
                        <option value="read">Read access</option>
                        <option value="write">Write access</option>
                    </select>
                </div>
            </div>

            <div class="mt-6 flex justify-end gap-2">
                <Button variant="ghost" onclick={() => (dialogOpen = false)}>
                    Cancel
                </Button>
                <Button loading={saving} onclick={createToken}>
                    {saving ? "Creating..." : "Create Token"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>