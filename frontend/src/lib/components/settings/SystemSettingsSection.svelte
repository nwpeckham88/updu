<script lang="ts">
    import { onMount } from 'svelte';
    import { Shield } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import Skeleton from '$lib/components/ui/skeleton.svelte';
    import ConfirmActionDialog from '$lib/components/settings/ConfirmActionDialog.svelte';
    import {
        applySystemUpdate,
        checkForUpdates,
        getSettings,
        type SettingsMap,
        type UpdateInfo,
        updateSettings,
    } from '$lib/api/settings';

    type VersionContextNote = {
        title: string;
        body: string;
        tone: 'info' | 'warning';
    };

    const apiTokenManagerPromise = import(
        '$lib/components/settings/APITokenManager.svelte'
    );
    const auditLogBrowserPromise = import(
        '$lib/components/settings/AuditLogBrowser.svelte'
    );

    let settings = $state<SettingsMap>({});
    let settingsLoading = $state(true);
    let updateInfo = $state<UpdateInfo | null>(null);
    let updateLoading = $state(false);
    let updateApplying = $state(false);
    let updateMsg = $state('');
    let updateConfirmOpen = $state(false);
    let updateChannelSaving = $state(false);
    let updateChannelMsg = $state('');
    let auditRefreshVersion = $state(0);
    let contextNote = $derived(versionContextNote());
    let releaseNotesItems = $derived(
        releaseNotesPreview(updateInfo?.release_notes),
    );

    function refreshAuditLogs() {
        auditRefreshVersion += 1;
    }

    async function loadSettings() {
        try {
            settingsLoading = true;
            settings = (await getSettings()) || {};
        } catch {
            settings = {};
        } finally {
            settingsLoading = false;
        }
    }

    async function checkUpdate() {
        updateLoading = true;
        updateMsg = '';

        try {
            updateInfo = await checkForUpdates();
        } catch (error) {
            const message =
                error instanceof Error
                    ? error.message
                    : 'Failed to check for updates';
            updateMsg = `Error: ${message}`;
        } finally {
            updateLoading = false;
        }
    }

    function inferUpdateChannelFromVersion(currentVersion?: string): string {
        if (!currentVersion) {
            return 'stable';
        }

        return currentVersion === 'dev' ||
            currentVersion === 'unknown' ||
            currentVersion.includes('-')
            ? 'prerelease'
            : 'stable';
    }

    function isPrereleaseVersion(value?: string): boolean {
        if (!value) {
            return false;
        }

        return value === 'dev' || value === 'unknown' || value.includes('-');
    }

    function selectedUpdateChannel(): string {
        return (
            settings['update_channel'] ||
            inferUpdateChannelFromVersion(updateInfo?.current_version)
        );
    }

    function effectiveReleaseChannel(): 'stable' | 'prerelease' {
        return selectedUpdateChannel() === 'prerelease' ? 'prerelease' : 'stable';
    }

    function latestVersionHeading(): string {
        return effectiveReleaseChannel() === 'stable'
            ? 'Latest Stable Release'
            : 'Latest Prerelease';
    }

    function latestVersionDescription(): string {
        return effectiveReleaseChannel() === 'stable'
            ? 'Compared against stable releases only.'
            : 'Compared against prerelease and stable releases.';
    }

    function currentVersionDescription(): string {
        return isPrereleaseVersion(updateInfo?.current_version)
            ? 'Installed prerelease build.'
            : 'Installed stable build.';
    }

    function formatPublishedAt(value?: string): string {
        if (!value) {
            return 'Unknown';
        }

        const publishedAt = new Date(value);

        if (Number.isNaN(publishedAt.getTime())) {
            return 'Unknown';
        }

        return new Intl.DateTimeFormat('en', {
            dateStyle: 'medium',
            timeStyle: 'short',
        }).format(publishedAt);
    }

    function releaseNotesPreview(notes?: string): string[] {
        if (!notes) {
            return [];
        }

        return notes
            .split('\n')
            .map((line) => line.trim())
            .filter(Boolean)
            .filter((line) => !line.startsWith('#'))
            .map((line) => line.replace(/^[-*]\s*/, ''))
            .slice(0, 3);
    }

    function versionContextNote(): VersionContextNote | null {
        const currentVersion = updateInfo?.current_version;
        const latestVersion = updateInfo?.latest_version;
        const channel = effectiveReleaseChannel();

        if (!currentVersion || !latestVersion) {
            return null;
        }

        if (channel === 'stable' && isPrereleaseVersion(currentVersion)) {
            return {
                tone: 'info',
                title: 'Installed build is ahead of the stable track',
                body: 'You are running a prerelease build, so the latest stable release can appear older than the version already installed here. Switch the release channel to prerelease to compare against beta and RC releases.',
            };
        }

        if (
            channel === 'prerelease' &&
            !isPrereleaseVersion(currentVersion) &&
            isPrereleaseVersion(latestVersion)
        ) {
            return {
                tone: 'warning',
                title: 'This page is comparing stable and prerelease builds',
                body: 'The selected release channel includes prereleases, so the latest available build may be a beta or release candidate even though the installed version is stable.',
            };
        }

        return null;
    }

    async function saveUpdateChannel() {
        updateChannelSaving = true;
        updateChannelMsg = '';

        const channel = selectedUpdateChannel();

        try {
            await updateSettings({ update_channel: channel });
            settings = {
                ...settings,
                update_channel: channel,
            };
            updateChannelMsg =
                channel === 'prerelease'
                    ? 'Release channel saved: prereleases enabled.'
                    : 'Release channel saved: stable releases only.';
            await checkUpdate();
        } catch (error) {
            const message =
                error instanceof Error
                    ? error.message
                    : 'Failed to save update channel';
            updateChannelMsg = `Error: ${message}`;
        } finally {
            updateChannelSaving = false;
        }
    }

    async function applyUpdate() {
        updateConfirmOpen = false;
        updateApplying = true;
        updateMsg = '';

        try {
            const response = await applySystemUpdate();
            updateMsg =
                response.message ||
                'Update applied successfully. System is restarting...';
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Update failed';
            updateMsg = `Error: ${message}`;
        } finally {
            updateApplying = false;
        }
    }

    onMount(() => {
        void loadSettings();
        void checkUpdate();
    });
