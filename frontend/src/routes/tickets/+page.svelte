<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import {
		LifeBuoy,
		CheckCircle2,
		Clock,
		AlertTriangle,
		Trash2,
		RefreshCw,
		Check,
		Search,
		Filter,
		User,
	} from "lucide-svelte";
	import Button from "$lib/components/ui/button.svelte";
	import Badge from "$lib/components/ui/badge.svelte";
	import EmptyState from "$lib/components/ui/empty-state.svelte";
	import Skeleton from "$lib/components/ui/skeleton.svelte";
	import { fetchAPI } from "$lib/api/client";
	import { toastStore, toastFromError } from "$lib/stores/toast.svelte";
	import { confirmAction } from "$lib/stores/confirm.svelte";

	interface TicketItem {
		id: string;
		title: string;
		description: string;
		status: "open" | "in_progress" | "resolved" | "closed";
		severity: "low" | "medium" | "high";
		service_id?: string;
		service_name?: string;
		created_by: string;
		created_at: string;
		updated_at: string;
		resolved_at?: string;
	}

	let tickets = $state<TicketItem[]>([]);
	let loading = $state(true);
	let statusFilter = $state<string>("all");
	let searchQuery = $state("");
	let eventSource: EventSource | null = null;

	onMount(() => {
		loadTickets();
		startRealtime();
	});

	onDestroy(() => {
		stopRealtime();
	});

	async function loadTickets() {
		loading = true;
		try {
			const data = await fetchAPI<TicketItem[]>("/api/v1/tickets");
			tickets = data || [];
		} catch (e) {
			tickets = [];
			toastFromError(e, "Failed to load tickets");
		} finally {
			loading = false;
		}
	}

	function startRealtime() {
		if (typeof window === "undefined") return;
		try {
			eventSource = new EventSource("/api/v1/events");
			eventSource.addEventListener("ticket:create", () => void loadTickets());
			eventSource.addEventListener("ticket:update", () => void loadTickets());
			eventSource.addEventListener("ticket:delete", () => void loadTickets());
			eventSource.onerror = () => {
				eventSource?.close();
				eventSource = null;
			};
		} catch {
			// fallback
		}
	}

	function stopRealtime() {
		eventSource?.close();
		eventSource = null;
	}

	const filteredTickets = $derived.by(() => {
		const q = searchQuery.toLowerCase().trim();
		return tickets.filter((t) => {
			if (statusFilter !== "all" && t.status !== statusFilter) return false;
			if (!q) return true;
			return (
				t.title.toLowerCase().includes(q) ||
				t.description.toLowerCase().includes(q) ||
				t.created_by.toLowerCase().includes(q) ||
				(t.service_name && t.service_name.toLowerCase().includes(q))
			);
		});
	});

	const counts = $derived.by(() => {
		const c = { all: tickets.length, open: 0, in_progress: 0, resolved: 0, closed: 0 };
		for (const t of tickets) {
			if (t.status === "open") c.open += 1;
			else if (t.status === "in_progress") c.in_progress += 1;
			else if (t.status === "resolved") c.resolved += 1;
			else if (t.status === "closed") c.closed += 1;
		}
		return c;
	});

	async function updateTicketStatus(ticket: TicketItem, newStatus: string) {
		try {
			await fetchAPI(`/api/v1/tickets/${ticket.id}`, {
				method: "PUT",
				body: JSON.stringify({ status: newStatus }),
			});
			toastStore.success(`Ticket status updated to ${newStatus}`);
			await loadTickets();
		} catch (e) {
			toastFromError(e, "Failed to update ticket status");
		}
	}

	async function deleteTicket(ticket: TicketItem) {
		const ok = await confirmAction({
			title: "Delete Ticket?",
			description: `Are you sure you want to delete ticket "${ticket.title}" submitted by ${ticket.created_by}?`,
			confirmLabel: "Delete Ticket",
			variant: "destructive",
		});
		if (!ok) return;

		try {
			await fetchAPI(`/api/v1/tickets/${ticket.id}`, { method: "DELETE" });
			toastStore.success("Ticket deleted");
			await loadTickets();
		} catch (e) {
			toastFromError(e, "Failed to delete ticket");
		}
	}

	function formatDateTime(iso: string): string {
		try {
			return new Date(iso).toLocaleString();
		} catch {
			return iso;
		}
	}
</script>

<svelte:head>
	<title>Tickets · updu</title>
</svelte:head>

