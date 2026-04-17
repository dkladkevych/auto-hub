/**
 * theme.js
 *
 * Переключение между светлой и тёмной темой.
 * Сохраняет выбор в localStorage и применяет перед отрисовкой страницы.
 *
 * Связан с base.html (атрибут data-theme на <html>).
 * Иконки sun/moon переключаются через CSS, JS только меняет data-theme.
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

    // Применяем сохранённую тему мгновенно (до первой отрисовки)
    setTheme(getSavedTheme());

    window.toggleTheme = toggleTheme;
})();