</script>

<div class="space-y-6">
    <section class="card space-y-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
                <h2 class="text-base font-semibold text-text">System Update</h2>
                <p class="text-xs text-text-muted mt-1 max-w-2xl">
                    Check the installed build, compare it to the selected release track, and decide whether this instance should follow stable releases or prereleases.
                </p>

                <div class="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
                    <span class="inline-flex items-center rounded-full border border-primary/20 bg-primary/8 px-2.5 py-1 font-semibold text-primary">
                        {effectiveReleaseChannel() === 'stable'
                            ? 'Stable track'
                            : 'Prerelease track'}
                    </span>
                    {#if updateInfo?.published_at}
                        <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2.5 py-1 text-text-muted">
                            Release published {formatPublishedAt(updateInfo.published_at)}
                        </span>
                    {/if}
                </div>
            </div>
        </div>

        <div class="rounded-xl border border-border/60 bg-surface-elevated/40 p-4 space-y-3">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
                <div class="space-y-1">
                    <label for="update-channel" class="text-sm font-medium text-text">
                        Release Channel
                    </label>
                    <p class="text-xs text-text-muted max-w-xl">
                        Stable ignores beta and release-candidate builds. Prerelease follows beta and RC builds for your current platform.
                    </p>
                </div>

                <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
                    <select
                        id="update-channel"
                        class="input-base min-w-48 text-sm"
                        disabled={settingsLoading}
                        value={selectedUpdateChannel()}
                        onchange={(event) => {
                            settings = {
                                ...settings,
                                update_channel: (
                                    event.currentTarget as HTMLSelectElement
                                ).value,
                            };
                            updateChannelMsg = '';
                        }}
                    >
                        <option value="stable">Stable only</option>
                        <option value="prerelease">Include prereleases</option>
                    </select>

                    <Button variant="outline" loading={updateChannelSaving} onclick={saveUpdateChannel}>
                        {updateChannelSaving ? 'Saving...' : 'Save & Recheck'}
                    </Button>
                </div>
            </div>

            {#if updateChannelMsg}
                <div
                    class={`p-3 rounded-lg text-sm border ${updateChannelMsg.startsWith('Error') ? 'bg-danger/10 border-danger/20 text-danger' : 'bg-success/10 border-success/20 text-success'}`}
                    aria-live="polite"
                >
                    {updateChannelMsg}
                </div>
            {/if}
        </div>

        {#if updateLoading}
            <div class="flex items-center gap-2 text-sm text-text-muted">
                <svg class="size-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Checking for updates...
            </div>
        {:else if updateInfo}
            <div class="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(18rem,0.85fr)]">
                <div class="space-y-4">
                    <div class="grid gap-3 sm:grid-cols-2">
                        <article class="rounded-2xl border border-border/60 bg-background/60 p-4 space-y-2">
                            <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                Installed Build
                            </p>
                            <p class="text-lg font-semibold font-mono text-text break-all">
                                {updateInfo.current_version || 'unknown'}
                            </p>
                            <p class="text-xs text-text-muted">
                                {currentVersionDescription()}
                            </p>
                        </article>

                        <article class="rounded-2xl border border-border/60 bg-background/60 p-4 space-y-2">
                            <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                {latestVersionHeading()}
                            </p>
                            <p class="text-lg font-semibold font-mono text-text break-all">
                                {updateInfo.latest_version || 'unknown'}
                            </p>
                            <p class="text-xs text-text-muted">
                                {latestVersionDescription()}
                            </p>
                        </article>
                    </div>

                    {#if contextNote}
                        <div
                            class={`rounded-2xl border p-4 ${contextNote.tone === 'warning' ? 'border-warning/30 bg-warning/10' : 'border-primary/20 bg-primary/8'}`}
                        >
                            <h3 class="text-sm font-semibold text-text">{contextNote.title}</h3>
                            <p class="text-xs text-text-muted mt-1 max-w-2xl">
                                {contextNote.body}
                            </p>
                        </div>
                    {/if}

                    {#if updateInfo.update_available}
                        <div class="p-4 rounded-2xl bg-primary/10 border border-primary/20">
                            <div class="flex items-start gap-3">
                                <div class="size-8 rounded-lg bg-primary/20 flex items-center justify-center shrink-0">
                                    <Shield class="size-4 text-primary" />
                                </div>
                                <div class="min-w-0 flex-1">
                                    <h3 class="text-sm font-semibold text-text">New version available</h3>
                                    <p class="text-xs text-text-muted mt-1 max-w-2xl">
                                        {updateInfo.latest_version} is available on the {effectiveReleaseChannel()} track. Update when you are ready to restart the process.
                                    </p>
                                    <div class="mt-4 flex flex-wrap items-center gap-3">
                                        <Button
                                            size="sm"
                                            loading={updateApplying}
                                            onclick={() => (updateConfirmOpen = true)}
                                        >
                                            {updateApplying ? 'Updating...' : 'Update Now'}
                                        </Button>
                                        {#if updateInfo.release_url}
                                            <a
                                                href={updateInfo.release_url}
                                                target="_blank"
                                                rel="noreferrer"
                                                class="text-xs font-medium text-primary hover:underline"
                                            >
                                                Open release notes
                                            </a>
                                        {/if}
                                    </div>
                                </div>
                            </div>
                        </div>
                    {:else}
                        <div class="rounded-2xl border border-success/20 bg-success/10 p-4">
                            <div class="flex items-start gap-3">
                                <div class="size-8 rounded-lg bg-success/20 flex items-center justify-center shrink-0">
                                    <Shield class="size-4 text-success" />
                                </div>
                                <div>
                                    <h3 class="text-sm font-semibold text-text">Version check complete</h3>
                                    <p class="text-xs text-text-muted mt-1">
                                        {#if effectiveReleaseChannel() === 'stable' && isPrereleaseVersion(updateInfo.current_version)}
                                            This installed prerelease build is already ahead of the latest stable release.
                                        {:else}
                                            This instance is current for the selected release track.
                                        {/if}
                                    </p>
                                </div>
                            </div>
                        </div>
                    {/if}
                </div>

                <aside class="rounded-2xl border border-border/60 bg-surface/25 p-5 space-y-4">
                    <div>
                        <h3 class="text-sm font-semibold text-text">Release Snapshot</h3>
                        <p class="text-xs text-text-muted mt-1">
                            Extra context for the build selected by this page.
                        </p>
                    </div>

                    <dl class="grid gap-4 text-sm">
                        <div>
                            <dt class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                Release Channel
                            </dt>
                            <dd class="mt-1 text-text">
                                {effectiveReleaseChannel() === 'stable'
                                    ? 'Stable releases only'
                                    : 'Stable and prerelease builds'}
                            </dd>
                        </div>

                        <div>
                            <dt class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                Published
                            </dt>
                            <dd class="mt-1 text-text">
                                {formatPublishedAt(updateInfo.published_at)}
                            </dd>
                        </div>

                        {#if updateInfo.asset_name}
                            <div>
                                <dt class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                    Matching Asset
                                </dt>
                                <dd class="mt-1 font-mono text-xs text-text break-all">
                                    {updateInfo.asset_name}
                                </dd>
                            </div>
                        {/if}
                    </dl>

                    {#if releaseNotesItems.length > 0}
                        <div class="space-y-2">
                            <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                                Release Notes Preview
                            </p>
                            <ul class="space-y-2 text-xs text-text-muted">
                                {#each releaseNotesItems as item, index (`${item}-${index}`)}
                                    <li class="rounded-xl border border-border/50 bg-background/50 px-3 py-2">
                                        {item}
                                    </li>
                                {/each}
                            </ul>
                        </div>
                    {/if}

                    {#if updateInfo.release_url}
                        <a
                            href={updateInfo.release_url}
                            target="_blank"
                            rel="noreferrer"
                            class="inline-flex items-center text-xs font-medium text-primary hover:underline"
                        >
                            View full release details
                        </a>
                    {/if}
                </aside>
            </div>
        {:else}
            <Button variant="outline" onclick={checkUpdate}>Check for Updates</Button>
        {/if}

        {#if updateMsg}
            <div
                class={`p-3 rounded-lg text-sm border ${updateMsg.startsWith('Error') ? 'bg-danger/10 border-danger/20 text-danger' : 'bg-success/10 border-success/20 text-success'}`}
                aria-live="polite"
            >
                {updateMsg}
            </div>
        {/if}
    </section>

    {#await apiTokenManagerPromise}
        <section class="card space-y-4">
            <Skeleton height="h-5" width="w-40" />
            <Skeleton height="h-16" />
            <Skeleton height="h-28" />
        </section>
    {:then { default: APITokenManagerComponent }}
        <APITokenManagerComponent onAuditRefresh={refreshAuditLogs} />
    {:catch}
        <section class="card p-4 text-sm text-danger bg-danger/10 border border-danger/20">
            Token management failed to load. Refresh the page to try again.
        </section>
    {/await}

    {#await auditLogBrowserPromise}
        <section class="card space-y-4">
            <Skeleton height="h-5" width="w-32" />
            <Skeleton height="h-16" />
            <Skeleton height="h-28" />
        </section>
    {:then { default: AuditLogBrowserComponent }}
        <AuditLogBrowserComponent refreshVersion={auditRefreshVersion} />
    {:catch}
        <section class="card p-4 text-sm text-danger bg-danger/10 border border-danger/20">
            Audit history failed to load. Refresh the page to try again.
        </section>
    {/await}

    <ConfirmActionDialog
        bind:open={updateConfirmOpen}
        title="Apply Update"
        description="The current updu process will download the selected release, replace the running binary, and restart. Use this when a brief reconnect is acceptable."
        confirmLabel="Update & Restart"
        confirmVariant="default"
        loading={updateApplying}
        onConfirm={applyUpdate}
    >
        <div class="grid gap-3 text-sm sm:grid-cols-2">
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Current
                </p>
                <p class="mt-1 font-mono text-text break-all">
                    {updateInfo?.current_version || 'unknown'}
                </p>
            </div>
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Update To
                </p>
                <p class="mt-1 font-mono text-text break-all">
                    {updateInfo?.latest_version || 'unknown'}
                </p>
            </div>
        </div>
    </ConfirmActionDialog>
</div>