<div class="max-w-6xl mx-auto space-y-6">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-2xl font-bold tracking-tight text-text">Issue Tickets</h1>
				{#if counts.open > 0}
					<span class="px-2 py-0.5 text-xs font-bold rounded-full bg-warning/10 text-warning border border-warning/20">
						{counts.open} open
					</span>
				{/if}
			</div>
			<p class="text-sm text-text-subtle mt-0.5">
				Triage and resolve outage reports submitted by homelab users
			</p>
		</div>

		<Button variant="secondary" size="sm" onclick={loadTickets} class="flex items-center gap-1.5 self-start sm:self-auto">
			<RefreshCw class="size-3.5 {loading ? 'animate-spin' : ''}" />
			<span>Refresh</span>
		</Button>
	</div>

	<!-- Controls & Filters -->
	<div class="flex flex-col sm:flex-row items-center justify-between gap-3">
		<!-- Status tabs -->
		<div class="flex items-center gap-1 bg-surface p-1 rounded-xl border border-border w-full sm:w-auto overflow-x-auto">
			<button
				type="button"
				onclick={() => (statusFilter = "all")}
				class="px-3 py-1.5 text-xs font-medium rounded-lg transition-all whitespace-nowrap {statusFilter === 'all'
					? 'bg-primary/10 text-primary font-bold shadow-sm'
					: 'text-text-muted hover:text-text'}"
			>
				All ({counts.all})
			</button>
			<button
				type="button"
				onclick={() => (statusFilter = "open")}
				class="px-3 py-1.5 text-xs font-medium rounded-lg transition-all whitespace-nowrap {statusFilter === 'open'
					? 'bg-warning/10 text-warning font-bold shadow-sm'
					: 'text-text-muted hover:text-text'}"
			>
				Open ({counts.open})
			</button>
			<button
				type="button"
				onclick={() => (statusFilter = "in_progress")}
				class="px-3 py-1.5 text-xs font-medium rounded-lg transition-all whitespace-nowrap {statusFilter === 'in_progress'
					? 'bg-primary/10 text-primary font-bold shadow-sm'
					: 'text-text-muted hover:text-text'}"
			>
				In Progress ({counts.in_progress})
			</button>
			<button
				type="button"
				onclick={() => (statusFilter = "resolved")}
				class="px-3 py-1.5 text-xs font-medium rounded-lg transition-all whitespace-nowrap {statusFilter === 'resolved'
					? 'bg-success/10 text-success font-bold shadow-sm'
					: 'text-text-muted hover:text-text'}"
			>
				Resolved ({counts.resolved})
			</button>
		</div>

		<!-- Search -->
		<div class="relative w-full sm:w-64">
			<Search class="size-4 absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search tickets..."
				class="w-full pl-9 pr-3 py-1.5 text-xs rounded-xl bg-surface border border-border text-text placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-primary"
			/>
		</div>
	</div>

	<!-- Ticket List -->
	{#if loading}
		<div class="space-y-3">
			{#each Array(3) as _}
				<div class="p-5 rounded-xl border border-border bg-surface animate-pulse space-y-3">
					<div class="h-4 w-48 bg-surface-elevated rounded"></div>
					<div class="h-3 w-96 bg-surface-elevated rounded"></div>
				</div>
			{/each}
		</div>
	{:else if filteredTickets.length === 0}
		<div class="p-12 text-center rounded-2xl border border-border bg-surface space-y-2">
			<LifeBuoy class="size-8 text-text-muted mx-auto" />
			<p class="text-sm font-semibold text-text">No tickets found</p>
			<p class="text-xs text-text-subtle">
				{statusFilter === "open"
					? "Great job! There are no open issue tickets right now."
					: "No tickets matching your filter criteria."}
			</p>
		</div>
	{:else}
		<div class="divide-y divide-border border border-border rounded-xl bg-surface overflow-hidden">
			{#each filteredTickets as ticket (ticket.id)}
				<div class="p-5 hover:bg-surface-elevated/30 transition-colors flex flex-col md:flex-row md:items-center justify-between gap-4">
					<div class="space-y-2 min-w-0 max-w-2xl">
						<div class="flex items-center gap-2 flex-wrap">
							<span class="text-sm font-bold text-text">{ticket.title}</span>

							{#if ticket.service_name}
								<span class="px-2 py-0.5 text-[11px] font-semibold rounded-md bg-primary/10 text-primary border border-primary/20">
									{ticket.service_name}
								</span>
							{/if}

							<!-- Severity -->
							{#if ticket.severity === 'high'}
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-danger/10 text-danger border border-danger/20 uppercase">
									High Impact
								</span>
							{:else if ticket.severity === 'medium'}
								<span class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-warning/10 text-warning border border-warning/20">
									Medium
								</span>
							{:else}
								<span class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-surface-elevated text-text-subtle border border-border">
									Low
								</span>
							{/if}

							<!-- Status -->
							{#if ticket.status === 'open'}
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-warning/10 text-warning border border-warning/30">
									Open
								</span>
							{:else if ticket.status === 'in_progress'}
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-primary/10 text-primary border border-primary/30">
									In Progress
								</span>
							{:else if ticket.status === 'resolved'}
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-success/10 text-success border border-success/30">
									Resolved
								</span>
							{:else}
								<span class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-surface-elevated text-text-subtle border border-border">
									Closed
								</span>
							{/if}
						</div>

						<p class="text-xs text-text-muted leading-relaxed whitespace-pre-line">
							{ticket.description}
						</p>

						<div class="flex items-center gap-3 text-[11px] text-text-subtle">
							<span class="flex items-center gap-1 font-medium text-text">
								<User class="size-3 text-text-subtle" />
								{ticket.created_by}
							</span>
							<span>·</span>
							<span>Opened {formatDateTime(ticket.created_at)}</span>
							{#if ticket.resolved_at}
								<span>·</span>
								<span class="text-success font-medium">Resolved {formatDateTime(ticket.resolved_at)}</span>
							{/if}
						</div>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2 shrink-0 self-end md:self-center">
						{#if ticket.status === 'open'}
							<Button
								variant="secondary"
								size="sm"
								onclick={() => updateTicketStatus(ticket, "in_progress")}
								class="text-xs"
							>
								Start Triage
							</Button>
						{/if}

						{#if ticket.status !== 'resolved' && ticket.status !== 'closed'}
							<Button
								variant="default"
								size="sm"
								onclick={() => updateTicketStatus(ticket, "resolved")}
								class="text-xs flex items-center gap-1 bg-success hover:bg-success/90 text-white"
							>
								<Check class="size-3.5" />
								<span>Resolve</span>
							</Button>
						{:else}
							<Button
								variant="secondary"
								size="sm"
								onclick={() => updateTicketStatus(ticket, "open")}
								class="text-xs"
							>
								Reopen
							</Button>
						{/if}

						<Button
							variant="ghost"
							size="sm"
							onclick={() => deleteTicket(ticket)}
							class="text-text-muted hover:text-danger hover:bg-danger/10"
							aria-label="Delete ticket"
						>
							<Trash2 class="size-3.5" />
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
