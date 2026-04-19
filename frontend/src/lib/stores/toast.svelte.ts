// Toast notification store. Reactive queue with auto-dismiss, pause-on-hover.
export type ToastVariant = 'success' | 'info' | 'warning' | 'error';

export interface Toast {
    id: string;
    variant: ToastVariant;
    title: string;
    description?: string;
    duration: number; // ms; 0 = sticky
    createdAt: number;
}

export interface ToastOptions {
    title?: string;
    description?: string;
    duration?: number;
}

const DEFAULT_DURATIONS: Record<ToastVariant, number> = {
    success: 4000,
    info: 5000,
    warning: 6000,
    error: 8000,
};

function createToastStore() {
    let toasts = $state<Toast[]>([]);
    const timers = new Map<string, ReturnType<typeof setTimeout>>();

    function id(): string {
        return `t_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 7)}`;
    }

    function dismiss(toastId: string) {
        const timer = timers.get(toastId);
        if (timer) {
            clearTimeout(timer);
            timers.delete(toastId);
        }
        toasts = toasts.filter((t) => t.id !== toastId);
    }

    function dismissAll() {
        for (const timer of timers.values()) clearTimeout(timer);
        timers.clear();
        toasts = [];
    }

    function push(variant: ToastVariant, titleOrOptions: string | ToastOptions, options: ToastOptions = {}): string {
        const opts: ToastOptions =
            typeof titleOrOptions === 'string'
                ? { title: titleOrOptions, ...options }
                : titleOrOptions;
        const toast: Toast = {
            id: id(),
            variant,
            title: opts.title ?? '',
            description: opts.description,
            duration: opts.duration ?? DEFAULT_DURATIONS[variant],
            createdAt: Date.now(),
        };
        toasts = [...toasts, toast];
        if (toast.duration > 0) {
            const timer = setTimeout(() => dismiss(toast.id), toast.duration);
            timers.set(toast.id, timer);
        }
        return toast.id;
    }

    function pause(toastId: string) {
        const timer = timers.get(toastId);
        if (timer) {
            clearTimeout(timer);
            timers.delete(toastId);
        }
    }

    function resume(toastId: string) {
        const toast = toasts.find((t) => t.id === toastId);
        if (!toast || toast.duration <= 0 || timers.has(toastId)) return;
        const elapsed = Date.now() - toast.createdAt;
        const remaining = Math.max(1000, toast.duration - elapsed);
        const timer = setTimeout(() => dismiss(toastId), remaining);
        timers.set(toastId, timer);
    }

    return {
        get items() {
            return toasts;
        },
        success: (title: string | ToastOptions, options?: ToastOptions) => push('success', title, options),
        info: (title: string | ToastOptions, options?: ToastOptions) => push('info', title, options),
        warning: (title: string | ToastOptions, options?: ToastOptions) => push('warning', title, options),
        error: (title: string | ToastOptions, options?: ToastOptions) => push('error', title, options),
        dismiss,
        dismissAll,
        pause,
        resume,
    };
}

export const toastStore = createToastStore();

// Convenience helper for caught errors.
export function toastFromError(err: unknown, fallback = 'Something went wrong'): string {
    const message = err instanceof Error ? err.message : fallback;
    return toastStore.error(message);
}
