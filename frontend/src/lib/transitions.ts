// Centralized motion utilities. All helpers respect prefers-reduced-motion.
import { cubicOut } from 'svelte/easing';
import type { TransitionConfig } from 'svelte/transition';

function prefersReducedMotion(): boolean {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
        return false;
    }
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export const DURATION = {
    fast: 120,
    normal: 200,
    slow: 320,
} as const;

export const easeOutExpo = (t: number) => 1 - Math.pow(2, -10 * t);

interface FadeOptions {
    duration?: number;
    delay?: number;
}

export function fade(_node: Element, options: FadeOptions = {}): TransitionConfig {
    const reduced = prefersReducedMotion();
    return {
        duration: reduced ? 0 : options.duration ?? DURATION.normal,
        delay: options.delay ?? 0,
        easing: cubicOut,
        css: (t) => `opacity: ${t};`,
    };
}

interface ScaleFadeOptions extends FadeOptions {
    start?: number;
}

export function scaleFade(
    _node: Element,
    options: ScaleFadeOptions = {},
): TransitionConfig {
    const reduced = prefersReducedMotion();
    const start = options.start ?? 0.96;
    return {
        duration: reduced ? 0 : options.duration ?? DURATION.normal,
        delay: options.delay ?? 0,
        easing: easeOutExpo,
        css: (t) => {
            const scale = start + (1 - start) * t;
            return `opacity: ${t}; transform: scale(${scale});`;
        },
    };
}

interface SlideOptions extends FadeOptions {
    distance?: number;
    axis?: 'x' | 'y';
}

export function slide(
    _node: Element,
    options: SlideOptions = {},
): TransitionConfig {
    const reduced = prefersReducedMotion();
    const distance = options.distance ?? 8;
    const axis = options.axis ?? 'y';
    return {
        duration: reduced ? 0 : options.duration ?? DURATION.normal,
        delay: options.delay ?? 0,
        easing: easeOutExpo,
        css: (t, u) => {
            const offset = u * distance;
            const transform =
                axis === 'y' ? `translateY(${offset}px)` : `translateX(${offset}px)`;
            return `opacity: ${t}; transform: ${transform};`;
        },
    };
}
