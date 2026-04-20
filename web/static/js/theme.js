/**
 * theme.js
 *
 * Toggle between light and dark theme.
 * Saves the choice in localStorage and applies it before the page renders.
 *
 * Connected to base.html (data-theme attribute on <html>).
 * Sun/moon icons are toggled via CSS; JS only changes data-theme.
 */
(function () {
    const STORAGE_KEY = "theme";
    const html = document.documentElement;

    function getSavedTheme() {
        return localStorage.getItem(STORAGE_KEY) || "light";
    }

    function setTheme(theme) {
        html.setAttribute("data-theme", theme);
        localStorage.setItem(STORAGE_KEY, theme);
    }

    function toggleTheme() {
        const current = html.getAttribute("data-theme") === "dark" ? "dark" : "light";
        setTheme(current === "dark" ? "light" : "dark");
    }

    // Apply the saved theme instantly (before first paint)
    setTheme(getSavedTheme());

    window.toggleTheme = toggleTheme;
})();
