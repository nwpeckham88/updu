import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

export function afterNextPaint(callback: () => void): () => void {
    const schedule =
        typeof globalThis.requestAnimationFrame === 'function'
            ? globalThis.requestAnimationFrame.bind(globalThis)
            : (fn: FrameRequestCallback) =>
                    globalThis.setTimeout(() => fn(Date.now()), 16);
    const cancel =
        typeof globalThis.cancelAnimationFrame === 'function'
            ? globalThis.cancelAnimationFrame.bind(globalThis)
            : globalThis.clearTimeout.bind(globalThis);

    let secondFrame = 0;
    const firstFrame = schedule(() => {
        secondFrame = schedule(() => {
            secondFrame = 0;
            callback();
        });
    });

    return () => {
        cancel(firstFrame);
        if (secondFrame !== 0) {
            cancel(secondFrame);
        }
    };
}
