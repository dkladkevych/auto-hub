/**
 * filters.js
 *
 * Controls the advanced filters panel on the home page.
 * Connected to public/base.html (the #advancedFilters element).
 *
 * Functions:
 * - toggleAdvancedFilters — opens/closes the filter panel on ⚙ click.
 * - On page load, automatically opens the panel if the URL contains
 *   any filter parameters.
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
