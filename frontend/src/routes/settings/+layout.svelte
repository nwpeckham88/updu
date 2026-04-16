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

<div class="max-w-6xl mx-auto w-full pb-10 space-y-8">
    <header class="space-y-6">
        <div>
            <h1 class="text-2xl font-bold tracking-tight text-text mb-1">Settings</h1>
            <p class="text-sm text-text-muted max-w-2xl">
                Configure this updu instance by section. Each area now has its own URL, so you can deep link directly to the workflow you need.
            </p>
        </div>

        <nav aria-label="Settings sections">
            <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                {#each items as item}
                    {@const active = isActive(item.href)}
                    <a
                        href={item.href}
                        aria-current={active ? 'page' : undefined}
                        class={`rounded-2xl border p-4 transition-all duration-150 group ${active ? 'border-primary/30 bg-primary/8 text-primary' : 'border-border/60 bg-surface/40 text-text hover:border-primary/25 hover:bg-surface-elevated'}`}
                    >
                        <div class="flex items-start gap-3">
                            <div
                                class={`size-9 rounded-xl flex items-center justify-center shrink-0 ${active ? 'bg-primary/15 text-primary' : 'bg-surface text-text-muted'}`}
                            >
                                <item.icon class="size-4" />
                            </div>
                            <div class="min-w-0">
                                <p class="text-sm font-semibold leading-none">{item.label}</p>
                                <p class={`text-[11px] mt-2 leading-5 ${active ? 'text-primary/80' : 'text-text-subtle'}`}>
                                    {item.description}
                                </p>
                            </div>
                        </div>
                    </a>
                {/each}
            </div>
        </nav>
    </header>

    <main class="space-y-6 min-w-0">
        {@render children()}
    </main>
</div>