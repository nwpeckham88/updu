<script lang="ts">
    import { onMount } from 'svelte';
    import { Bell, Pencil, Plus, Send, Trash2, X } from 'lucide-svelte';
    import { Dialog } from 'bits-ui';
    import Button from '$lib/components/ui/button.svelte';
    import ConfirmActionDialog from '$lib/components/settings/ConfirmActionDialog.svelte';
    import EmptyState from '$lib/components/ui/empty-state.svelte';
    import Skeleton from '$lib/components/ui/skeleton.svelte';
    import {
        createNotificationChannel,
        deleteNotificationChannel,
        listNotificationChannels,
        sendNotificationChannelTest,
        type NotificationChannel,
        type NotificationChannelConfig,
        updateNotificationChannel,
    } from '$lib/api/settings';

    let channels = $state<NotificationChannel[]>([]);
    let channelsLoading = $state(true);
    let channelsMsg = $state('');

    let dialogOpen = $state(false);
    let editTarget = $state<NotificationChannel | null>(null);
    let channelName = $state('');
    let channelType = $state('webhook');
    let channelEnabled = $state(true);
    let channelUrl = $state('');
    let emailHost = $state('');
    let emailPort = $state(587);
    let emailUser = $state('');
    let emailPass = $state('');
    let emailFrom = $state('');
    let emailTo = $state('');
    let saveError = $state('');
    let saving = $state(false);
    let testingId = $state<string | null>(null);
    let deleteDialogOpen = $state(false);
    let deleteTarget = $state<NotificationChannel | null>(null);
    let actionChannelID = $state<string | null>(null);

    function scheduleChannelsMessageClear() {
        setTimeout(() => (channelsMsg = ''), 3000);
    }

    function activeChannelCount(): number {
        return channels.filter((channel) => channel.enabled).length;
    }

    function disabledChannelCount(): number {
        return channels.filter((channel) => !channel.enabled).length;
    }

    function channelDestination(channel: NotificationChannel): string {
        if (channel.type === 'email') {
            if (typeof channel.config?.to === 'string' && channel.config.to) {
                return channel.config.to;
            }

            if (typeof channel.config?.host === 'string' && channel.config.host) {
                return channel.config.host;
            }

            return 'Email delivery target';
        }

        if (typeof channel.config?.url === 'string' && channel.config.url) {
            try {
                return new URL(channel.config.url).host;
            } catch {
                return channel.config.url;
            }
        }

        return `${channel.type} endpoint`;
    }

    function openDeleteDialog(channel: NotificationChannel) {
        deleteTarget = channel;
        deleteDialogOpen = true;
        channelsMsg = '';
    }

    async function loadChannels() {
        try {
            channelsLoading = true;
            channels = (await listNotificationChannels()) || [];
        } catch {
            channels = [];
        } finally {
            channelsLoading = false;
        }
    }

    function openCreateChannel() {
        editTarget = null;
        channelName = '';
        channelType = 'webhook';
        channelEnabled = true;
        channelUrl = '';
        emailHost = '';
        emailPort = 587;
        emailUser = '';
        emailPass = '';
        emailFrom = '';
        emailTo = '';
        saveError = '';
        dialogOpen = true;
    }

    function openEditChannel(channel: NotificationChannel) {
        editTarget = channel;
        channelName = channel.name;
        channelType = channel.type;
        channelEnabled = channel.enabled;
        channelUrl = typeof channel.config?.url === 'string' ? channel.config.url : '';
        emailHost = typeof channel.config?.host === 'string' ? channel.config.host : '';
        emailPort =
            typeof channel.config?.port === 'number' ? channel.config.port : 587;
        emailUser = typeof channel.config?.user === 'string' ? channel.config.user : '';
        emailPass = typeof channel.config?.pass === 'string' ? channel.config.pass : '';
        emailFrom = typeof channel.config?.from === 'string' ? channel.config.from : '';
        emailTo = typeof channel.config?.to === 'string' ? channel.config.to : '';
        saveError = '';
        dialogOpen = true;
    }

    function buildChannelConfig(): NotificationChannelConfig {
        if (channelType === 'email') {
            return {
                host: emailHost,
                port: emailPort,
                user: emailUser,
                pass: emailPass,
                from: emailFrom,
                to: emailTo,
            };
        }

        return { url: channelUrl };
    }

    async function saveChannel() {
        if (!channelName.trim()) {
            saveError = 'Name is required';
            return;
        }

        if (channelType === 'email') {
            if (!emailHost.trim()) {
                saveError = 'SMTP host is required';
                return;
            }

            if (!emailFrom.trim()) {
                saveError = 'From address is required';
                return;
            }

            if (!emailTo.trim()) {
                saveError = 'To address is required';
                return;
            }
        } else {
            if (!channelUrl.trim()) {
                saveError = 'URL is required';
                return;
            }

            try {
                new URL(channelUrl);
            } catch {
                saveError = 'URL is not valid';
                return;
            }
        }

        saving = true;
        saveError = '';

        try {
            const payload = {
                name: channelName.trim(),
                type: channelType,
                enabled: channelEnabled,
                config: buildChannelConfig(),
            };

            if (editTarget) {
                await updateNotificationChannel(editTarget.id, payload);
                channelsMsg = 'Channel updated successfully.';
            } else {
                await createNotificationChannel(payload);
                channelsMsg = 'Channel created successfully.';
            }

            dialogOpen = false;
            await loadChannels();
            scheduleChannelsMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Failed to save channel';
            saveError = message;
        } finally {
            saving = false;
        }
    }

    async function deleteChannel() {
        if (!deleteTarget) {
            return;
        }

        const channel = deleteTarget;
        deleteDialogOpen = false;
        actionChannelID = channel.id;

        try {
            await deleteNotificationChannel(channel.id);
            channelsMsg = 'Channel deleted successfully.';
            deleteTarget = null;
            await loadChannels();
            scheduleChannelsMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Unknown error';
            channelsMsg = `Error: ${message}`;
        } finally {
            actionChannelID = null;
        }
    }

    async function testChannel(id: string) {
        testingId = id;

        try {
            await sendNotificationChannelTest(id);
            channelsMsg = 'Test notification queued.';
            scheduleChannelsMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Unknown error';
            channelsMsg = `Error: ${message}`;
        } finally {
            setTimeout(() => (testingId = null), 1500);
        }
    }

    onMount(() => {
        void loadChannels();
    });
