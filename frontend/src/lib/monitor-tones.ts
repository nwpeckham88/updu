// Shared tone mappers for monitor-related UI. Keeps inline ternaries out of templates.
export type Tone = "neutral" | "primary" | "success" | "warning" | "danger";

export function uptimeTone(pct: number | null | undefined): Tone {
	if (pct == null || Number.isNaN(pct)) return "neutral";
	if (pct >= 99) return "success";
	if (pct >= 95) return "warning";
	return "danger";
}

export function latencyTone(ms: number | null | undefined): Tone {
	if (ms == null || Number.isNaN(ms)) return "neutral";
	if (ms > 3000) return "danger";
	if (ms > 1000) return "warning";
	return "neutral";
}

export function statusTone(status: string | null | undefined): Tone {
	switch (status) {
		case "up":
			return "success";
		case "down":
			return "danger";
		case "degraded":
		case "warning":
			return "warning";
		case "paused":
			return "neutral";
		default:
			return "neutral";
	}
}

export function statusTextClass(status: string | null | undefined): string {
	switch (status) {
		case "up":
			return "text-success";
		case "down":
			return "text-danger";
		case "degraded":
		case "warning":
			return "text-warning";
		default:
			return "text-text-subtle";
	}
}

export function uptimeTextClass(pct: number | null | undefined): string {
	const tone = uptimeTone(pct);
	if (tone === "success") return "text-success";
	if (tone === "warning") return "text-warning";
	if (tone === "danger") return "text-danger";
	return "text-text-subtle";
}

export function latencyTextClass(ms: number | null | undefined): string {
	const tone = latencyTone(ms);
	if (tone === "warning") return "text-warning";
	if (tone === "danger") return "text-danger";
	return "text-text";
}
