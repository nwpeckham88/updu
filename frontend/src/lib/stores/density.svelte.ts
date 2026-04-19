// Density store: persists UI compactness to localStorage, applies class to <html>
type Density = "comfortable" | "cozy" | "compact";

const STORAGE_KEY = "updu_density";
const DENSITY_ORDER: Density[] = ["comfortable", "cozy", "compact"];

function createDensityStore() {
    let density = $state<Density>("cozy");

    function init() {
        if (typeof localStorage === "undefined") return;
        const stored = localStorage.getItem(STORAGE_KEY) as Density | null;
        if (stored && DENSITY_ORDER.includes(stored)) {
            density = stored;
        }
        apply();
    }

    function apply() {
        if (typeof document === "undefined") return;
        const root = document.documentElement;
        for (const d of DENSITY_ORDER) {
            root.classList.toggle(`density-${d}`, density === d);
        }
    }

    function set(next: Density) {
        if (!DENSITY_ORDER.includes(next)) return;
        density = next;
        if (typeof localStorage !== "undefined") {
            localStorage.setItem(STORAGE_KEY, next);
        }
        apply();
    }

    function cycle() {
        const idx = DENSITY_ORDER.indexOf(density);
        set(DENSITY_ORDER[(idx + 1) % DENSITY_ORDER.length]);
    }

    return {
        get current() {
            return density;
        },
        get options() {
            return DENSITY_ORDER;
        },
        init,
        set,
        cycle,
    };
}

export const densityStore = createDensityStore();
