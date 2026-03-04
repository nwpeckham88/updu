import { fetchAPI } from "$lib/api/client";

function createSettingsStore() {
    let settings = $state<Record<string, string>>({});
    let initialized = $state(false);
    let error = $state<string | null>(null);

    return {
        get settings() {
            return settings;
        },
        get initialized() {
            return initialized;
        },
        get error() {
            return error;
        },
        async init() {
            if (initialized) return;
            try {
                const data = await fetchAPI("/api/v1/settings");
                settings = data || {};
                error = null;
            } catch (e: any) {
                // Ignore auth errors, as settings might only be available to admins or we just use default settings
                error = e.message || "Failed to load settings";
                settings = {}; // fallback to empty
            } finally {
                initialized = true;
            }
        },
        async refresh() {
            try {
                const data = await fetchAPI("/api/v1/settings");
                settings = data || {};
                error = null;
            } catch (e: any) {
                error = e.message || "Failed to load settings";
            }
        },
        // Helper getter for specific settings
        get(key: string, defaultValue: string = ""): string {
            return settings[key] ?? defaultValue;
        }
    };
}

export const settingsStore = createSettingsStore();
