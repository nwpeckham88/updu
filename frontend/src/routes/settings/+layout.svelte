<script lang="ts">
    import { page } from '$app/stores';
    import {
        Bell,
        HardDrive,
        Settings,
        Shield,
        Users,
    } from 'lucide-svelte';
    import type { Icon } from 'lucide-svelte';

    let { children } = $props();

    type SettingsNavItem = {
        href: string;
        label: string;
        description: string;
        icon: typeof Icon;
    };

    const items: SettingsNavItem[] = [
        {
            href: '/settings/general',
            label: 'General',
            description: 'Identity, appearance, and access defaults',
            icon: Settings,
        },
        {
            href: '/settings/notifications',
            label: 'Notifications',
            description: 'Alert destinations and delivery channels',
            icon: Bell,
        },
        {
            href: '/settings/users',
            label: 'Users',
            description: 'Local user accounts and roles',
            icon: Users,
        },
        {
            href: '/settings/backup',
            label: 'Backup',
            description: 'Import and export configuration snapshots',
            icon: HardDrive,
        },
        {
            href: '/settings/system',
            label: 'System',
            description: 'Updates, API tokens, and audit history',
            icon: Shield,
        },
    ];

    function isActive(href: string) {
        return $page.url.pathname === href;
    }
</script>

<div class="settings-shell">
    <header class="settings-page-header">
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text mb-1">Settings</h1>
            <p class="text-sm text-text-muted max-w-2xl">
                Configure this updu instance by section. Each area now has its own URL, so you can deep link directly to the workflow you need.
            </p>
        </div>

        <nav aria-label="Settings sections">
            <div class="settings-nav-grid">
                {#each items as item (item.href)}
                    {@const active = isActive(item.href)}
                    <a
                        href={item.href}
                        aria-current={active ? 'page' : undefined}
                        class={[
                            'settings-nav-card',
                            active && 'settings-nav-card-active',
                        ]}
                    >
                        <div class="flex items-start gap-3">
                            <div class={[
                                'settings-nav-icon',
                                active && 'settings-nav-icon-active',
                            ]}>
                                <item.icon class="size-4" />
                            </div>
                            <div class="min-w-0">
                                <p class="text-sm font-semibold leading-none">{item.label}</p>
                                <p class={[
                                    'settings-nav-description',
                                    active && 'settings-nav-description-active',
                                ]}>
                                    {item.description}
                                </p>
                            </div>
                        </div>
                    </a>
                {/each}
            </div>
        </nav>
    </header>

    <main class="settings-content">
        {@render children()}
    </main>
</div>