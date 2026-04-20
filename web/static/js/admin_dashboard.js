/**
 * admin_dashboard.js
 *
 * AJAX pagination for the admin listings table.
 * Connected to admin/dashboard.html.
 *
 * The loadAdminPage function fetches an HTML fragment from the server
 * and injects it into #adminListingsBlock, updating the URL without a reload.
 *
 * Depends on the global window.ADMIN_PATH variable (passed from the template).
 */
async function loadAdminPage(page) {
    const block = document.getElementById("adminListingsBlock");
    if (!block) return;

    const adminPath = window.ADMIN_PATH || "admin";
    const response = await fetch(`/${adminPath}/listings-block?page=${page}`);
    const html = await response.text();
    block.innerHTML = html;

    const url = new URL(window.location);
    url.searchParams.set("page", page);
    window.history.replaceState({}, "", url);
}
