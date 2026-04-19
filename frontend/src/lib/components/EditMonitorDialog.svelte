<script lang="ts">
    import MonitorForm from "$lib/components/monitors/monitor-form.svelte";

    let {
        open = $bindable(false),
        monitor = $bindable(null),
    }: { open: boolean; monitor: any } = $props();
</script>

<MonitorForm mode="edit" bind:open bind:monitor />
