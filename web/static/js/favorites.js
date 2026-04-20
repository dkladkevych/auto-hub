/**
 * favorites.js
 *
 * Saved listings. Stores IDs in localStorage.
 * Updates the header counter and button states.
 *
 * Connected to:
 * - public/home.html (card buttons)
 * - public/listing.html (detail page button)
 * - public/base.html (Saved link + counter)
 * - public/saved.html (saved page)
 */
(function () {
    const STORAGE_KEY = "favorites";

    function getFavorites() {
        try {
            return JSON.parse(localStorage.getItem(STORAGE_KEY)) || [];
        } catch (e) {
            return [];
        }
    }

    function saveFavorites(ids) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
        updateCounter();
        updateSavedLink();
    }

    function toggleFavorite(listingId) {
        const ids = getFavorites();
        const idx = ids.indexOf(listingId);
        if (idx >= 0) {
            ids.splice(idx, 1);
        } else {
            ids.push(listingId);
        }
        saveFavorites(ids);
        return idx < 0; // true if added
    }

    function isFavorite(listingId) {
        return getFavorites().includes(listingId);
    }

    function updateButtons() {
        document.querySelectorAll("[data-favorite]").forEach(function (btn) {
            const id = Number(btn.dataset.favorite);
            const active = isFavorite(id);
            btn.classList.toggle("is-active", active);
            btn.setAttribute("aria-pressed", String(active));
        });
    }

    function updateCounter() {
        const counter = document.getElementById("savedCount");
        if (counter) {
            counter.textContent = getFavorites().length;
        }
    }

    function updateSavedLink() {
        const link = document.getElementById("savedLink");
        if (link) {
            const ids = getFavorites();
            if (ids.length) {
                link.href = "/saved?ids=" + ids.join(",");
            } else {
                link.href = "/saved";
            }
        }
    }

    document.addEventListener("click", function (e) {
        const btn = e.target.closest("[data-favorite]");
        if (!btn) return;
        e.preventDefault();
        const id = Number(btn.dataset.favorite);
        const added = toggleFavorite(id);
        btn.classList.toggle("is-active", added);
        btn.setAttribute("aria-pressed", String(added));
    });

    document.addEventListener("DOMContentLoaded", function () {
        updateButtons();
        updateCounter();
        updateSavedLink();
    });

    window.getFavorites = getFavorites;
    window.isFavorite = isFavorite;
})();
