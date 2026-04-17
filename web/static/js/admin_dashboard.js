/**
 * admin_dashboard.js
 *
 * AJAX-пагинация таблицы объявлений в админ-панели.
 * Связан с шаблоном admin/dashboard.html.
 *
 * Функция loadAdminPage подгружает HTML-фрагмент с сервера
 * и вставляет его в #adminListingsBlock, обновляя URL без перезагрузки.
 *
 * Зависит от глобальной переменной window.ADMIN_PATH (передаётся из шаблона).
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
