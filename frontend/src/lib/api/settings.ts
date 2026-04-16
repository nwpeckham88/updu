import { fetchAPI } from '$lib/api/client';

export type SettingsMap = Record<string, string>;

export const GENERAL_SETTINGS_KEYS = [
    'site_name',
    'site_description',
    'base_url',
    'custom_css',
    'enable_custom_css',
    'logo_url',
    'favicon_url',
    'theme',
    'timezone',
    'date_format',
    'enable_public',
    'maintenance_mode',
    'notify_on_down',
    'notify_on_up',
] as const;

export type GeneralSettingKey = (typeof GENERAL_SETTINGS_KEYS)[number];

export interface NotificationChannelConfig {
    url?: string;
    host?: string;
    port?: number;
    user?: string;
    pass?: string;
    from?: string;
    to?: string;
    [key: string]: unknown;
}

export interface NotificationChannel {
    id: string;
    name: string;
    type: string;
    config?: NotificationChannelConfig;
    enabled: boolean;
}

export type UserRole = 'admin' | 'viewer';

export interface AdminUser {
    id: string;
    username: string;
    role: UserRole;
    created_at?: string;
}

export interface UpdateInfo {
    current_version: string;
    latest_version: string;
    update_available: boolean;
    release_url?: string;
    asset_url?: string;
    asset_name?: string;
    release_notes?: string;
    published_at?: string;
}

export interface UpdateActionResponse {
    message?: string;
    version?: string;
    new_version?: string;
    restart?: string;
}

export interface NotificationChannelInput {
    name: string;
    type: string;
    enabled: boolean;
    config: NotificationChannelConfig;
}

export function getSettings() {
    return fetchAPI<SettingsMap>('/api/v1/settings');
}

export function updateSettings(settings: SettingsMap) {
    return fetchAPI<{ message: string }>('/api/v1/settings', {
        method: 'POST',
        body: JSON.stringify(settings),
    });
}

export function listNotificationChannels() {
    return fetchAPI<NotificationChannel[]>('/api/v1/notifications');
}

export function createNotificationChannel(input: NotificationChannelInput) {
    return fetchAPI<NotificationChannel>('/api/v1/notifications', {
        method: 'POST',
        body: JSON.stringify(input),
    });
}

export function updateNotificationChannel(
    id: string,
    input: NotificationChannelInput,
) {
    return fetchAPI<NotificationChannel>(`/api/v1/notifications/${id}`, {
        method: 'PUT',
        body: JSON.stringify(input),
    });
}

export function deleteNotificationChannel(id: string) {
    return fetchAPI<{ message: string }>(`/api/v1/notifications/${id}`, {
        method: 'DELETE',
    });
}

export function sendNotificationChannelTest(id: string) {
    return fetchAPI<{ message?: string }>(`/api/v1/notifications/${id}/test`, {
        method: 'POST',
    });
}

export function listUsers() {
    return fetchAPI<AdminUser[]>('/api/v1/admin/users');
}

export function changeUserRole(userId: string, role: UserRole) {
    return fetchAPI<{ message: string }>(`/api/v1/admin/users/${userId}/role`, {
        method: 'PUT',
        body: JSON.stringify({ role }),
    });
}

export function deleteUser(userId: string) {
    return fetchAPI<{ message: string }>(`/api/v1/admin/users/${userId}`, {
        method: 'DELETE',
    });
}

export function createUser(username: string, password: string) {
    return fetchAPI<{ message?: string }>('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
    });
}

export function changePassword(currentPassword: string, newPassword: string) {
    return fetchAPI<{ message?: string }>('/api/v1/auth/password', {
        method: 'PUT',
        body: JSON.stringify({
            current_password: currentPassword,
            new_password: newPassword,
        }),
    });
}

export function checkForUpdates() {
    return fetchAPI<UpdateInfo>('/api/v1/system/version');
}

export function applySystemUpdate() {
    return fetchAPI<UpdateActionResponse>('/api/v1/system/update', {
        method: 'POST',
    });
}

export function exportBackupJSON() {
    return fetchAPI<unknown>('/api/v1/system/backup');
}

export async function exportBackupYAML() {
    const response = await fetch('/api/v1/system/export/yaml', {
        credentials: 'same-origin',
    });

    if (!response.ok) {
        let message = 'Export failed';

        try {
            const payload = await response.json();
            message = payload.error || message;
        } catch {
            // fall back to the default message
        }

        throw new Error(message);
    }

    return response.blob();
}

export function importBackupJSON(content: string) {
    return fetchAPI<{ message: string; errors: number }>('/api/v1/system/backup', {
        method: 'POST',
        body: content,
    });
}