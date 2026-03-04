// Theme store: persists to localStorage, applies class to <html>
type Theme = "dark" | "light";

function createThemeStore() {
    let theme = $state<Theme>("dark");

    function init() {
        const stored = localStorage.getItem("updu_theme") as Theme | null;
        if (stored === "light" || stored === "dark") {
            theme = stored;
        }
        apply();
    }

    function apply() {
        if (typeof document !== "undefined") {
            document.documentElement.classList.toggle("light", theme === "light");
        }
    }

    function toggle() {
        theme = theme === "dark" ? "light" : "dark";
        localStorage.setItem("updu_theme", theme);
        apply();
    }

    return {
        get current() {
            return theme;
        },
        init,
        toggle,
    };
}

export const themeStore = createThemeStore();
