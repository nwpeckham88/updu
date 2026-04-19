<script lang="ts">
    import MonitorForm from "$lib/components/monitors/monitor-form.svelte";

    let { open = $bindable(false) }: { open: boolean } = $props();
</script>

<MonitorForm mode="create" bind:open />
