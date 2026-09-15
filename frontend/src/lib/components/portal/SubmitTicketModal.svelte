<script lang="ts">
	import Modal from "$lib/components/ui/modal.svelte";
	import Button from "$lib/components/ui/button.svelte";
	import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
	import { fetchAPI } from "$lib/api/client";
	import { authStore } from "$lib/stores/auth.svelte";
	import { Send, AlertCircle, LifeBuoy } from "lucide-svelte";

	interface Props {
		open: boolean;
		services?: Array<{ id: string; name: string }>;
		initialServiceId?: string;
		onSubmitted?: (ticket: any) => void;
	}

	let {
		open = $bindable(false),
		services = [],
		initialServiceId = "",
		onSubmitted,
	}: Props = $props();

	let selectedServiceId = $state("");
	let title = $state("");
	let severity = $state<"low" | "medium" | "high">("medium");
	let description = $state("");
	let submitting = $state(false);
	let errorMsg = $state("");

	// Reset form when opened or initialServiceId changes
	$effect(() => {
		if (open) {
			selectedServiceId = initialServiceId || "";
			title = "";
			severity = "medium";
			description = "";
			errorMsg = "";
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!title.trim()) {
			errorMsg = "Please provide an issue summary";
			return;
		}
		if (!description.trim()) {
			errorMsg = "Please describe the problem you are experiencing";
			return;
		}

		submitting = true;
		errorMsg = "";

		try {
			const payload: Record<string, any> = {
				title: title.trim(),
				description: description.trim(),
				severity,
			};
			if (selectedServiceId) {
				payload.service_id = selectedServiceId;
			}

			const ticket = await fetchAPI("/api/v1/tickets", {
				method: "POST",
				body: JSON.stringify(payload),
			});

			toastStore.success("Ticket submitted! The administrator has been notified.");
			open = false;
			onSubmitted?.(ticket);
		} catch (err) {
			errorMsg = toastFromError(err, "Failed to submit ticket");
		} finally {
			submitting = false;
		}
	}
</script>

<Modal
	bind:open
	title="Report an Outage or Issue"
	description="Submit a ticket directly to your homelab administrator. We'll investigate and post updates here."
	size="md"
>
	<form onsubmit={handleSubmit} class="space-y-4 pt-1">
		{#if errorMsg}
			<div class="flex items-center gap-2 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-xs">
				<AlertCircle class="size-4 shrink-0" />
				<span>{errorMsg}</span>
			</div>
		{/if}

		<!-- Service selector -->
		<div class="space-y-1.5">
			<label for="ticket-service" class="block text-xs font-semibold text-text">
				Affected Service
			</label>
			<select
				id="ticket-service"
				bind:value={selectedServiceId}
				class="w-full px-3 py-2 text-sm rounded-lg bg-surface border border-border text-text focus:outline-none focus:ring-1 focus:ring-primary"
			>
				<option value="">General / Homelab Network Issue</option>
				{#each services as svc (svc.id)}
					<option value={svc.id}>{svc.name}</option>
				{/each}
			</select>
		</div>

		<!-- Title -->
		<div class="space-y-1.5">
			<label for="ticket-title" class="block text-xs font-semibold text-text">
				Issue Summary <span class="text-danger">*</span>
			</label>
			<input
				id="ticket-title"
				type="text"
				bind:value={title}
				placeholder="e.g. Cannot stream video, page not loading, login failure"
				required
				class="w-full px-3 py-2 text-sm rounded-lg bg-surface border border-border text-text placeholder:text-text-muted/60 focus:outline-none focus:ring-1 focus:ring-primary"
			/>
		</div>

		<!-- Severity -->
		<div class="space-y-1.5">
			<span class="block text-xs font-semibold text-text">
				Impact / Severity
			</span>
			<div class="grid grid-cols-3 gap-2">
				<button
					type="button"
					onclick={() => (severity = "low")}
					class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center {severity === 'low'
						? 'bg-primary/10 border-primary text-primary font-bold shadow-sm'
						: 'bg-surface border-border text-text-muted hover:border-border-hover'}"
				>
					Low (Minor)
				</button>
				<button
					type="button"
					onclick={() => (severity = "medium")}
					class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center {severity === 'medium'
						? 'bg-warning/10 border-warning text-warning font-bold shadow-sm'
						: 'bg-surface border-border text-text-muted hover:border-border-hover'}"
				>
					Medium (Degraded)
				</button>
				<button
					type="button"
					onclick={() => (severity = "high")}
					class="px-3 py-2 text-xs font-medium rounded-lg border transition-all text-center {severity === 'high'
						? 'bg-danger/10 border-danger text-danger font-bold shadow-sm'
						: 'bg-surface border-border text-text-muted hover:border-border-hover'}"
				>
					High (Outage)
				</button>
			</div>
		</div>

		<!-- Description -->
		<div class="space-y-1.5">
			<label for="ticket-desc" class="block text-xs font-semibold text-text">
				Description & Details <span class="text-danger">*</span>
			</label>
			<textarea
				id="ticket-desc"
				bind:value={description}
				rows="3"
				placeholder="What happened? Which device or app were you using? Any error message?"
				required
				class="w-full px-3 py-2 text-sm rounded-lg bg-surface border border-border text-text placeholder:text-text-muted/60 focus:outline-none focus:ring-1 focus:ring-primary resize-none"
			></textarea>
		</div>

		<!-- Submitter note -->
		{#if authStore.user}
			<div class="text-[11px] text-text-subtle">
				Submitting as <strong class="text-text">{authStore.user.username}</strong>
				{#if authStore.user.groups?.length}
					<span class="ml-1 text-primary">({authStore.user.groups.join(", ")})</span>
				{/if}
			</div>
		{/if}

		<!-- Action buttons -->
		<div class="flex items-center justify-end gap-2 pt-2 border-t border-border/60">
			<Button
				variant="secondary"
				size="sm"
				type="button"
				onclick={() => (open = false)}
				disabled={submitting}
			>
				Cancel
			</Button>
			<Button
				variant="default"
				size="sm"
				type="submit"
				disabled={submitting}
				class="flex items-center gap-1.5"
			>
				<Send class="size-3.5" />
				<span>{submitting ? "Submitting..." : "Submit Ticket"}</span>
			</Button>
		</div>
	</form>
</Modal>
