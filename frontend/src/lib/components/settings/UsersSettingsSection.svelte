<script lang="ts">
    import { onMount } from 'svelte';
    import { Dialog } from 'bits-ui';
    import { Eye, Plus, Shield, Trash2, Users, X } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import ConfirmActionDialog from '$lib/components/settings/ConfirmActionDialog.svelte';
    import EmptyState from '$lib/components/ui/empty-state.svelte';
    import Skeleton from '$lib/components/ui/skeleton.svelte';
    import {
        changeUserRole,
        createUser,
        deleteUser,
        listUsers,
        type AdminUser,
        type UserRole,
    } from '$lib/api/settings';
    import { authStore } from '$lib/stores/auth.svelte';

    let users = $state<AdminUser[]>([]);
    let usersLoading = $state(true);
    let usersMsg = $state('');

    let dialogOpen = $state(false);
    let newUsername = $state('');
    let newPassword = $state('');
    let userSaving = $state(false);
    let userError = $state('');
    let userActionID = $state<string | null>(null);
    let deleteDialogOpen = $state(false);
    let deleteTarget = $state<AdminUser | null>(null);
    let roleDialogOpen = $state(false);
    let pendingRoleChange = $state<
        | {
              user: AdminUser;
              role: UserRole;
          }
        | null
    >(null);

    function scheduleUsersMessageClear() {
        setTimeout(() => (usersMsg = ''), 3000);
    }

    function adminCount(): number {
        return users.filter((user) => user.role === 'admin').length;
    }

    function viewerCount(): number {
        return users.filter((user) => user.role !== 'admin').length;
    }

    function openRoleDialog(user: AdminUser, role: UserRole) {
        if (user.role === role) {
            return;
        }

        pendingRoleChange = { user, role };
        roleDialogOpen = true;
        usersMsg = '';
    }

    async function loadUsers() {
        try {
            usersLoading = true;
            users = (await listUsers()) || [];
        } catch {
            users = [];
        } finally {
            usersLoading = false;
        }
    }

    async function updateRole() {
        if (!pendingRoleChange) {
            return;
        }

        const { user, role } = pendingRoleChange;
        roleDialogOpen = false;
        userActionID = user.id;

        try {
            await changeUserRole(user.id, role);
            usersMsg = 'User role updated successfully.';
            pendingRoleChange = null;
            await loadUsers();
            scheduleUsersMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Unknown error';
            usersMsg = `Error: ${message}`;
        } finally {
            userActionID = null;
        }
    }

    function openDeleteDialog(user: AdminUser) {
        deleteTarget = user;
        deleteDialogOpen = true;
        usersMsg = '';
    }

    async function removeUser() {
        if (!deleteTarget) {
            return;
        }

        const user = deleteTarget;
        deleteDialogOpen = false;
        userActionID = user.id;

        try {
            await deleteUser(user.id);
            usersMsg = 'User deleted successfully.';
            deleteTarget = null;
            await loadUsers();
            scheduleUsersMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Unknown error';
            usersMsg = `Error: ${message}`;
        } finally {
            userActionID = null;
        }
    }

    function openInviteUser() {
        newUsername = '';
        newPassword = '';
        userError = '';
        dialogOpen = true;
    }

    async function inviteUser() {
        if (!newUsername.trim() || !newPassword.trim()) {
            userError = 'Username and password are required';
            return;
        }

        if (newUsername.trim().length < 3) {
            userError = 'Username must be at least 3 characters';
            return;
        }

        userSaving = true;
        userError = '';

        try {
            await createUser(newUsername.trim(), newPassword);
            dialogOpen = false;
            usersMsg = 'User created successfully.';
            await loadUsers();
            scheduleUsersMessageClear();
        } catch (error) {
            userError =
                error instanceof Error ? error.message : 'Failed to create user';
        } finally {
            userSaving = false;
        }
    }

    onMount(() => {
        void loadUsers();
    });
</script>