</script>

<div class="settings-stack">
    {#if channelsMsg}
        <div
            class={[
                'settings-banner',
                channelsMsg.startsWith('Error')
                    ? 'settings-banner-danger'
                    : 'settings-banner-success',
            ]}
            aria-live="polite"
        >
            {channelsMsg}
        </div>
    {/if}

    <section class="card settings-section">
        <div class="settings-section-header-split">
            <div class="settings-section-header">
                <div class="settings-section-icon">
                    <Bell class="size-4 text-primary" />
                </div>
                <div>
                    <h2 class="text-base font-semibold text-text">Notification Channels</h2>
                    <p class="text-[11px] text-text-subtle mt-0.5 max-w-2xl">
                        Keep routing explicit and test each destination before you rely on it for incidents or heartbeat failures.
                    </p>
                    {#if !channelsLoading}
                        <div class="settings-meta-row">
                            <span class="settings-pill settings-pill-primary">
                                {channels.length} configured
                            </span>
                            <span class="settings-pill settings-pill-muted">
                                {activeChannelCount()} enabled
                            </span>
                            {#if disabledChannelCount() > 0}
                                <span class="settings-pill settings-pill-muted">
                                    {disabledChannelCount()} disabled
                                </span>
                            {/if}
                        </div>
                    {/if}
                </div>
            </div>

            <Button class="settings-header-action" onclick={openCreateChannel}>
                <Plus class="size-4" />
                New Channel
            </Button>
        </div>

        {#if channelsLoading}
            <div class="space-y-3">
                {#each Array.from({ length: 3 }) as _, index (index)}
                    <div class="settings-skeleton-item flex gap-4">
                        <Skeleton height="h-9" width="w-9" rounded="rounded-xl" />
                        <div class="flex-1 space-y-2">
                            <Skeleton height="h-4" width="w-1/3" />
                            <Skeleton height="h-3" width="w-1/4" />
                        </div>
                    </div>
                {/each}
            </div>
        {:else if channels.length === 0}
            <EmptyState
                icon={Bell}
                title="No notification channels"
                description="Add a webhook, chat integration, or email target to start receiving alerts."
            >
                <Button onclick={openCreateChannel} variant="outline" size="sm">
                    Add Channel
                </Button>
            </EmptyState>
        {:else}
            <div class="space-y-3">
                {#each channels as channel (channel.id)}
                    <article
                        data-testid="notification-channel-row"
                        class="settings-list-item"
                    >
                        <div class="flex items-start gap-4 min-w-0">
                            <div
                                class={`size-9 rounded-xl flex items-center justify-center shrink-0 ${channel.enabled ? 'bg-primary/10 text-primary' : 'bg-surface text-text-subtle'}`}
                            >
                                <Bell class="size-4" />
                            </div>
                            <div class="min-w-0">
                                <div class="flex flex-wrap items-center gap-2">
                                    <h2 class="font-semibold text-text text-sm">{channel.name}</h2>
                                    <span class="settings-pill-label settings-pill-label-muted">
                                        {channel.type}
                                    </span>
                                    {#if !channel.enabled}
                                        <span class="settings-pill-label settings-pill-label-warning">
                                            Disabled
                                        </span>
                                    {/if}
                                </div>
                                <p class="text-[11px] text-text-subtle mt-1 break-all">
                                    {channelDestination(channel)}
                                </p>
                            </div>
                        </div>

                        <div class="flex flex-wrap items-center gap-2 xl:justify-end">
                            <Button
                                size="sm"
                                variant="outline"
                                onclick={() => testChannel(channel.id)}
                                loading={testingId === channel.id}
                            >
                                <Send class="size-3.5" />
                                Send Test
                            </Button>
                            <Button size="sm" variant="ghost" onclick={() => openEditChannel(channel)}>
                                <Pencil class="size-3.5" />
                                Edit
                            </Button>
                            <Button
                                size="sm"
                                variant="ghost"
                                class="text-danger hover:bg-danger/10 hover:text-danger"
                                onclick={() => openDeleteDialog(channel)}
                                loading={actionChannelID === channel.id}
                            >
                                <Trash2 class="size-3.5" />
                                Delete
                            </Button>
                        </div>
                    </article>
                {/each}
            </div>
        {/if}
    </section>
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
                        {editTarget ? 'Edit Channel' : 'New Channel'}
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        {editTarget
                            ? 'Update this notification channel.'
                            : 'Add a new notification channel.'}
                    </Dialog.Description>
                </div>
                <Dialog.Close class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors">
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if saveError}
                <div class="settings-banner settings-banner-danger mb-4">
                    {saveError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="channel-name">
                        Name <span class="text-danger">*</span>
                    </label>
                    <input
                        id="channel-name"
                        type="text"
                        bind:value={channelName}
                        placeholder="Production Alerts"
                        class="input-base"
                    />
                </div>

                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="channel-type">
                        Type
                    </label>
                    <select id="channel-type" bind:value={channelType} class="input-base text-sm">
                        <option value="webhook">Webhook</option>
                        <option value="discord">Discord</option>
                        <option value="slack">Slack</option>
                        <option value="email">Email</option>
                        <option value="ntfy">ntfy</option>
                    </select>
                </div>

                {#if channelType === 'email'}
                    <fieldset class="space-y-3 rounded-xl border border-border/60 p-4">
                        <legend class="px-1 text-sm font-medium text-text">Email configuration</legend>
                        <p class="text-[11px] text-text-subtle">
                            Configure an SMTP server or mail relay to deliver alert messages.
                        </p>

                        <div class="grid gap-3 sm:grid-cols-2">
                            <div class="space-y-1.5">
                                <label class="text-sm font-medium text-text-muted" for="email-host">
                                    SMTP Host
                                </label>
                                <input id="email-host" type="text" bind:value={emailHost} class="input-base" />
                            </div>
                            <div class="space-y-1.5">
                                <label class="text-sm font-medium text-text-muted" for="email-port">
                                    Port
                                </label>
                                <input id="email-port" type="number" bind:value={emailPort} class="input-base" />
                            </div>
                        </div>

                        <div class="grid gap-3 sm:grid-cols-2">
                            <div class="space-y-1.5">
                                <label class="text-sm font-medium text-text-muted" for="email-user">
                                    Username
                                </label>
                                <input id="email-user" type="text" bind:value={emailUser} class="input-base" />
                            </div>
                            <div class="space-y-1.5">
                                <label class="text-sm font-medium text-text-muted" for="email-pass">
                                    Password
                                </label>
                                <input id="email-pass" type="password" bind:value={emailPass} class="input-base" />
                            </div>
                        </div>

                        <div class="space-y-1.5">
                            <label class="text-sm font-medium text-text-muted" for="email-from">
                                From Address <span class="text-danger">*</span>
                            </label>
                            <input id="email-from" type="email" bind:value={emailFrom} class="input-base" />
                        </div>

                        <div class="space-y-1.5">
                            <label class="text-sm font-medium text-text-muted" for="email-to">
                                To Address(es) <span class="text-danger">*</span>
                            </label>
                            <input
                                id="email-to"
                                type="text"
                                bind:value={emailTo}
                                class="input-base"
                                placeholder="ops@example.com, alerts@example.com"
                            />
                        </div>
                    </fieldset>
                {:else}
                    <div class="space-y-1.5">
                        <label class="text-sm font-medium text-text-muted" for="channel-url">
                            URL
                        </label>
                        <input
                            id="channel-url"
                            type="url"
                            bind:value={channelUrl}
                            placeholder="https://..."
                            class="input-base"
                        />
                        <p class="text-[11px] text-text-subtle mt-1">
                            Provide the destination endpoint or webhook URL for this alert integration.
                        </p>
                    </div>
                {/if}

                <label class="flex items-center gap-3 cursor-pointer select-none">
                    <div class="relative">
                        <input type="checkbox" bind:checked={channelEnabled} class="sr-only peer" />
                        <div class="w-9 h-5 rounded-full border border-border bg-surface-elevated peer-checked:bg-primary peer-checked:border-primary transition-colors"></div>
                        <div class="absolute top-0.5 left-0.5 size-4 rounded-full bg-white shadow transition-transform peer-checked:translate-x-4"></div>
                    </div>
                    <div>
                        <p class="text-sm font-medium text-text">Enabled</p>
                        <p class="text-[11px] text-text-subtle">
                            Send notifications through this channel.
                        </p>
                    </div>
                </label>
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button variant="outline" onclick={() => (dialogOpen = false)}>Cancel</Button>
                <Button loading={saving} onclick={saveChannel}>
                    {saving ? 'Saving...' : editTarget ? 'Save Changes' : 'Create Channel'}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>

<ConfirmActionDialog
    bind:open={deleteDialogOpen}
    title="Delete Notification Channel"
    description="Alerts will stop flowing to this destination immediately after removal."
    confirmLabel="Delete Channel"
    loading={Boolean(deleteTarget && actionChannelID === deleteTarget.id)}
    onConfirm={deleteChannel}
>
    {#if deleteTarget}
        <div class="space-y-3 text-sm">
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Channel
                </p>
                <p class="mt-1 text-text">{deleteTarget.name}</p>
            </div>
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Destination
                </p>
                <p class="mt-1 text-text break-all">{channelDestination(deleteTarget)}</p>
            </div>
        </div>
    {/if}
</ConfirmActionDialog>