<script lang="ts">
    import { Download, HardDrive, Upload } from 'lucide-svelte';
    import Button from '$lib/components/ui/button.svelte';
    import ConfirmActionDialog from '$lib/components/settings/ConfirmActionDialog.svelte';
    import {
        exportBackupJSON,
        exportBackupYAML,
        importBackupJSON,
    } from '$lib/api/settings';

    let importing = $state(false);
    let backupMsg = $state('');
    let importDialogOpen = $state(false);
    let selectedFiles = $state<FileList | undefined>();
    let stagedImportFile = $state<File | null>(null);

    function scheduleBackupMessageClear(delay = 3000) {
        setTimeout(() => (backupMsg = ''), delay);
    }

    function formatBytes(bytes: number): string {
        if (bytes < 1024) {
            return `${bytes} B`;
        }

        if (bytes < 1024 * 1024) {
            return `${(bytes / 1024).toFixed(1)} KB`;
        }

        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }

    function clearImportSelection() {
        stagedImportFile = null;

        if (typeof DataTransfer !== 'undefined') {
            selectedFiles = new DataTransfer().files;
            return;
        }

        selectedFiles = undefined;
    }

    function stageImportSelection() {
        const file = selectedFiles?.[0];

        if (!file) {
            clearImportSelection();
            return;
        }

        stagedImportFile = file;
        importDialogOpen = true;
        backupMsg = '';
    }

    async function exportBackup() {
        try {
            const payload = await exportBackupJSON();
            const blob = new Blob([JSON.stringify(payload, null, 2)], {
                type: 'application/json',
            });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');

            link.href = url;
            link.download = 'updu-backup.json';
            link.click();

            URL.revokeObjectURL(url);
            backupMsg = 'Backup exported successfully.';
            scheduleBackupMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Export failed';
            backupMsg = `Error: ${message}`;
        }
    }

    async function exportYAML() {
        try {
            const blob = await exportBackupYAML();
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');

            link.href = url;
            link.download = 'exported.updu.conf';
            link.click();

            URL.revokeObjectURL(url);
            backupMsg = 'Configuration exported as updu.conf successfully.';
            scheduleBackupMessageClear();
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Export failed';
            backupMsg = `Error: ${message}`;
        }
    }

    async function importBackup() {
        if (!stagedImportFile) {
            return;
        }

        const file = stagedImportFile;
        importDialogOpen = false;

        importing = true;
        backupMsg = '';

        try {
            const text = await file.text();
            const response = await importBackupJSON(text);
            backupMsg =
                response.errors > 0
                    ? `Configuration imported with ${response.errors} skipped records.`
                    : 'Configuration imported successfully.';
            scheduleBackupMessageClear(4000);
        } catch (error) {
            const message =
                error instanceof Error ? error.message : 'Import failed';
            backupMsg = `Error: ${message}`;
        } finally {
            importing = false;
            clearImportSelection();
        }
    }
</script>

<section class="card space-y-6">
    <div class="flex items-start gap-3">
        <div class="size-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
            <HardDrive class="size-4 text-primary" />
        </div>
        <div>
            <h2 class="text-base font-semibold text-text">Backups & Export</h2>
            <p class="text-[11px] text-text-subtle mt-0.5">
                Download configuration snapshots or import an existing backup into this instance.
            </p>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
                <span class="inline-flex items-center rounded-full border border-primary/20 bg-primary/8 px-2.5 py-1 font-semibold text-primary">
                    JSON snapshots
                </span>
                <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2.5 py-1 text-text-muted">
                    GitOps export
                </span>
                <span class="inline-flex items-center rounded-full border border-border/60 bg-surface/40 px-2.5 py-1 text-text-muted">
                    Reviewed imports
                </span>
            </div>
        </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-2">
        <div class="rounded-2xl border border-border/60 p-5 space-y-4">
            <div>
                <h3 class="text-sm font-semibold text-text">Export JSON Backup</h3>
                <p class="text-xs text-text-muted mt-1">
                    Download monitors, incidents, maintenance windows, notification channels, and settings as a JSON file.
                </p>
            </div>
            <Button onclick={exportBackup} variant="outline">
                <Download class="size-4" />
                Export Backup
            </Button>
        </div>

        <div class="rounded-2xl border border-border/60 p-5 space-y-4">
            <div>
                <h3 class="text-sm font-semibold text-text">Export updu.conf</h3>
                <p class="text-xs text-text-muted mt-1">
                    Generate a YAML configuration file for GitOps workflows or manual deployment.
                </p>
            </div>
            <Button onclick={exportYAML} variant="outline">
                <Download class="size-4" />
                Export updu.conf
            </Button>
        </div>
    </div>

    <div class="rounded-2xl border border-border/60 p-5 space-y-4">
        <div>
            <h3 class="text-sm font-semibold text-text">Import Configuration</h3>
            <p class="text-xs text-text-muted mt-1">
                Upload a previously exported JSON backup. Existing data is merged; invalid records are reported after import.
            </p>
        </div>

        <label class="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-transparent hover:bg-surface text-text text-sm font-medium tracking-wide cursor-pointer transition-all duration-150">
            {#if importing}
                <svg class="size-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Importing...
            {:else}
                <Upload class="size-4" />
                Choose JSON File
            {/if}
            <input
                type="file"
                accept=".json"
                class="sr-only"
                bind:files={selectedFiles}
                onchange={stageImportSelection}
                disabled={importing}
            />
        </label>
        <p class="text-[11px] text-text-subtle">
            Selecting a file opens a short review step before anything is imported.
        </p>
    </div>

    {#if backupMsg}
        <div
            class={`p-3 rounded-lg text-sm border ${backupMsg.startsWith('Error') ? 'bg-danger/10 border-danger/20 text-danger' : 'bg-success/10 border-success/20 text-success'}`}
            aria-live="polite"
        >
            {backupMsg}
        </div>
    {/if}
</section>

<ConfirmActionDialog
    bind:open={importDialogOpen}
    title="Import Configuration Backup"
    description="The uploaded JSON backup will be merged into the current instance. Invalid records are skipped and reported after import."
    confirmLabel="Import Backup"
    confirmVariant="default"
    loading={importing}
    onConfirm={importBackup}
    onCancel={clearImportSelection}
>
    {#if stagedImportFile}
        <div class="grid gap-3 text-sm sm:grid-cols-2">
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    File
                </p>
                <p class="mt-1 text-text break-all">{stagedImportFile.name}</p>
            </div>
            <div>
                <p class="text-[10px] uppercase tracking-[0.18em] text-text-subtle font-bold">
                    Size
                </p>
                <p class="mt-1 text-text">{formatBytes(stagedImportFile.size)}</p>
            </div>
        </div>
    {/if}
</ConfirmActionDialog>