<script lang="ts">
    import {
        describeMonitorCheck,
        type MonitorCheckResult,
        type MonitorConfigTarget,
    } from "$lib/monitor-config";
    import { cardFor } from "./monitors/check-cards";

    let {
        monitor,
        latestCheck = null,
    }: {
        monitor: MonitorConfigTarget;
        latestCheck?: MonitorCheckResult | null;
    } = $props();

    const description = $derived(describeMonitorCheck(monitor, latestCheck));
    const Card = $derived(cardFor(monitor.type));
</script>

<Card {monitor} {latestCheck} {description} />
