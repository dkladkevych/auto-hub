/**
 * filters.js
 *
 * Управляет панелью расширенных фильтров на главной странице.
 * Связан с шаблоном public/base.html (элемент #advancedFilters).
 *
 * Функции:
 * - toggleAdvancedFilters — открывает/закрывает панель фильтров по клику на ⚙.
 * - При загрузке страницы автоматически открывает панель, если в URL
 *   присутствуют параметры фильтрации.
 */

function toggleAdvancedFilters() {
    const el = document.getElementById("advancedFilters");
    if (el) {
        el.classList.toggle("open");
    }
}

document.addEventListener("DOMContentLoaded", function () {
    const el = document.getElementById("advancedFilters");
    const params = new URLSearchParams(window.location.search);

    const filterKeys = [
        "location",
        "price_min",
        "price_max",
        "year_min",
        "year_max",
        "mileage_min",
        "mileage_max",
        "risk_level",
        "include_unknown"
    ];

    const hasFilter = filterKeys.some(key => params.get(key));

    if (el && hasFilter) {
        el.classList.add("open");
    }
});
