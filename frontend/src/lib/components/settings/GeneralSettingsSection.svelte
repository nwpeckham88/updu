<script lang="ts">
    import { onMount } from 'svelte';
    import { Bell, Globe, Image, Lock, Palette } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import Skeleton from '$lib/components/ui/skeleton.svelte';
    import {
        GENERAL_SETTINGS_KEYS,
        changePassword as changePasswordRequest,
        getSettings,
        type GeneralSettingKey,
        type SettingsMap,
        updateSettings,
    } from '$lib/api/settings';
    import { settingsStore } from '$lib/stores/settings.svelte';

    const themeOptions = [
        { value: 'dark', label: 'Dark' },
        { value: 'light', label: 'Light' },
    ];

    let settings = $state<SettingsMap>({});
    let settingsLoading = $state(true);
    let settingsSaving = $state(false);
    let settingsMsg = $state('');

    let pwCurrent = $state('');
    let pwNew = $state('');
    let pwConfirm = $state('');
    let pwSaving = $state(false);
    let pwMsg = $state('');

    function syncSettings(source: SettingsMap) {
        const nextSettings = { ...source };

        if (!nextSettings.theme) {
            nextSettings.theme = 'dark';
        }

        if (!nextSettings.enable_custom_css) {
            nextSettings.enable_custom_css = 'false';
        }

        settings = nextSettings;
    }

    async function loadSettings() {
        try {
            settingsLoading = true;
            syncSettings((await getSettings()) || {});
        } catch {
            settings = {};
        } finally {
            settingsLoading = false;
        }
    }

    function updateSetting(key: GeneralSettingKey, value: string) {
        settings = {
            ...settings,
            [key]: value,
        };
    }

    function settingValue(key: GeneralSettingKey): string {
        return settings[key] ?? '';
    }

    function booleanSetting(key: GeneralSettingKey, fallback = false): boolean {
        const value = settings[key];

        if (value === undefined || value === '') {
            return fallback;
        }

        return value === 'true';
    }

    function toggleSetting(key: GeneralSettingKey, nextValue: boolean) {
        updateSetting(key, nextValue ? 'true' : 'false');
    }

    function selectedThemeLabel(): string {
        return (
            themeOptions.find((option) => option.value === settingValue('theme'))
                ?.label ?? 'Dark'
        );
    }

    function instanceVisibilityLabel(): string {
        return booleanSetting('enable_public') ? 'Public' : 'Private';
    }

    function buildSettingsPayload(): SettingsMap {
        const payload: SettingsMap = {};

        for (const key of GENERAL_SETTINGS_KEYS) {
            const value = settings[key];

            if (value !== undefined) {
                payload[key] = value;
            }
        }

        return payload;
    }

    async function saveSettings() {
        settingsSaving = true;
        settingsMsg = '';

        try {
            await updateSettings(buildSettingsPayload());
            await settingsStore.refresh();
            await loadSettings();
            settingsMsg = 'Settings saved successfully.';
            setTimeout(() => (settingsMsg = ''), 3000);
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Failed to save settings';
            settingsMsg = `Error: ${message}`;
        } finally {
            settingsSaving = false;
        }
    }

    async function changePassword() {
        if (pwNew !== pwConfirm) {
            pwMsg = 'Error: Passwords do not match';
            return;
        }

        if (pwNew.length < 8) {
            pwMsg = 'Error: New password must be at least 8 characters';
            return;
        }

        pwSaving = true;
        pwMsg = '';

        try {
            await changePasswordRequest(pwCurrent, pwNew);
            pwMsg = 'Password changed successfully.';
            pwCurrent = '';
            pwNew = '';
            pwConfirm = '';
            setTimeout(() => (pwMsg = ''), 3000);
        } catch (error) {
            const message =
                error instanceof Error
                    ? error.message
                    : 'Failed to change password';
            pwMsg = `Error: ${message}`;
        } finally {
            pwSaving = false;
        }
    }

    onMount(() => {
        void loadSettings();
    });
</script>

