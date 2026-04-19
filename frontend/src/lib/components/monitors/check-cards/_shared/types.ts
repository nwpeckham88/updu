import type {
    MonitorCheckDescription,
    MonitorCheckResult,
    MonitorConfigTarget,
} from "$lib/monitor-config";

export interface CheckCardProps {
    monitor: MonitorConfigTarget;
    latestCheck: MonitorCheckResult | null;
    description: MonitorCheckDescription;
}

export type { MonitorCheckDescription, MonitorCheckResult, MonitorConfigTarget };
