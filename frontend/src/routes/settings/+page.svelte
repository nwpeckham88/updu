<script lang="ts">
    import { onMount } from "svelte";
    import { fetchAPI } from "$lib/api/client";
    import { authStore } from "$lib/stores/auth.svelte";
    import { settingsStore } from "$lib/stores/settings.svelte";
    import {
        Settings,
        Bell,
        Users,
        HardDrive,
        Plus,
        Pencil,
        Trash2,
        Send,
        Download,
        Upload,
        X,
        Shield,
        ShieldAlert,
        Eye,
        Lock,
    } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import Skeleton from "$lib/components/ui/skeleton.svelte";
    import EmptyState from "$lib/components/ui/empty-state.svelte";
    import { Dialog } from "bits-ui";

    // --- Tab State ---
    type Tab = "general" | "notifications" | "users" | "backup";
    let activeTab = $state<Tab>("general");

    const tabs: { id: Tab; label: string; icon: any }[] = [
        { id: "general", label: "General", icon: Settings },
        { id: "notifications", label: "Notifications", icon: Bell },
        { id: "users", label: "Users", icon: Users },
        { id: "backup", label: "Backup", icon: HardDrive },
    ];

    // ===== GENERAL =====
    let settings = $state<Record<string, string>>({});
    let settingsLoading = $state(true);
    let settingsSaving = $state(false);
    let settingsMsg = $state("");

    // Keys managed by dedicated UI cards (Dashboard Customization, Custom CSS)
    const managedKeys = new Set(["dashboard_style", "custom_css"]);

    // General settings = everything except managed keys
    const generalSettings = $derived(
        Object.entries(settings).filter(([k]) => !managedKeys.has(k)),
    );

    async function loadSettings() {
        try {
            settingsLoading = true;
            const data = await fetchAPI("/api/v1/settings");
            settings = data || {};
            // Ensure dashboard settings have default values
            if (!settings["dashboard_style"])
                settings["dashboard_style"] = "default";
        } catch {
            settings = {};
        } finally {
            settingsLoading = false;
        }
    }

    async function saveSettings() {
        settingsSaving = true;
        settingsMsg = "";
        try {
            await fetchAPI("/api/v1/settings", {
                method: "POST",
                body: JSON.stringify(settings),
            });
            settingsMsg = "Settings saved successfully.";
            // Refresh the global settings store so dashboard reflects changes immediately
            await settingsStore.refresh();
            setTimeout(() => (settingsMsg = ""), 3000);
        } catch (e: any) {
            settingsMsg = "Error: " + (e.message || "Failed to save");
        } finally {
            settingsSaving = false;
        }
    }

    // ===== NOTIFICATIONS =====
    let channels = $state<any[]>([]);
    let channelsLoading = $state(true);
    let ncDialogOpen = $state(false);
    let ncEditTarget = $state<any>(null);
    let ncName = $state("");
    let ncType = $state("webhook");
    let ncEnabled = $state(true);
    let ncConfigUrl = $state("");
    let ncEmailHost = $state("");
    let ncEmailPort = $state<number>(587);
    let ncEmailUser = $state("");
    let ncEmailPass = $state("");
    let ncEmailFrom = $state("");
    let ncEmailTo = $state("");
    let ncSaving = $state(false);
    let ncError = $state("");
    let ncTestingId = $state<string | null>(null);

    async function loadChannels() {
        try {
            channelsLoading = true;
            const data = await fetchAPI("/api/v1/notifications");
            channels = data || [];
        } catch {
            channels = [];
        } finally {
            channelsLoading = false;
        }
    }

    function openCreateChannel() {
        ncEditTarget = null;
        ncName = "";
        ncType = "webhook";
        ncEnabled = true;
        ncConfigUrl = "";
        ncEmailHost = "";
        ncEmailPort = 587;
        ncEmailUser = "";
        ncEmailPass = "";
        ncEmailFrom = "";
        ncEmailTo = "";
        ncError = "";
        ncDialogOpen = true;
    }

    function openEditChannel(ch: any) {
        ncEditTarget = ch;
        ncName = ch.name;
        ncType = ch.type;
        ncEnabled = ch.enabled;
        ncConfigUrl = ch.config?.url || "";
        ncEmailHost = ch.config?.host || "";
        ncEmailPort = ch.config?.port || 587;
        ncEmailUser = ch.config?.user || "";
        ncEmailPass = ch.config?.pass || "";
        ncEmailFrom = ch.config?.from || "";
        ncEmailTo = ch.config?.to || "";
        ncError = "";
        ncDialogOpen = true;
    }

    async function saveChannel() {
        if (!ncName.trim()) {
            ncError = "Name is required";
            return;
        }
        ncSaving = true;
        ncError = "";
        try {
            let configPayload: any = {};
            if (ncType === "email") {
                configPayload = {
                    host: ncEmailHost,
                    port: ncEmailPort,
                    user: ncEmailUser,
                    pass: ncEmailPass,
                    from: ncEmailFrom,
                    to: ncEmailTo,
                };
            } else {
                configPayload = { url: ncConfigUrl };
            }

            const body = {
                name: ncName,
                type: ncType,
                enabled: ncEnabled,
                config: configPayload,
            };
            if (ncEditTarget) {
                await fetchAPI(`/api/v1/notifications/${ncEditTarget.id}`, {
                    method: "PUT",
                    body: JSON.stringify(body),
                });
            } else {
                await fetchAPI("/api/v1/notifications", {
                    method: "POST",
                    body: JSON.stringify(body),
                });
            }
            ncDialogOpen = false;
            loadChannels();
        } catch (e: any) {
            ncError = e.message || "Failed to save";
        } finally {
            ncSaving = false;
        }
    }

    async function deleteChannel(id: string) {
        if (!confirm("Delete this notification channel?")) return;
        await fetchAPI(`/api/v1/notifications/${id}`, { method: "DELETE" });
        loadChannels();
    }

    async function testChannel(id: string) {
        ncTestingId = id;
        try {
            await fetchAPI(`/api/v1/notifications/${id}/test`, {
                method: "POST",
            });
        } catch {
            // silently fail — the backend dispatches async
        } finally {
            setTimeout(() => (ncTestingId = null), 2000);
        }
    }

    // ===== USERS =====
    let users = $state<any[]>([]);
    let usersLoading = $state(true);
    let userDialogOpen = $state(false);
    let newUsername = $state("");
    let newPassword = $state("");
    let userSaving = $state(false);
    let userError = $state("");

    async function loadUsers() {
        try {
            usersLoading = true;
            const data = await fetchAPI("/api/v1/admin/users");
            users = data || [];
        } catch {
            users = [];
        } finally {
            usersLoading = false;
        }
    }

    async function changeRole(userId: string, role: string) {
        await fetchAPI(`/api/v1/admin/users/${userId}/role`, {
            method: "PUT",
            body: JSON.stringify({ role }),
        });
        loadUsers();
    }

    async function deleteUser(userId: string) {
        if (!confirm("Delete this user?")) return;
        await fetchAPI(`/api/v1/admin/users/${userId}`, { method: "DELETE" });
        loadUsers();
    }

    function openInviteUser() {
        newUsername = "";
        newPassword = "";
        userError = "";
        userDialogOpen = true;
    }

    async function inviteUser() {
        if (!newUsername.trim() || !newPassword.trim()) {
            userError = "Username and password are required";
            return;
        }
        userSaving = true;
        userError = "";
        try {
            await fetchAPI("/api/v1/auth/register", {
                method: "POST",
                body: JSON.stringify({
                    username: newUsername,
                    password: newPassword,
                }),
            });
            userDialogOpen = false;
            loadUsers();
        } catch (e: any) {
            userError = e.message || "Failed to create user";
        } finally {
            userSaving = false;
        }
    }

    // ===== BACKUP =====
    let importing = $state(false);
    let backupMsg = $state("");

    async function exportBackup() {
        try {
            const data = await fetchAPI("/api/v1/system/backup");
            const blob = new Blob([JSON.stringify(data, null, 2)], {
                type: "application/json",
            });
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = "updu-backup.json";
            a.click();
            URL.revokeObjectURL(url);
            backupMsg = "Backup exported successfully.";
            setTimeout(() => (backupMsg = ""), 3000);
        } catch (e: any) {
            backupMsg = "Error: " + (e.message || "Export failed");
        }
    }

    async function importBackup(event: Event) {
        const input = event.target as HTMLInputElement;
        const file = input.files?.[0];
        if (!file) return;
        importing = true;
        backupMsg = "";
        try {
            const text = await file.text();
            await fetchAPI("/api/v1/system/backup", {
                method: "POST",
                body: text,
            });
            backupMsg = "Configuration imported successfully.";
            setTimeout(() => (backupMsg = ""), 3000);
        } catch (e: any) {
            backupMsg = "Error: " + (e.message || "Import failed");
        } finally {
            importing = false;
            input.value = "";
        }
    }

    // ===== PASSWORD CHANGE =====
    let pwCurrent = $state("");
    let pwNew = $state("");
    let pwConfirm = $state("");
    let pwSaving = $state(false);
    let pwMsg = $state("");

    async function changePassword() {
        if (pwNew !== pwConfirm) {
            pwMsg = "Error: Passwords do not match";
            return;
        }
        if (pwNew.length < 8) {
            pwMsg = "Error: New password must be at least 8 characters";
            return;
        }
        pwSaving = true;
        pwMsg = "";
        try {
            await fetchAPI("/api/v1/auth/password", {
                method: "PUT",
                body: JSON.stringify({
                    current_password: pwCurrent,
                    new_password: pwNew,
                }),
            });
            pwMsg = "Password changed successfully.";
            pwCurrent = "";
            pwNew = "";
            pwConfirm = "";
            setTimeout(() => (pwMsg = ""), 3000);
        } catch (e: any) {
            pwMsg = "Error: " + (e.message || "Failed to change password");
        } finally {
            pwSaving = false;
        }
    }

    // ===== INIT =====
    onMount(() => {
        loadSettings();
        loadChannels();
        loadUsers();
    });
