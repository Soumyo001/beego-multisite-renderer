// Behaviour for the server-rendered property cards in the publisher's
// tiles_component blocks: paging (tiles_per_page), favorites, the
// shared photo carousel and click-through to the details page.
(() => {
    "use strict";

    const blocks = document.querySelectorAll(".bmr-tiles");
    if (blocks.length === 0) return;

    // Same storage key as /refine, /all/:country and the details page,
    // so a favorite set here shows there too.
    const siteKey = blocks[0].dataset.siteKey || location.host;
    const favoritesKey = `bmr:favorites:v1:${siteKey}`;

    function readFavorites() {
        try {
            const parsed = JSON.parse(localStorage.getItem(favoritesKey) || "[]");
            return Array.isArray(parsed)
                ? new Set(parsed.filter(id =>
                    typeof id === "string" && id.length > 0 && id.length <= 100))
                : new Set();
        } catch {
            return new Set();
        }
    }

    function syncFavorites(favorites) {
        document.querySelectorAll(".bmr-tiles button.refine-favorite")
            .forEach(button => {
                const selected = favorites.has(button.dataset.propertyId);
                const name = button.dataset.propertyName || "property";
                button.setAttribute("aria-pressed", String(selected));
                button.setAttribute("aria-label", selected
                    ? `Remove ${name} from favorites`
                    : `Add ${name} to favorites`);
            });
    }

    function toggleFavorite(id) {
        const favorites = readFavorites();
        if (favorites.has(id)) favorites.delete(id);
        else favorites.add(id);
        try {
            localStorage.setItem(favoritesKey, JSON.stringify([...favorites]));
        } catch {
            // Storage is optional; the buttons still update.
        }
        syncFavorites(favorites);
    }

    // Shows page `page` (0-based) of a block's cards.
    function showPage(block, page) {
        const cards = [...block.querySelectorAll(".bmr-tiles-grid > .refine-card")];
        const perPage = Number(block.dataset.perPage) || cards.length;
        const pages = Math.max(1, Math.ceil(cards.length / perPage));
        const current = Math.min(Math.max(page, 0), pages - 1);

        cards.forEach((card, index) => {
            card.hidden = Math.floor(index / perPage) !== current;
        });

        block.dataset.page = String(current);

        const status = block.querySelector(".bmr-tiles-status");
        if (status) status.textContent = `${current + 1} / ${pages}`;

        block.querySelectorAll("[data-tiles-step]").forEach(button => {
            const step = Number(button.dataset.tilesStep);
            button.disabled = step < 0 ? current === 0 : current === pages - 1;
        });
    }

    syncFavorites(readFavorites());

    blocks.forEach(block => {
        // The identical /refine carousel, from static/js/listing_cards.js.
        if (window.ListingCards) {
            window.ListingCards.initializeSSRCarousels(block);
            window.ListingCards.bindCarousel(block);
        }

        block.addEventListener("click", event => {
            const target = event.target instanceof Element ? event.target : null;
            if (!target) return;

            const pager = target.closest("[data-tiles-step]");
            if (pager) {
                const page = Number(block.dataset.page || "0");
                showPage(block, page + Number(pager.dataset.tilesStep));
                return;
            }

            const favorite = target.closest("button.refine-favorite");
            if (favorite) {
                toggleFavorite(favorite.dataset.propertyId);
                return;
            }

            // Links and other controls keep their own behaviour.
            if (target.closest("button, a, input, select, textarea, label")) return;

            const card = target.closest("article.refine-card");
            const path = card?.dataset.detailsUrl || "";
            if (path.startsWith("/property/")) window.location.assign(path);
        });
    });
})();