<div class="space-y-6">
    {#if settingsMsg}
        <div
            class={`p-3 rounded-lg text-sm border ${settingsMsg.startsWith('Error') ? 'bg-danger/10 border-danger/20 text-danger' : 'bg-success/10 border-success/20 text-success'}`}
            aria-live="polite"
        >
            {settingsMsg}
        </div>
    {/if}

    <section class="card space-y-5">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <Globe class="size-4 text-primary" />
            </div>
            <div>
                <h2 class="text-base font-semibold text-text">Instance Profile</h2>
                <p class="text-[11px] text-text-subtle mt-0.5">
                    Define the public identity and core URLs for this updu instance.
                </p>
                {#if !settingsLoading}
                    <div class="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
                        <span class="inline-flex items-center rounded-full border border-primary/20 bg-primary/8 px-2.5 py-1 font-semibold text-primary">
                            {selectedThemeLabel()} theme
                        </span>
                        <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2.5 py-1 text-text-muted">
                            {instanceVisibilityLabel()}
                        </span>
                        <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2.5 py-1 text-text-muted">
                            {booleanSetting('enable_custom_css') ? 'Custom CSS enabled' : 'Default styling'}
                        </span>
                    </div>
                {/if}
            </div>
        </div>

        {#if settingsLoading}
            <div class="grid gap-4 lg:grid-cols-2">
                {#each Array.from({ length: 6 }) as _, index (index)}
                    <div class="space-y-2">
                        <Skeleton height="h-4" width="w-28" />
                        <Skeleton height="h-10" />
                    </div>
                {/each}
            </div>
        {:else}
            <div class="grid gap-4 lg:grid-cols-2">
                <div class="space-y-1.5 lg:col-span-2">
                    <label class="text-sm font-medium text-text-muted" for="site-name">
                        Site Name
                    </label>
                    <input
                        id="site-name"
                        name="site_name"
                        type="text"
                        value={settingValue('site_name')}
                        oninput={(event) =>
                            updateSetting(
                                'site_name',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="updu"
                    />
                </div>

                <div class="space-y-1.5 lg:col-span-2">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="site-description"
                    >
                        Site Description
                    </label>
                    <textarea
                        id="site-description"
                        name="site_description"
                        class="input-base min-h-28 resize-y"
                        value={settingValue('site_description')}
                        placeholder="Describe who this instance serves and what it monitors."
                        oninput={(event) =>
                            updateSetting(
                                'site_description',
                                (event.currentTarget as HTMLTextAreaElement).value,
                            )}
                    ></textarea>
                </div>

                <div class="space-y-1.5 lg:col-span-2">
                    <label class="text-sm font-medium text-text-muted" for="base-url">
                        Base URL
                    </label>
                    <input
                        id="base-url"
                        name="base_url"
                        type="url"
                        value={settingValue('base_url')}
                        oninput={(event) =>
                            updateSetting(
                                'base_url',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="https://status.example.com"
                    />
                    <p class="text-[11px] text-text-subtle">
                        Used in exported links, notifications, and status page references.
                    </p>
                </div>
            </div>
        {/if}
    </section>

    <section class="card space-y-5">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <Image class="size-4 text-primary" />
            </div>
            <div>
                <h2 class="text-base font-semibold text-text">Branding & Appearance</h2>
                <p class="text-[11px] text-text-subtle mt-0.5">
                    Configure branding assets and the default UI theme for new sessions.
                </p>
            </div>
        </div>

        {#if settingsLoading}
            <div class="grid gap-4 lg:grid-cols-2">
                {#each Array.from({ length: 3 }) as _, index (index)}
                    <div class="space-y-2">
                        <Skeleton height="h-4" width="w-24" />
                        <Skeleton height="h-10" />
                    </div>
                {/each}
            </div>
        {:else}
            <div class="grid gap-4 lg:grid-cols-2">
                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="logo-url">
                        Logo URL
                    </label>
                    <input
                        id="logo-url"
                        type="url"
                        value={settingValue('logo_url')}
                        oninput={(event) =>
                            updateSetting(
                                'logo_url',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="https://cdn.example.com/logo.svg"
                    />
                </div>

                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="favicon-url"
                    >
                        Favicon URL
                    </label>
                    <input
                        id="favicon-url"
                        type="url"
                        value={settingValue('favicon_url')}
                        oninput={(event) =>
                            updateSetting(
                                'favicon_url',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="https://cdn.example.com/favicon.ico"
                    />
                </div>

                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="theme">
                        Default Theme
                    </label>
                    <select
                        id="theme"
                        class="input-base text-sm"
                        value={settingValue('theme')}
                        onchange={(event) =>
                            updateSetting(
                                'theme',
                                (event.currentTarget as HTMLSelectElement).value,
                            )}
                    >
                        {#each themeOptions as option (option.value)}
                            <option value={option.value}>{option.label}</option>
                        {/each}
                    </select>
                </div>

                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="timezone">
                        Timezone
                    </label>
                    <input
                        id="timezone"
                        type="text"
                        value={settingValue('timezone')}
                        oninput={(event) =>
                            updateSetting(
                                'timezone',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="UTC"
                    />
                </div>

                <div class="space-y-1.5 lg:col-span-2">
                    <label class="text-sm font-medium text-text-muted" for="date-format">
                        Date Format
                    </label>
                    <input
                        id="date-format"
                        type="text"
                        value={settingValue('date_format')}
                        oninput={(event) =>
                            updateSetting(
                                'date_format',
                                (event.currentTarget as HTMLInputElement).value,
                            )}
                        class="input-base"
                        placeholder="2006-01-02 15:04"
                    />
                    <p class="text-[11px] text-text-subtle">
                        Keep this readable and consistent with the timezone above.
                    </p>
                </div>
            </div>
        {/if}
    </section>

    <section class="card space-y-5">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <Bell class="size-4 text-primary" />
            </div>
            <div>
                <h2 class="text-base font-semibold text-text">Access & Alerts</h2>
                <p class="text-[11px] text-text-subtle mt-0.5">
                    Control public visibility, maintenance mode, and default notification behavior.
                </p>
            </div>
        </div>

        {#if settingsLoading}
            <div class="space-y-3">
                {#each Array.from({ length: 4 }) as _, index (index)}
                    <div class="rounded-xl border border-border/60 p-4 flex items-center justify-between gap-4">
                        <div class="space-y-2 flex-1">
                            <Skeleton height="h-4" width="w-36" />
                            <Skeleton height="h-3" width="w-56" />
                        </div>
                        <Skeleton height="h-6" width="w-10" rounded="rounded-full" />
                    </div>
                {/each}
            </div>
        {:else}
            <div class="grid gap-3 lg:grid-cols-2">
                <label class="rounded-xl border border-border/60 p-4 flex items-start justify-between gap-4 cursor-pointer select-none">
                    <div>
                        <p class="text-sm font-medium text-text">Enable Public Access</p>
                        <p class="text-[11px] text-text-subtle mt-1">
                            Allow public-facing pages and links to surface this instance.
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={booleanSetting('enable_public')}
                        onchange={(event) =>
                            toggleSetting(
                                'enable_public',
                                (event.currentTarget as HTMLInputElement).checked,
                            )}
                        class="mt-1 size-4 accent-primary"
                    />
                </label>

                <label class="rounded-xl border border-border/60 p-4 flex items-start justify-between gap-4 cursor-pointer select-none">
                    <div>
                        <p class="text-sm font-medium text-text">Maintenance Mode</p>
                        <p class="text-[11px] text-text-subtle mt-1">
                            Temporarily suppress normal operations during planned work.
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={booleanSetting('maintenance_mode')}
                        onchange={(event) =>
                            toggleSetting(
                                'maintenance_mode',
                                (event.currentTarget as HTMLInputElement).checked,
                            )}
                        class="mt-1 size-4 accent-primary"
                    />
                </label>

                <label class="rounded-xl border border-border/60 p-4 flex items-start justify-between gap-4 cursor-pointer select-none">
                    <div>
                        <p class="text-sm font-medium text-text">Notify on Down</p>
                        <p class="text-[11px] text-text-subtle mt-1">
                            Emit a notification the moment a monitor enters a failing state.
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={booleanSetting('notify_on_down', true)}
                        onchange={(event) =>
                            toggleSetting(
                                'notify_on_down',
                                (event.currentTarget as HTMLInputElement).checked,
                            )}
                        class="mt-1 size-4 accent-primary"
                    />
                </label>

                <label class="rounded-xl border border-border/60 p-4 flex items-start justify-between gap-4 cursor-pointer select-none">
                    <div>
                        <p class="text-sm font-medium text-text">Notify on Recovery</p>
                        <p class="text-[11px] text-text-subtle mt-1">
                            Emit a notification when a monitor returns to a healthy state.
                        </p>
                    </div>
                    <input
                        type="checkbox"
                        checked={booleanSetting('notify_on_up', true)}
                        onchange={(event) =>
                            toggleSetting(
                                'notify_on_up',
                                (event.currentTarget as HTMLInputElement).checked,
                            )}
                        class="mt-1 size-4 accent-primary"
                    />
                </label>
            </div>

            <div class="flex justify-end">
                <Button loading={settingsSaving} onclick={saveSettings}>
                    {settingsSaving ? 'Saving...' : 'Save Settings'}
                </Button>
            </div>
        {/if}
    </section>

    <section class="card space-y-5">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <Palette class="size-4 text-primary" />
            </div>
            <div>
                <h2 class="text-base font-semibold text-text">Custom CSS</h2>
                <p class="text-[11px] text-text-subtle mt-0.5">
                    Apply global visual overrides for your embedded frontend.
                </p>
            </div>
        </div>

        {#if settingsLoading}
            <div class="space-y-3">
                <Skeleton height="h-6" width="w-40" />
                <Skeleton height="h-40" />
            </div>
        {:else}
            <label class="rounded-xl border border-border/60 p-4 flex items-start justify-between gap-4 cursor-pointer select-none">
                <div>
                    <p class="text-sm font-medium text-text">Enable Custom CSS</p>
                    <p class="text-[11px] text-text-subtle mt-1">
                        The backend still sanitizes CSS before serving it to the app shell.
                    </p>
                </div>
                <input
                    type="checkbox"
                    checked={booleanSetting('enable_custom_css')}
                    onchange={(event) =>
                        toggleSetting(
                            'enable_custom_css',
                            (event.currentTarget as HTMLInputElement).checked,
                        )}
                    class="mt-1 size-4 accent-primary"
                />
            </label>

            <div class="space-y-1.5">
                <label class="text-sm font-medium text-text-muted" for="custom-css-editor">
                    CSS Overrides
                </label>
                <textarea
                    id="custom-css-editor"
                    class="input-base font-mono text-xs w-full min-h-48 resize-y"
                    spellcheck="false"
                    value={settingValue('custom_css')}
                    placeholder={"/* Override colors or spacing */\n:root {\n  --color-primary: hsl(210 90% 56%);\n}"}
                    oninput={(event) =>
                        updateSetting(
                            'custom_css',
                            (event.currentTarget as HTMLTextAreaElement).value,
                        )}
                ></textarea>
                <p class="text-[11px] text-text-subtle">
                    Saved CSS is available at <span class="font-mono text-primary/80">/api/v1/custom.css</span>.
                </p>
            </div>

            <div class="flex justify-end">
                <Button variant="outline" loading={settingsSaving} onclick={saveSettings}>
                    {settingsSaving ? 'Saving...' : 'Save CSS'}
                </Button>
            </div>
        {/if}
    </section>

    <section class="card space-y-5">
        <div class="flex items-start gap-3">
            <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                <Lock class="size-4 text-primary" />
            </div>
            <div>
                <h2 class="text-base font-semibold text-text">Change Password</h2>
                <p class="text-[11px] text-text-subtle mt-0.5">
                    Update the current administrator password for this session login.
                </p>
            </div>
        </div>

        {#if pwMsg}
            <div
                class={`p-3 rounded-lg text-sm border ${pwMsg.startsWith('Error') ? 'bg-danger/10 border-danger/20 text-danger' : 'bg-success/10 border-success/20 text-success'}`}
                aria-live="polite"
            >
                {pwMsg}
            </div>
        {/if}

        <div class="grid gap-4 lg:grid-cols-3">
            <div class="space-y-1.5">
                <label class="text-sm font-medium text-text-muted" for="pw-current">
                    Current Password
                </label>
                <input
                    id="pw-current"
                    type="password"
                    bind:value={pwCurrent}
                    class="input-base"
                />
            </div>

            <div class="space-y-1.5">
                <label class="text-sm font-medium text-text-muted" for="pw-new">
                    New Password
                </label>
                <input
                    id="pw-new"
                    type="password"
                    bind:value={pwNew}
                    class="input-base"
                    placeholder="Minimum 8 characters"
                />
            </div>

            <div class="space-y-1.5">
                <label class="text-sm font-medium text-text-muted" for="pw-confirm">
                    Confirm Password
                </label>
                <input
                    id="pw-confirm"
                    type="password"
                    bind:value={pwConfirm}
                    class="input-base"
                />
            </div>
        </div>

        <div class="flex justify-end">
            <Button loading={pwSaving} onclick={changePassword}>
                {pwSaving ? 'Saving...' : 'Change Password'}
            </Button>
        </div>
    </section>
</div>