<div class="settings-stack">
    {#if usersMsg}
        <div
            class={[
                'settings-banner',
                usersMsg.startsWith('Error')
                    ? 'settings-banner-danger'
                    : 'settings-banner-success',
            ]}
            aria-live="polite"
        >
            {usersMsg}
        </div>
    {/if}

    <section class="card settings-section">
        <div class="settings-section-header-split">
            <div class="settings-section-header">
                <div class="settings-section-icon">
                    <Users class="size-4 text-primary" />
                </div>
                <div>
                    <h2 class="text-base font-semibold text-text">Users & Roles</h2>
                    <p class="text-[11px] text-text-subtle mt-0.5 max-w-2xl">
                        Manage local operators and keep access changes explicit. Admins can configure the instance while viewers stay read-only.
                    </p>
                    {#if !usersLoading}
                        <div class="settings-meta-row">
                            <span class="settings-pill settings-pill-primary">
                                {users.length} total
                            </span>
                            <span class="settings-pill settings-pill-muted">
                                {adminCount()} admins
                            </span>
                            <span class="settings-pill settings-pill-muted">
                                {viewerCount()} viewers
                            </span>
                        </div>
                    {/if}
                </div>
            </div>

            <Button class="settings-header-action" onclick={openInviteUser}>
                <Plus class="size-4" />
                Invite User
            </Button>
        </div>

        {#if usersLoading}
            <div class="space-y-3" aria-busy="true" aria-label="Loading users">
                {#each Array.from({ length: 3 }) as _, index (index)}
                    <div class="settings-skeleton-item flex gap-4">
                        <Skeleton height="h-9" width="w-9" rounded="rounded-full" />
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
                description="User accounts will appear here once they are created."
            />
        {:else}
            <div class="space-y-3">
                {#each users as user (user.id)}
                    <article
                        data-testid="user-row"
                        class="settings-list-item"
                    >
                        <div class="flex items-center gap-4 min-w-0">
                            <div class="size-9 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                                <span class="text-sm font-bold text-primary uppercase">
                                    {user.username?.[0] || '?'}
                                </span>
                            </div>
                            <div class="min-w-0">
                                <div class="flex flex-wrap items-center gap-2">
                                    <h2 class="font-semibold text-text text-sm">{user.username}</h2>
                                    {#if user.id === authStore.user?.id}
                                        <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2 py-0.5 text-[10px] font-medium text-text-muted">
                                            You
                                        </span>
                                    {/if}
                                </div>
                                <div class="flex items-center gap-1.5 mt-0.5 text-[11px] text-text-subtle">
                                    {#if user.role === 'admin'}
                                        <Shield class="size-3 text-primary" />
                                        <span class="text-primary font-medium">Admin</span>
                                    {:else}
                                        <Eye class="size-3" />
                                        <span>Viewer</span>
                                    {/if}
                                </div>
                            </div>
                        </div>

                        {#if user.id !== authStore.user?.id}
                            <div class="flex flex-wrap items-center gap-2 xl:justify-end">
                                <Button
                                    size="sm"
                                    variant="outline"
                                    onclick={() =>
                                        openRoleDialog(
                                            user,
                                            user.role === 'admin' ? 'viewer' : 'admin',
                                        )}
                                    loading={userActionID === user.id && Boolean(pendingRoleChange)}
                                >
                                    {user.role === 'admin' ? 'Make Viewer' : 'Make Admin'}
                                </Button>
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    class="text-danger hover:bg-danger/10 hover:text-danger"
                                    onclick={() => openDeleteDialog(user)}
                                    loading={userActionID === user.id && Boolean(deleteTarget)}
                                >
                                    <Trash2 class="size-3.5" />
                                    Delete
                                </Button>
                            </div>
                        {/if}
                    </article>
                {/each}
            </div>
        {/if}
    </section>

    <ConfirmActionDialog
        bind:open={roleDialogOpen}
        title="Change User Role"
        description="This updates the permissions for the selected account immediately."
        confirmLabel="Apply Role Change"
        confirmVariant="default"
        loading={Boolean(pendingRoleChange && userActionID === pendingRoleChange.user.id)}
        onConfirm={updateRole}
    >
        {#if pendingRoleChange}
            <div class="grid gap-3 text-sm sm:grid-cols-2">
                <div>
                    <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                        User
                    </p>
                    <p class="mt-1 text-text">{pendingRoleChange.user.username}</p>
                </div>
                <div>
                    <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                        New Role
                    </p>
                    <p class="mt-1 text-text capitalize">{pendingRoleChange.role}</p>
                </div>
            </div>
        {/if}
    </ConfirmActionDialog>

    <ConfirmActionDialog
        bind:open={deleteDialogOpen}
        title="Delete User"
        description="This permanently removes the account from the local user list."
        confirmLabel="Delete User"
        loading={Boolean(deleteTarget && userActionID === deleteTarget.id)}
        onConfirm={removeUser}
    >
        {#if deleteTarget}
            <div class="space-y-1 text-sm">
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Account
                </p>
                <p class="text-text">{deleteTarget.username}</p>
                <p class="text-xs text-text-muted">
                    Remove access before deleting if this account is still shared with automation or operators.
                </p>
            </div>
        {/if}
    </ConfirmActionDialog>
</div>

<Dialog.Root bind:open={dialogOpen}>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in"
        />
        <Dialog.Content
            class="fixed left-1/2 top-1/2 z-50 w-full max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface/95 backdrop-blur-2xl p-6 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:zoom-out-95 data-[state=open]:fade-in data-[state=open]:zoom-in-95"
        >
            <div class="flex items-center justify-between mb-5">
                <div>
                    <Dialog.Title class="text-base font-semibold text-text">Invite User</Dialog.Title>
                    <Dialog.Description class="text-xs text-text-muted mt-0.5">
                        Create a new local user account.
                    </Dialog.Description>
                </div>
                <Dialog.Close class="size-7 inline-flex items-center justify-center rounded-lg hover:bg-surface-elevated text-text-muted hover:text-text transition-colors">
                    <X class="size-4" />
                </Dialog.Close>
            </div>

            {#if userError}
                <div class="settings-banner settings-banner-danger mb-4">
                    {userError}
                </div>
            {/if}

            <div class="space-y-4">
                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="invite-username">
                        Username <span class="text-danger">*</span>
                    </label>
                    <input
                        id="invite-username"
                        type="text"
                        bind:value={newUsername}
                        placeholder="johndoe"
                        class="input-base"
                    />
                </div>
                <div class="space-y-1.5">
                    <label class="text-sm font-medium text-text-muted" for="invite-password">
                        Password <span class="text-danger">*</span>
                    </label>
                    <input
                        id="invite-password"
                        type="password"
                        bind:value={newPassword}
                        placeholder="Must satisfy the server policy"
                        class="input-base"
                    />
                </div>
            </div>

            <div class="flex gap-2 justify-end mt-6">
                <Button variant="outline" onclick={() => (dialogOpen = false)}>Cancel</Button>
                <Button loading={userSaving} onclick={inviteUser}>
                    {userSaving ? 'Creating...' : 'Create User'}
                </Button>
            </div>
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>