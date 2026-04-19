// Imperative confirmation dialog: replaces window.confirm().
export interface ConfirmOptions {
    title: string;
    description: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: 'default' | 'destructive';
}

interface PendingConfirm extends ConfirmOptions {
    id: string;
    resolve: (value: boolean) => void;
}

function createConfirmStore() {
    let pending = $state<PendingConfirm | null>(null);
    let loading = $state(false);

    function confirm(options: ConfirmOptions): Promise<boolean> {
        // Resolve any pending dialog as cancelled before opening a new one.
        pending?.resolve(false);
        loading = false;
        return new Promise<boolean>((resolve) => {
            pending = {
                id: `c_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 6)}`,
                ...options,
                resolve,
            };
        });
    }

    function setLoading(value: boolean) {
        loading = value;
    }

    function respond(answer: boolean) {
        const current = pending;
        if (!current) return;
        current.resolve(answer);
        pending = null;
        loading = false;
    }

    return {
        get current() {
            return pending;
        },
        get loading() {
            return loading;
        },
        confirm,
        setLoading,
        accept: () => respond(true),
        cancel: () => respond(false),
    };
}

export const confirmStore = createConfirmStore();

// Convenience helper.
export function confirmAction(options: ConfirmOptions): Promise<boolean> {
    return confirmStore.confirm(options);
}