</script>

<svelte:head>
    <title>Settings – updu</title>
</svelte:head>

<div class="max-w-6xl mx-auto w-full pb-10 space-y-8">
    <!-- Header & Navigation -->
    <header>
        <h1 class="text-2xl font-bold tracking-tight text-text mb-1">
            Settings
        </h1>
        <p class="text-sm text-text-muted mb-6">
            Manage your instance configuration
        </p>

        <nav
            class="flex items-center gap-1 border-b border-border/60 pb-px overflow-x-auto no-scrollbar"
        >
            {#each tabs as t}
                <button
                    onclick={() => (activeTab = t.id)}
                    class="flex items-center gap-2.5 px-5 py-3 rounded-t-xl text-sm font-medium transition-all relative group shrink-0 {activeTab ===
                    t.id
                        ? 'text-primary bg-primary/5'
                        : 'text-text-muted hover:text-text hover:bg-surface-elevated'}"
                >
                    <t.icon class="size-4" />
                    <span>{t.label}</span>
                    {#if activeTab === t.id}
                        <div
                            class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-full shadow-[0_0_10px_rgba(59,130,246,0.5)]"
                        ></div>
                    {/if}
                </button>
            {/each}
        </nav>
    </header>

    <!-- Main Content Area -->
    <main class="space-y-6 min-w-0">
        <!-- ===== GENERAL TAB ===== -->
        {#if activeTab === "general"}
            <div class="card">
                {#if settingsLoading}
                    <div class="space-y-4">
                        {#each { length: 3 } as _}
                            <div class="flex items-center gap-4">
                                <Skeleton height="h-4" width="w-1/4" />
                                <Skeleton height="h-9" width="w-1/2" />
                            </div>
                        {/each}
                    </div>
                {:else if generalSettings.length === 0}
                    <EmptyState
                        icon={Settings}
                        title="No general settings configured"
                        description="General settings like site name will appear here. Dashboard options are below."
                    />
                {:else}
                    <div class="space-y-4">
                        {#each generalSettings as [key, value]}
                            <div
                                class="flex flex-col sm:flex-row sm:items-center gap-2"
                            >
                                <label
                                    for="setting-{key}"
                                    class="text-sm font-medium text-text-muted w-48 shrink-0 capitalize"
                                >
                                    {key.replace(/_/g, " ")}
                                </label>
                                <input
                                    id="setting-{key}"
                                    type="text"
                                    bind:value={settings[key]}
                                    class="input-base flex-1"
                                />
                            </div>
                        {/each}
                    </div>

                    {#if settingsMsg}
                        <div
                            class="mt-4 p-3 rounded-lg text-sm {settingsMsg.startsWith(
                                'Error',
                            )
                                ? 'bg-danger/10 border border-danger/20 text-danger'
                                : 'bg-success/10 border border-success/20 text-success'}"
                        >
                            {settingsMsg}
                        </div>
                    {/if}

                    <div class="flex justify-end mt-5">
                        <Button loading={settingsSaving} onclick={saveSettings}>
                            {settingsSaving ? "Saving..." : "Save Settings"}
                        </Button>
                    </div>
                {/if}
            </div>

            <!-- Dashboard Customization -->
            <div class="card">
                <div class="flex items-center gap-3 mb-4">
                    <div
                        class="size-9 rounded-xl bg-primary/10 flex items-center justify-center"
                    >
                        <Settings class="size-4 text-primary" />
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-text">
                            Dashboard Customization
                        </h3>
                        <p class="text-[11px] text-text-subtle">
                            Personalize how your dashboard looks
                        </p>
                    </div>
                </div>

                <div class="space-y-4 max-w-lg">
                    <div class="flex items-center justify-between gap-4">
                        <div>
                            <label
                                for="dashboard-style"
                                class="text-sm font-medium text-text"
                                >Layout Style</label
                            >
                            <p class="text-[11px] text-text-subtle mt-0.5">
                                Choose between the default layout and a more
                                compact one.
                            </p>
                        </div>
                        <select
                            id="dashboard-style"
                            bind:value={settings["dashboard_style"]}
                            class="input-base w-32 shrink-0 text-sm"
                        >
                            <option value="default">Default</option>
                            <option value="compact">Compact</option>
                        </select>
                    </div>
                </div>

                <div class="flex justify-end mt-5">
                    <Button loading={settingsSaving} onclick={saveSettings}>
                        {settingsSaving ? "Saving..." : "Save Customizations"}
                    </Button>
                </div>
            </div>

            <!-- Custom CSS Editor -->
            <div class="card">
                <div class="flex items-center gap-3 mb-4">
                    <div
                        class="size-9 rounded-xl bg-primary/10 flex items-center justify-center"
                    >
                        <Pencil class="size-4 text-primary" />
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-text">
                            Custom CSS
                        </h3>
                        <p class="text-[11px] text-text-subtle">
                            Add custom styles to personalize your dashboard
                        </p>
                    </div>
                </div>
                <textarea
                    id="custom-css-editor"
                    bind:value={settings["custom_css"]}
                    placeholder={"/* Override CSS variables or add custom styles */\n:root {\n  --color-primary: hsl(280 80% 60%);\n}"}
                    class="input-base font-mono text-xs w-full"
                    style="min-height: 160px; resize: vertical; tab-size: 2;"
                    spellcheck="false"
                ></textarea>
                <div class="flex items-center justify-between mt-3">
                    <p class="text-[10px] text-text-subtle">
                        Saved CSS is served at <code class="text-primary/80"
                            >/api/v1/custom.css</code
                        > and injected into every page.
                    </p>
                    <Button
                        variant="outline"
                        size="sm"
                        loading={settingsSaving}
                        onclick={saveSettings}
                    >
                        {settingsSaving ? "Saving..." : "Save CSS"}
                    </Button>
                </div>
            </div>

            <!-- Password Change -->
            <div class="card">
                <div class="flex items-center gap-3 mb-4">
                    <div
                        class="size-9 rounded-xl bg-primary/10 flex items-center justify-center"
                    >
                        <Lock class="size-4 text-primary" />
                    </div>
                    <div>
                        <h3 class="text-sm font-semibold text-text">
                            Change Password
                        </h3>
                        <p class="text-[11px] text-text-subtle">
                            Update your account password
                        </p>
                    </div>
                </div>
                <div class="space-y-3 max-w-sm">
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="pw-current">Current Password</label
                        >
                        <input
                            id="pw-current"
                            type="password"
                            bind:value={pwCurrent}
                            class="input-base"
                        />
                    </div>
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="pw-new">New Password</label
                        >
                        <input
                            id="pw-new"
                            type="password"
                            bind:value={pwNew}
                            class="input-base"
                            placeholder="Min. 8 characters"
                        />
                    </div>
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="pw-confirm">Confirm New Password</label
                        >
                        <input
                            id="pw-confirm"
                            type="password"
                            bind:value={pwConfirm}
                            class="input-base"
                        />
                    </div>
                </div>
                {#if pwMsg}
                    <div
                        class="mt-3 p-3 rounded-lg text-sm {pwMsg.startsWith(
                            'Error',
                        )
                            ? 'bg-danger/10 border border-danger/20 text-danger'
                            : 'bg-success/10 border border-success/20 text-success'}"
                    >
                        {pwMsg}
                    </div>
                {/if}
                <div class="flex justify-end mt-4">
                    <Button loading={pwSaving} onclick={changePassword}>
                        {pwSaving ? "Saving..." : "Change Password"}
                    </Button>
                </div>
            </div>
        {/if}

        <!-- ===== NOTIFICATIONS TAB ===== -->
        {#if activeTab === "notifications"}
            <div class="flex justify-end">
                <Button onclick={openCreateChannel}>
                    <Plus class="size-4" />
                    New Channel
                </Button>
            </div>
            <div class="card overflow-hidden" style="padding: 0;">
                {#if channelsLoading}
                    <div class="divide-y divide-border">
                        {#each { length: 3 } as _}
                            <div class="p-5 flex gap-4">
                                <Skeleton
                                    height="h-9"
                                    width="w-9"
                                    rounded="rounded-xl"
                                />
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
                        description="Add a webhook, Discord, or Slack channel to get alerts."
                    >
                        <Button
                            onclick={openCreateChannel}
                            variant="outline"
                            size="sm">Add Channel</Button
                        >
                    </EmptyState>
                {:else}
                    <div class="divide-y divide-border/60">
                        {#each channels as ch (ch.id)}
                            <div
                                class="p-5 flex items-center gap-4 group hover:bg-surface/30 transition-colors"
                            >
                                <div
                                    class="size-9 rounded-xl flex items-center justify-center shrink-0 {ch.enabled
                                        ? 'bg-primary/10 text-primary'
                                        : 'bg-surface text-text-subtle'}"
                                >
                                    <Bell class="size-4" />
                                </div>
                                <div class="flex-1 min-w-0">
                                    <h3 class="font-semibold text-text text-sm">
                                        {ch.name}
                                    </h3>
                                    <p
                                        class="text-[11px] text-text-subtle mt-0.5"
                                    >
                                        {ch.type}
                                        {#if !ch.enabled}
                                            <span
                                                class="ml-1.5 text-warning font-medium"
                                                >· Disabled</span
                                            >
                                        {/if}
                                    </p>
                                </div>
                                <div
                                    class="shrink-0 flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
                                >
                                    <button
                                        onclick={() => testChannel(ch.id)}
                                        class="size-7 flex items-center justify-center rounded-lg hover:bg-primary/10 text-text-subtle hover:text-primary transition-colors"
                                        title="Send test"
                                        disabled={ncTestingId === ch.id}
                                    >
                                        <Send
                                            class="size-3.5 {ncTestingId ===
                                            ch.id
                                                ? 'animate-pulse'
                                                : ''}"
                                        />
                                    </button>
                                    <button
                                        onclick={() => openEditChannel(ch)}
                                        class="size-7 flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-subtle hover:text-text transition-colors"
                                        title="Edit"
                                    >
                                        <Pencil class="size-3.5" />
                                    </button>
                                    <button
                                        onclick={() => deleteChannel(ch.id)}
                                        class="size-7 flex items-center justify-center rounded-lg hover:bg-danger/10 text-text-subtle hover:text-danger transition-colors"
                                        title="Delete"
                                    >
                                        <Trash2 class="size-3.5" />
                                    </button>
                                </div>
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}

        <!-- ===== USERS TAB ===== -->
        {#if activeTab === "users"}
            <div class="flex justify-end">
                <Button onclick={openInviteUser}>
                    <Plus class="size-4" />
                    Invite User
                </Button>
            </div>
            <div class="card overflow-hidden" style="padding: 0;">
                {#if usersLoading}
                    <div class="divide-y divide-border">
                        {#each { length: 3 } as _}
                            <div class="p-5 flex gap-4">
                                <Skeleton
                                    height="h-9"
                                    width="w-9"
                                    rounded="rounded-full"
                                />
                                <div class="flex-1 space-y-2">
                                    <Skeleton height="h-4" width="w-1/3" />
                                    <Skeleton height="h-3" width="w-1/5" />
                                </div>
                            </div>
                        {/each}
                    </div>
                {:else if users.length === 0}
                    <EmptyState
                        icon={Users}
                        title="No users found"
                        description="Something went wrong fetching users."
                    />
                {:else}
                    <div class="divide-y divide-border/60">
                        {#each users as u (u.id)}
                            <div
                                class="p-5 flex items-center gap-4 group hover:bg-surface/30 transition-colors"
                            >
                                <div
                                    class="size-9 rounded-full bg-primary/10 flex items-center justify-center shrink-0"
                                >
                                    <span
                                        class="text-sm font-bold text-primary uppercase"
                                        >{u.username?.[0] || "?"}</span
                                    >
                                </div>
                                <div class="flex-1 min-w-0">
                                    <h3 class="font-semibold text-text text-sm">
                                        {u.username}
                                    </h3>
                                    <div
                                        class="flex items-center gap-1.5 mt-0.5 text-[11px] text-text-subtle"
                                    >
                                        {#if u.role === "admin"}
                                            <Shield
                                                class="size-3 text-primary"
                                            />
                                            <span
                                                class="text-primary font-medium"
                                                >Admin</span
                                            >
                                        {:else}
                                            <Eye class="size-3" />
                                            <span>Viewer</span>
                                        {/if}
                                    </div>
                                </div>
                                {#if u.id !== authStore.user?.id}
                                    <div
                                        class="shrink-0 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity"
                                    >
                                        <select
                                            value={u.role}
                                            onchange={(e) =>
                                                changeRole(
                                                    u.id,
                                                    (
                                                        e.target as HTMLSelectElement
                                                    ).value,
                                                )}
                                            class="input-base text-xs h-8 py-0 w-24"
                                        >
                                            <option value="admin">Admin</option>
                                            <option value="viewer"
                                                >Viewer</option
                                            >
                                        </select>
                                        <button
                                            onclick={() => deleteUser(u.id)}
                                            class="size-7 flex items-center justify-center rounded-lg hover:bg-danger/10 text-text-subtle hover:text-danger transition-colors"
                                            title="Delete user"
                                        >
                                            <Trash2 class="size-3.5" />
                                        </button>
                                    </div>
                                {:else}
                                    <span
                                        class="text-[10px] text-text-subtle italic"
                                        >You</span
                                    >
                                {/if}
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}

        <!-- ===== BACKUP TAB ===== -->
        {#if activeTab === "backup"}
            <div class="card space-y-6">
                <!-- Export -->
                <div>
                    <h3 class="text-sm font-semibold text-text mb-1">
                        Export Configuration
                    </h3>
                    <p class="text-xs text-text-muted mb-3">
                        Download all monitors, incidents, maintenance windows,
                        notification channels, and settings as a JSON backup.
                    </p>
                    <Button onclick={exportBackup} variant="outline">
                        <Download class="size-4" />
                        Export Backup
                    </Button>
                </div>

                <hr class="border-border/50" />

                <!-- Import -->
                <div>
                    <h3 class="text-sm font-semibold text-text mb-1">
                        Import Configuration
                    </h3>
                    <p class="text-xs text-text-muted mb-3">
                        Upload a previously exported JSON backup to restore your
                        configuration. This merges with existing data.
                    </p>
                    <label
                        class="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-transparent hover:bg-surface text-text text-sm font-medium tracking-wide cursor-pointer transition-all duration-150"
                    >
                        {#if importing}
                            <svg
                                class="size-4 animate-spin"
                                viewBox="0 0 24 24"
                                fill="none"
                            >
                                <circle
                                    class="opacity-25"
                                    cx="12"
                                    cy="12"
                                    r="10"
                                    stroke="currentColor"
                                    stroke-width="4"
                                />
                                <path
                                    class="opacity-75"
                                    fill="currentColor"
                                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                                />
                            </svg>
                            Importing...
                        {:else}
                            <Upload class="size-4" />
                            Choose File
                        {/if}
                        <input
                            type="file"
                            accept=".json"
                            class="sr-only"
                            onchange={importBackup}
                            disabled={importing}
                        />
                    </label>
                </div>

                {#if backupMsg}
                    <div
                        class="p-3 rounded-lg text-sm {backupMsg.startsWith(
                            'Error',
                        )
                            ? 'bg-danger/10 border border-danger/20 text-danger'
                            : 'bg-success/10 border border-success/20 text-success'}"
                    >
                        {backupMsg}
                    </div>
                {/if}
            </div>
        {/if}
    </main>
</div>

<!-- Notification Channel Dialog -->
<Dialog.Root bind:open={ncDialogOpen}>
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
                        {ncEditTarget ? "Edit Channel" : "New Channel"}
                    </Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        {ncEditTarget
                            ? "Update this notification channel."
                            : "Add a new notification channel."}
                    </Dialog.Description>
                </div>
                <Dialog.Close
                    class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                >
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if ncError}
                <div
                    class="mb-4 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm"
                >
                    {ncError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="nc-name"
                        >Name <span class="text-danger">*</span></label
                    >
                    <input
                        id="nc-name"
                        type="text"
                        bind:value={ncName}
                        placeholder="Production Alerts"
                        class="input-base"
                    />
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="nc-type">Type</label
                    >
                    <select
                        id="nc-type"
                        bind:value={ncType}
                        class="input-base text-sm"
                    >
                        <option value="webhook">Webhook</option>
                        <option value="discord">Discord</option>
                        <option value="slack">Slack</option>
                        <option value="email">Email</option>
                        <option value="ntfy">ntfy</option>
                    </select>
                </div>
                {#if ncType === "email"}
                    <div class="space-y-3">
                        <p class="text-[11px] text-text-subtle mb-2">
                            Configure an SMTP server to send email alerts. You
                            can use services like SendGrid, AWS SES, or a
                            standard email provider.
                        </p>
                        <div class="grid grid-cols-2 gap-3">
                            <div class="space-y-1.5">
                                <label
                                    class="text-sm font-medium text-text-muted"
                                    for="nc-e-host">SMTP Host</label
                                >
                                <input
                                    id="nc-e-host"
                                    type="text"
                                    bind:value={ncEmailHost}
                                    placeholder="smtp.example.com"
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    class="text-sm font-medium text-text-muted"
                                    for="nc-e-port">Port</label
                                >
                                <input
                                    id="nc-e-port"
                                    type="number"
                                    bind:value={ncEmailPort}
                                    class="input-base"
                                />
                            </div>
                        </div>
                        <div class="grid grid-cols-2 gap-3">
                            <div class="space-y-1.5">
                                <label
                                    class="text-sm font-medium text-text-muted"
                                    for="nc-e-user">Username</label
                                >
                                <input
                                    id="nc-e-user"
                                    type="text"
                                    bind:value={ncEmailUser}
                                    class="input-base"
                                />
                            </div>
                            <div class="space-y-1.5">
                                <label
                                    class="text-sm font-medium text-text-muted"
                                    for="nc-e-pass">Password</label
                                >
                                <input
                                    id="nc-e-pass"
                                    type="password"
                                    bind:value={ncEmailPass}
                                    class="input-base"
                                />
                            </div>
                        </div>
                        <div class="space-y-1.5">
                            <label
                                class="text-sm font-medium text-text-muted"
                                for="nc-e-from"
                                >From Address <span class="text-danger">*</span
                                ></label
                            >
                            <input
                                id="nc-e-from"
                                type="email"
                                bind:value={ncEmailFrom}
                                placeholder="alerts@example.com"
                                class="input-base"
                            />
                        </div>
                        <div class="space-y-1.5">
                            <label
                                class="text-sm font-medium text-text-muted"
                                for="nc-e-to"
                                >To Address(es) <span class="text-danger"
                                    >*</span
                                ></label
                            >
                            <input
                                id="nc-e-to"
                                type="text"
                                bind:value={ncEmailTo}
                                placeholder="admin@example.com, ops@example.com"
                                class="input-base"
                            />
                        </div>
                    </div>
                {:else}
                    <div class="space-y-1.5">
                        <label
                            class="text-sm font-medium text-text-muted"
                            for="nc-url">URL</label
                        >
                        <input
                            id="nc-url"
                            type="url"
                            bind:value={ncConfigUrl}
                            placeholder="https://..."
                            class="input-base"
                        />
                        {#if ncType === "webhook"}
                            <p class="text-[11px] text-text-subtle mt-1">
                                Enter the full URL where a POST request with the
                                alert JSON will be sent.
                            </p>
                        {:else if ncType === "discord"}
                            <p class="text-[11px] text-text-subtle mt-1">
                                Enter the Webhook URL from your Discord Server
                                Settings > Integrations.
                            </p>
                        {:else if ncType === "slack"}
                            <p class="text-[11px] text-text-subtle mt-1">
                                Enter the Incoming Webhook URL from your Slack
                                Workspace app.
                            </p>
                        {:else if ncType === "ntfy"}
                            <p class="text-[11px] text-text-subtle mt-1">
                                Enter the topic URL (e.g.,
                                https://ntfy.sh/my_secret_topic).
                            </p>
                        {/if}
                    </div>
                {/if}
                <label
                    class="flex items-center gap-3 cursor-pointer select-none"
                >
                    <div class="relative">
                        <input
                            type="checkbox"
                            bind:checked={ncEnabled}
                            class="sr-only peer"
                        />
                        <div
                            class="w-9 h-5 rounded-full border border-border bg-surface-elevated peer-checked:bg-primary peer-checked:border-primary transition-colors"
                        ></div>
                        <div
                            class="absolute top-0.5 left-0.5 size-4 rounded-full bg-white shadow transition-transform peer-checked:translate-x-4"
                        ></div>
                    </div>
                    <div>
                        <p class="text-sm font-medium text-text">Enabled</p>
                        <p class="text-[11px] text-text-subtle">
                            Send notifications through this channel
                        </p>
                    </div>
                </label>
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button variant="outline" onclick={() => (ncDialogOpen = false)}
                    >Cancel</Button
                >
                <Button loading={ncSaving} onclick={saveChannel}>
                    {ncSaving
                        ? "Saving..."
                        : ncEditTarget
                          ? "Save Changes"
                          : "Create Channel"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>

<!-- Invite User Dialog -->
<Dialog.Root bind:open={userDialogOpen}>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-center justify-between mb-5">
                <div>
                    <Dialog.Title class="text-base font-semibold text-text"
                        >Invite User</Dialog.Title
                    >
                    <Dialog.Description class="text-xs text-text-muted mt-0.5"
                        >Create a new user account.</Dialog.Description
                    >
                </div>
                <Dialog.Close
                    class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors"
                >
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if userError}
                <div
                    class="mb-4 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm"
                >
                    {userError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="u-name"
                        >Username <span class="text-danger">*</span></label
                    >
                    <input
                        id="u-name"
                        type="text"
                        bind:value={newUsername}
                        placeholder="johndoe"
                        class="input-base"
                    />
                </div>
                <div class="space-y-1.5">
                    <label
                        class="text-sm font-medium text-text-muted"
                        for="u-pass"
                        >Password <span class="text-danger">*</span></label
                    >
                    <input
                        id="u-pass"
                        type="password"
                        bind:value={newPassword}
                        placeholder="Minimum 8 characters"
                        class="input-base"
                    />
                </div>
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button
                    variant="outline"
                    onclick={() => (userDialogOpen = false)}>Cancel</Button
                >
                <Button loading={userSaving} onclick={inviteUser}>
                    {userSaving ? "Creating..." : "Create User"}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
