<script lang="ts">
    import { goto } from "$app/navigation";
    import { onMount } from "svelte";

    onMount(() => {
        void goto("/settings/general", { replaceState: true });
    });
</script>

<svelte:head>
    <title>Settings – updu</title>
</svelte:head>

<div class="card settings-redirect-card">
    <div>
        <h2 class="text-base font-semibold text-text">Redirecting to General Settings</h2>
        <p class="text-sm text-text-muted mt-1">
            The settings area now uses dedicated routes for each section.
        </p>
    </div>
    <a href="/settings/general" class="text-sm font-medium text-primary hover:underline">
        Continue
    </a>
</div>
