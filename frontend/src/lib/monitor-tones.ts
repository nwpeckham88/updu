// Shared tone mappers for monitor-related UI. Keeps inline ternaries out of templates.
import {
	Check,
	X,
	AlertTriangle,
	Pause,
	Circle,
	type Icon as LucideIconType,
} from "lucide-svelte";

export type Tone = "neutral" | "primary" | "success" | "warning" | "danger";

export interface StatusSample {
	status?: string | null;
	checked_at?: string | null;
}

export const FLAPPING_WINDOW_MS = 10 * 60 * 1000;
export const FLAPPING_CHANGE_THRESHOLD = 3;

// Shape pattern for status — used as a secondary, non-color encoding so colorblind
// users can still distinguish bucket states. Values map to a CSS background pattern
// and a short label suitable for aria text.
export type StatusPattern = "solid" | "diagonal" | "dotted" | "empty" | "muted";

export type StatusGlyph = typeof LucideIconType;

export function statusIcon(status: string | null | undefined): StatusGlyph {
	switch (status) {
		case "up":
			return Check;
		case "down":
			return X;
		case "degraded":
		case "warning":
			return AlertTriangle;
		case "paused":
			return Pause;
		default:
			return Circle;
	}
}

export function statusPattern(status: string | null | undefined): StatusPattern {
	switch (status) {
		case "up":
			return "solid";
		case "down":
			return "diagonal";
		case "degraded":
		case "warning":
			return "dotted";
		case "paused":
			return "muted";
		default:
			return "empty";
	}
}

export function statusLabel(status: string | null | undefined): string {
	switch (status) {
		case "up":
			return "Operational";
		case "down":
			return "Down";
		case "degraded":
		case "warning":
			return "Degraded";
		case "paused":
			return "Paused";
		case "pending":
			return "Pending";
		default:
			return "Unknown";
	}
}

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

export function statusChangeCount(
	samples: StatusSample[] | null | undefined,
	now = Date.now(),
	windowMs = FLAPPING_WINDOW_MS,
): number {
	if (!samples || samples.length < 2) return 0;

	const cutoff = now - windowMs;
	const windowed = samples
		.map((sample) => {
			const checkedAt = sample.checked_at
				? new Date(sample.checked_at).getTime()
				: Number.NaN;
			return { status: sample.status, checkedAt };
		})
		.filter(
			(sample) =>
				sample.status &&
				Number.isFinite(sample.checkedAt) &&
				sample.checkedAt >= cutoff &&
				sample.checkedAt <= now,
		)
		.sort((a, b) => a.checkedAt - b.checkedAt);

	let changes = 0;
	let previous = windowed[0]?.status;
	for (const sample of windowed.slice(1)) {
		if (sample.status !== previous) {
			changes += 1;
			previous = sample.status;
		}
	}
	return changes;
}

export function isFlapping(
	samples: StatusSample[] | null | undefined,
	now = Date.now(),
): boolean {
	return statusChangeCount(samples, now) >= FLAPPING_CHANGE_THRESHOLD;
}
