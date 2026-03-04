<script lang="ts">
    import { onMount } from "svelte";
    import { Activity, Lock, LogIn, UserPlus } from "lucide-svelte";
    import Button from "$lib/components/ui/button.svelte";
    import { fetchAPI } from "$lib/api/client";
    import { authStore } from "$lib/stores/auth.svelte";

    let username = $state("");
    let password = $state("");
    let loading = $state(false);
    let errorMsg = $state("");
    let setupRequired = $state(false);
    let oidcEnabled = $state(false);
    let checkingSetup = $state(true);

    onMount(async () => {
        try {
            const [setupRes, pRes] = await Promise.all([
                fetch("/api/v1/auth/setup"),
                fetch("/api/v1/auth/providers"),
            ]);

            if (setupRes.ok) {
                const data = await setupRes.json();
                setupRequired = data.setup_required === true;
            }
            if (pRes.ok) {
                const data = await pRes.json();
                oidcEnabled = data.oidc === true;
            }
        } catch {
            // ignore — proceed as login
        } finally {
            checkingSetup = false;
        }
    });

    async function handleSubmit(e: Event) {
        e.preventDefault();
        loading = true;
        errorMsg = "";

        try {
            if (setupRequired) {
                // Register first admin
                await fetchAPI("/api/v1/auth/register", {
                    method: "POST",
                    body: JSON.stringify({ username, password }),
                });
            }
            // Login
            await fetchAPI("/api/v1/auth/login", {
                method: "POST",
                body: JSON.stringify({ username, password }),
            });
            await authStore.init();
            window.location.href = "/";
        } catch (e: any) {
            errorMsg =
                e.message || (setupRequired ? "Setup failed" : "Login failed");
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>{setupRequired ? "Setup" : "Sign In"} – updu</title>
</svelte:head>

<div
    class="min-h-screen bg-background flex flex-col items-center justify-center p-4 relative overflow-hidden"
>
    <!-- Decorative background -->
    <div class="absolute inset-0 pointer-events-none" aria-hidden="true">
        <div
            class="absolute inset-0 opacity-[0.03]"
            style="background-image: linear-gradient(hsl(215 28% 60%) 1px, transparent 1px), linear-gradient(90deg, hsl(215 28% 60%) 1px, transparent 1px); background-size: 40px 40px;"
        ></div>
        <div
            class="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[400px] bg-primary/5 rounded-full blur-3xl"
        ></div>
    </div>

    <!-- Logo mark -->
    <div class="relative z-10 flex flex-col items-center mb-8">
        <div class="relative mb-4">
            <div
                class="size-14 rounded-2xl bg-primary/10 border border-primary/20 flex items-center justify-center shadow-[0_0_32px_hsl(217_91%_60%/0.15)]"
            >
                <Activity class="size-7 text-primary" />
            </div>
            <div
                class="absolute inset-0 rounded-2xl border border-primary/30 animate-ping opacity-30"
            ></div>
        </div>
        {#if checkingSetup}
            <h1 class="text-2xl font-bold tracking-tight text-text">updu</h1>
            <p class="text-sm text-text-muted mt-1">Loading...</p>
        {:else if setupRequired}
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Welcome to updu
            </h1>
            <p class="text-sm text-text-muted mt-1">
                Create your admin account to get started
            </p>
        {:else}
            <h1 class="text-2xl font-bold tracking-tight text-text">
                Sign in to updu
            </h1>
            <p class="text-sm text-text-muted mt-1">
                Uptime Dashboard Unlimited
            </p>
        {/if}
    </div>

    <!-- Card -->
    {#if !checkingSetup}
        <div class="relative z-10 w-full max-w-sm">
            <div
                class="bg-surface/60 backdrop-blur-2xl border border-border rounded-2xl p-8 shadow-[0_24px_64px_hsl(224_71%_4%/0.7)]"
            >
                <form onsubmit={handleSubmit} class="space-y-5">
                    {#if errorMsg}
                        <div
                            class="p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm flex items-center gap-2"
                        >
                            <svg
                                class="size-4 shrink-0"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                            >
                                <circle cx="12" cy="12" r="10" /><line
                                    x1="12"
                                    y1="8"
                                    x2="12"
                                    y2="12"
                                /><line x1="12" y1="16" x2="12.01" y2="16" />
                            </svg>
                            {errorMsg}
                        </div>
                    {/if}

                    {#if setupRequired}
                        <div
                            class="p-3 rounded-lg bg-primary/10 border border-primary/20 text-primary text-sm"
                        >
                            Choose a username and password for your admin
                            account. Min 3 chars username, 8 chars password.
                        </div>
                    {/if}

                    <div class="space-y-1.5">
                        <label
                            for="username"
                            class="text-sm font-medium text-text-muted"
                            >Username</label
                        >
                        <div class="relative">
                            <div
                                class="absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle pointer-events-none"
                            >
                                <svg
                                    class="size-4"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    ><path
                                        d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"
                                    /><circle cx="12" cy="7" r="4" /></svg
                                >
                            </div>
                            <input
                                id="username"
                                name="username"
                                type="text"
                                required
                                autofocus
                                bind:value={username}
                                placeholder={setupRequired
                                    ? "admin"
                                    : "your_username"}
                                class="input-base pl-9"
                            />
                        </div>
                    </div>

                    <div class="space-y-1.5">
                        <label
                            for="password"
                            class="text-sm font-medium text-text-muted"
                            >Password</label
                        >
                        <div class="relative">
                            <div
                                class="absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle pointer-events-none"
                            >
                                <Lock class="size-4" />
                            </div>
                            <input
                                id="password"
                                name="password"
                                type="password"
                                required
                                bind:value={password}
                                placeholder="••••••••"
                                class="input-base pl-9"
                            />
                        </div>
                    </div>

                    <Button
                        type="submit"
                        {loading}
                        class="w-full h-11 text-sm mt-2"
                    >
                        {#if setupRequired}
                            <UserPlus class="size-4" />
                            {loading
                                ? "Creating account..."
                                : "Create Admin Account"}
                        {:else}
                            <LogIn class="size-4" />
                            {loading ? "Signing in..." : "Sign in"}
                        {/if}
                    </Button>
                </form>

                {#if oidcEnabled}
                    <div class="mt-6">
                        <div class="relative">
                            <div class="absolute inset-0 flex items-center">
                                <div
                                    class="w-full border-t border-border"
                                ></div>
                            </div>
                            <div
                                class="relative flex justify-center text-xs uppercase"
                            >
                                <span class="bg-surface/60 px-2 text-text-muted"
                                    >Or continue with</span
                                >
                            </div>
                        </div>
                        <Button
                            href="/api/v1/auth/oidc/login"
                            variant="outline"
                            class="w-full mt-4 h-11 border-border/50 bg-background/50 backdrop-blur-sm hover:bg-background transition-colors"
                        >
                            <svg
                                class="size-4 mr-2"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                            >
                                <rect
                                    x="3"
                                    y="11"
                                    width="18"
                                    height="11"
                                    rx="2"
                                    ry="2"
                                />
                                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                            </svg>
                            Single Sign-On (OIDC)
                        </Button>
                    </div>
                {/if}
            </div>

            <p class="text-center text-xs text-text-subtle mt-6">
                {setupRequired
                    ? "Your first account will have admin privileges"
                    : "Uptime Dashboard Unlimited"}
            </p>
        </div>
    {/if}
</div>
