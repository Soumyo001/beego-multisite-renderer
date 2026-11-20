
(() => {
    "use strict";

    const app = document.getElementById("refine-app") ||
        document.getElementById("country-results");
    if (!app) return;

    const country = (app.dataset.country || "").trim();
    const isCountry = country.length > 0;

    const $ = id => document.getElementById(id);

    const PAGE_SIZE = 8;
    const FAVORITES_KEY =
        `sparrow:favorites:v1:${app.dataset.siteKey || location.host}`;

    const HEART_SVG =
        `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">` +
        `<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>` +
        `</svg>`;

    function readFavorites() {
        try {
            const parsed = JSON.parse(
                localStorage.getItem(FAVORITES_KEY) || "[]"
            );

            if (!Array.isArray(parsed)) return new Set();

            return new Set(parsed.filter(id =>
                typeof id === "string" &&
                id.length > 0 &&
                id.length <= 100
            ));
        } catch {
            return new Set();
        }
    }

    const ui = {
        heading: $("refine-heading-destination"),
        breadcrumb: $("refine-breadcrumb-destination"),
        subtitle: $("refine-subtitle-destination"),
        grid: $("refine-grid"), loadingGrid: $("refine-loading-grid"),
        status: $("refine-status"),
        sort: $("refine-sort"),
        paginationWrap: $("refine-pagination-wrap"),
        pagination: $("refine-pagination"),
        pageSummary: $("refine-page-summary"),
        filter: $("refine-filter-dialog"), filterClose: $("refine-filter-close"),
        filterClear: $("refine-filter-clear"), filterSearch: $("refine-filter-search"),
        eco: $("refine-eco-only"), modalDates: $("refine-modal-dates"),
        modalDateSummary: $("refine-modal-date-summary"),
        guestSection: $("refine-guest-section"), guestNumber: $("refine-guest-number"),
        guestMinus: $("refine-guest-minus"), guestPlus: $("refine-guest-plus"),
        priceSection: $("refine-price-section"), priceMin: $("refine-min-price"),
        priceMax: $("refine-max-price"), rangeMin: $("refine-range-min"),
        rangeMax: $("refine-range-max"), rangeTrack: $("refine-range-track"),
        currencyMin: $("refine-price-currency-min"), currencyMax: $("refine-price-currency-max"),
        amenities: $("refine-amenities-grid"),
        calendar: $("refine-calendar-dialog"), calendarClose: $("refine-calendar-close"),
        calendarHint: $("refine-calendar-hint"), calendarMonths: $("refine-calendar-months"),
        monthPrev: $("refine-month-prev"), monthNext: $("refine-month-next"),
        calendarClear: $("refine-calendar-clear"), calendarApply: $("refine-calendar-apply"),
    };

    const state = {
        search: "", pax: 2, dateStart: "", dateEnd: "",
        properties: [], visibleCount: PAGE_SIZE,
        priceMin: null, priceMax: null, eco: false, amenities: new Set(),
        favorites: readFavorites()
    };

    function syncFavoriteButton(button) {
        const id = button.dataset.propertyId;
        const selected = state.favorites.has(id);
        const name = button.dataset.propertyName || "property";

        button.setAttribute("aria-pressed", String(selected));

        button.setAttribute(
            "aria-label",
            selected
                ? `Remove ${name} from favorites`
                : `Add ${name} to favorites`
        );
    }

    function makeFavoriteButton(id, name) {
        const button = node("button", "refine-favorite");

        button.type = "button";
        button.dataset.propertyId = String(id);
        button.dataset.propertyName = String(name);

        // Static icon markup, not HTML from the API.
        button.innerHTML = HEART_SVG;

        syncFavoriteButton(button);

        return button;
    }

    // Navigate to the SSR property page when the user clicks
    // a non-interactive part of a listing card.
    //
    // Keep carousel controls, favorites and provider links
    // independent from property-details navigation.
    ui.grid.addEventListener("click", event => {
        const target = event.target instanceof Element
            ? event.target
            : null;

        if (!target) return;

        // Native links and interactive controls retain
        // their existing behavior.
        if (target.closest(
            "button, a, input, select, textarea, label"
        )) {
            return;
        }

        const card = target.closest("article.refine-card");

        if (!card || !ui.grid.contains(card)) return;

        const path = card.dataset.detailsUrl || "";

        if (!path.startsWith("/property/")) return;

        window.location.assign(path);
    });

    // Event delegation supports both existing SSR cards
    // and CSR cards added later through infinite scrolling.
    ui.grid.addEventListener("click", event => {
        const button = event.target instanceof Element
            ? event.target.closest("button.refine-favorite")
            : null;

        if (!button || !ui.grid.contains(button)) return;

        const id = button.dataset.propertyId;
        if (!id) return;

        if (state.favorites.has(id)) {
            state.favorites.delete(id);
        } else {
            state.favorites.add(id);
        }

        ui.grid.querySelectorAll("button.refine-favorite")
            .forEach(candidate => {
                if (candidate.dataset.propertyId === id) {
                    syncFavoriteButton(candidate);
                }
            });

        try {
            localStorage.setItem(
                FAVORITES_KEY,
                JSON.stringify([...state.favorites])
            );
        } catch {
            // Storage is optional; in-memory favorites still work.
        }
    });

    // UI-L5: appended result batches, with the same eight-item grid.
    let visibleResults = [];
    let countryNextURL = "";
    let countryLoaded = 0;
    let countryRequestInFlight = false;
    let countryLoadError = false;
    let csrBatchPending = false;
    const countryPageStart = isCountry
        ? (Math.max(1, Number(app.dataset.page) || 1) - 1) * PAGE_SIZE
        : 0;
    let draft = null;
    let latestRequest = 0;
    let calendarOrigin = "standalone";
    let chosenStart = "", chosenEnd = "", visibleMonth = new Date();

    // Keep instant client operations visually consistent with real API loading.
    // This is a presentation-only minimum, not a server-side delay.
    const MIN_LOADING_MS = 180;
    const MIN_BATCH_LOADING_MS = 900;
    let loadingSequence = 0;
    let fetching = false;
    let pendingNavigation = null;

    function makeLoadingCards() {
        if (ui.loadingGrid.childElementCount) return;

        const fragment = document.createDocumentFragment();
        for (let i = 0; i < PAGE_SIZE; i++) {
            const card = node("div", "refine-skeleton-card");
            const image = node("div", "refine-skeleton-media refine-shimmer");
            const body = node("div", "refine-skeleton-content");

            for (const width of ["70%", "90%", "83%", "42%"]) {
                const line = node("div", "refine-skeleton-line refine-shimmer");
                line.style.width = width;
                body.append(line);
            }

            card.append(image, body);
            fragment.append(card);
        }
        ui.loadingGrid.replaceChildren(fragment);
    }

    function showLoading() {
        makeLoadingCards();
        const ticket = {
            id: ++loadingSequence,
            started: performance.now(),
            paginationWasHidden: ui.paginationWrap ? ui.paginationWrap.hidden : true
        };
        ui.grid.hidden = true;
        ui.loadingGrid.hidden = false;
        ui.grid.setAttribute("aria-busy", "true");
        if (ui.paginationWrap) ui.paginationWrap.hidden = true;
        ui.status.textContent = "Loading properties...";
        ui.status.classList.remove("is-visible");
        return ticket;
    }

    function finishLoading(ticket) {
        if (ticket.id !== loadingSequence) return;
        ui.loadingGrid.hidden = true;
        ui.grid.hidden = false;
        ui.grid.removeAttribute("aria-busy");
        if (isCountry && ui.paginationWrap) {
            ui.paginationWrap.hidden = ticket.paginationWasHidden;
        }
    }

    function waitForLoading(ticket) {
        const remaining = Math.max(0, MIN_LOADING_MS - (performance.now() - ticket.started));
        return new Promise(resolve => setTimeout(resolve, remaining));
    }

    // UI-L5: minimum visual duration, not scroll debouncing.
    function waitForBatchMinimum(started) {
        const remaining = Math.max(
            0, MIN_BATCH_LOADING_MS - (performance.now() - started)
        );

        return new Promise(resolve =>
            window.setTimeout(resolve, remaining)
        );
    }

    function renderWithLoading(afterRender) {
        // An API request already owns the loading state.
        // Its final render will use the latest filters.
        if (fetching) {
            updatePills();
            return;
        }

        // Client-side sort/filter operations already have their data.
        // Render immediately without hiding the grid or showing skeletons.
        render();

        if (typeof afterRender === "function") {
            afterRender();
        }
    }

    function navigateWithLoading(url) {
        pendingNavigation = showLoading();
        const ticket = pendingNavigation;
        // Allow the browser to paint the skeleton before starting navigation.
        requestAnimationFrame(() => setTimeout(() => {
            if (ticket.id === loadingSequence) window.location.assign(url);
        }, 0));
    }

    function node(tag, className = "", text = undefined) {
        const n = document.createElement(tag);
        if (className) n.className = className;
        if (text !== undefined) n.textContent = String(text);
        return n;
    }

    function parseURL() {
        const params = new URLSearchParams(location.search);

        const requestedSort = params.get("sort") || "popular";

        if (ui.sort) {
            ui.sort.value = ["popular", "price_low", "price_high"]
                .includes(requestedSort)
                ? requestedSort
                : "popular";
        }

        state.search = isCountry
            ? country
            : (params.get("search") || "").trim();
    
        const pax = Number(params.get("pax") || 2);
        state.pax = Number.isInteger(pax) && pax >= 1 && pax <= 20
            ? pax
            : 2;
    
        state.dateStart = params.get("dateStart") || "";
        state.dateEnd = params.get("dateEnd") || "";

        const parsePrice = value =>
            value !== null &&
            value.trim() !== "" &&
            Number.isSafeInteger(Number(value)) &&
            Number(value) >= 0
                ? Number(value)
                : null;

        state.priceMin = parsePrice(params.get("priceMin"));
        state.priceMax = parsePrice(params.get("priceMax"));

        state.eco = params.get("eco") === "1";

        state.amenities = new Set(
            params.getAll("amenity")
                .filter(value => value.length > 0 && value.length <= 128)
                .slice(0, 100)
        );
        
        if (!isCountry) {
            const search = state.search;
            const name = !search
                ? "your destination"
                : search.toLowerCase() === "usa"
                    ? "USA"
                    : search.charAt(0).toUpperCase() + search.slice(1);

            ui.heading.textContent = name;
            ui.breadcrumb.textContent = name;
            ui.subtitle.textContent = name;
        }
    
        updatePills();
    }

    // Build the same search URL from either page.
    function buildRefineURL() {
        const params = new URLSearchParams({
            search: state.search,
            pax: String(state.pax),
            order: "1"
        });

        if (state.dateStart && state.dateEnd) {
            params.set("dateStart", state.dateStart);
            params.set("dateEnd", state.dateEnd);
        }

        if (ui.sort && ui.sort.value !== "popular") {
            params.set("sort", ui.sort.value);
        }

        if (state.priceMin !== null) {
            params.set("priceMin", String(state.priceMin));
        }

        if (state.priceMax !== null) {
            params.set("priceMax", String(state.priceMax));
        }

        if (state.eco) {
            params.set("eco", "1");
        }

        for (const amenity of [...state.amenities].sort()) {
            params.append("amenity", amenity);
        }

        return `/refine?${params.toString()}`;
    }

    // Only the refine page modifies its URL in place.
    function updateURL() {
        if (!isCountry) {
            history.pushState({}, "", buildRefineURL());
        }
    }
    function updatePills() {
        const moreCount = (state.eco ? 1 : 0) + state.amenities.size;

        const active = {
            dates: Boolean(state.dateStart && state.dateEnd),
            price: state.priceMin !== null || state.priceMax !== null,
            guests: state.pax !== 2,
            more: moreCount > 0
        };

        let priceText = "Price";

        if (active.price) {
            const currency = priceBounds().currency;
            const symbols = {
                USD: "$",
                BDT: "৳",
                EUR: "€",
                GBP: "£"
            };

            priceText = `Price ${symbols[currency] || currency}`;
        }

        const labels = {
            dates: active.dates
                ? `${shortDate(state.dateStart)} – ${shortDate(state.dateEnd)}`
                : "Dates",
            price: priceText,
            guests: active.guests
                ? `${state.pax} Guests`
                : "Guests",
            more: "More"
        };

        document.querySelectorAll("[data-filter-chip]").forEach(chip => {
            const type = chip.dataset.filterChip;
            const opener = chip.querySelector("[data-filter-open]");
            const remove = chip.querySelector("[data-filter-remove]");

            opener.textContent = labels[type];
            opener.setAttribute("aria-pressed", String(active[type]));

            if (remove) {
                remove.hidden = !active[type];
            }
        });

        const countBadge = document.getElementById("refine-more-count");

        countBadge.textContent = String(moreCount);
        countBadge.hidden = moreCount === 0;
    }
    function priceAmount(p) {
        const n = Number(p.price_from);
        return Number.isFinite(n) && n > 0 ? n : 0;
    }
    function priceLabel(p) {
        const amount = priceAmount(p);
        if (!amount) return "";
        return p.currency === "BDT"
            ? `৳${amount.toLocaleString("en-US")}`
            : `${p.currency || ""} ${amount.toLocaleString("en-US")}`.trim();
    }
    function priceBounds() {
        if (isCountry) {
            const low = Number(app.dataset.priceLow) || 0;
            const high = Number(app.dataset.priceHigh) || 100;

            return {
                low,
                high: Math.max(low + 1, high),
                currency: app.dataset.priceCurrency || "USD"
            };
        }
        const values = state.properties.map(priceAmount).filter(n => n > 0);
        const low = values.length ? Math.floor(Math.min(...values)) : 0;
        const high = values.length ? Math.ceil(Math.max(...values)) : 100;
        const currencies = new Set(state.properties.map(p => p.currency).filter(Boolean));
        const currency = currencies.size === 1 ? [...currencies][0] : "USD";
        return { low, high: Math.max(low + 1, high), currency };
    }

    function imageURLs(images) {
        if (!Array.isArray(images)) return [];

        const result = [];

        for (const raw of images) {
            if (result.length === 5) break;
            if (typeof raw !== "string" || !raw) continue;

            try {
                const url = new URL(raw);

                if ((url.protocol === "https:" || url.protocol === "http:") &&
                    !url.username && !url.password) {
                    result.push(url.href);
                }
            } catch {
                // Skip invalid image URLs.
            }
        }

        return result;
    }

    const carousels = new WeakMap();

    function updateCarouselUI(picture, gallery) {
        picture.querySelectorAll(".refine-carousel-arrow").forEach(button => {
            const direction = button.dataset.carouselDirection;

            button.hidden = direction === "prev"
                ? gallery.index === 0
                : gallery.index === gallery.photos.length - 1;
        });

        picture.querySelectorAll(".refine-carousel-dot").forEach((dot, index) => {
            dot.setAttribute(
                "aria-pressed",
                String(index === gallery.index)
            );
        });
    }

    function initializeCarousel(picture, photos, name) {
        const img = picture.querySelector(".refine-card-image");

        if (!img || photos.length === 0 || carousels.has(picture)) {
            return;
        }

        let fallback = picture.querySelector(".refine-image-fallback");

        if (!fallback) {
            fallback = node(
                "div",
                "refine-image-fallback",
                "Photo unavailable"
            );
            fallback.hidden = true;
            picture.prepend(fallback);
        }

        img.addEventListener("error", () => {
            img.hidden = true;
            fallback.hidden = false;
        });

        img.addEventListener("load", () => {
            img.hidden = false;
            fallback.hidden = true;
        });

        if (img.complete && img.naturalWidth === 0) {
            img.hidden = true;
            fallback.hidden = false;
        }

        const gallery = {
            photos,
            index: 0,
            name,
            busy: false
        };

        carousels.set(picture, gallery);

        // A single-photo property needs no navigation controls.
        if (photos.length < 2) return;

        for (const [direction, path] of [
            ["prev", "M15 18l-6-6 6-6"],
            ["next", "M9 18l6-6-6-6"]
        ]) {
            const button = node(
                "button",
                `refine-carousel-arrow refine-carousel-${direction}`
            );

            button.type = "button";
            button.dataset.carouselDirection = direction;

            button.setAttribute(
                "aria-label",
                `${direction === "prev" ? "Previous" : "Next"} photo of ${name}`
            );

            // Static SVG path; no API-provided HTML is inserted.
            button.innerHTML =
                `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">` +
                `<path d="${path}"/></svg>`;

            picture.append(button);
        }

        const dots = node("div", "refine-carousel-dots");

        dots.setAttribute("role", "group");
        dots.setAttribute("aria-label", `Photos of ${name}`);

        photos.forEach((_, index) => {
            const dot = node("button", "refine-carousel-dot");

            dot.type = "button";
            dot.dataset.carouselGo = String(index);

            dot.setAttribute(
                "aria-label",
                `Show photo ${index + 1} of ${photos.length} for ${name}`
            );

            dot.setAttribute(
                "aria-pressed",
                String(index === 0)
            );

            dots.append(dot);
        });

        picture.append(dots);
        updateCarouselUI(picture, gallery);
    }

    // One listener supports SSR, CSR and future appended cards.
    ui.grid.addEventListener("click", async event => {
        const control = event.target instanceof Element
            ? event.target.closest(
                "button[data-carousel-direction], button[data-carousel-go]"
            )
            : null;

        if (!control || !ui.grid.contains(control)) return;

        const picture = control.closest(".refine-card-picture");
        const gallery = picture && carousels.get(picture);

        if (!gallery || gallery.busy) return;

        let target = gallery.index;

        if (control.hasAttribute("data-carousel-go")) {
            target = Number(control.dataset.carouselGo);
        } else {
            target += control.dataset.carouselDirection === "next"
                ? 1
                : -1;
        }

        // No wraparound: stop at the first and last photos.
        if (!Number.isInteger(target) ||
            target < 0 ||
            target >= gallery.photos.length ||
            target === gallery.index) {
            return;
        }

        gallery.busy = true;
        picture.setAttribute("aria-busy", "true");

        // Create an overlaid three-dot loading indicator.
        const loader = node("div", "refine-carousel-loader");
        loader.setAttribute("role", "status");
        loader.setAttribute("aria-label", "Loading photograph");

        for (let i = 0; i < 3; i++) {
            loader.append(node("span", ""));
        }

        loader.hidden = true;
        picture.append(loader);

        // Avoid flashing the loader for quickly cached photos.
        const timer = window.setTimeout(() => {
            if (gallery.busy && picture.isConnected) {
                loader.hidden = false;
            }
        }, 120);

        let incoming = null;
        let oldAnimation = null;
        let newAnimation = null;

        try {
            // Load the next photograph without replacing the
            // currently visible image.
            incoming = new Image();
            incoming.className =
                "refine-card-image refine-carousel-incoming";

            incoming.alt =
                `Illustrative accommodation photograph ${target + 1} ` +
                `of ${gallery.photos.length} for ${gallery.name}`;

            incoming.decoding = "async";
            incoming.src = gallery.photos[target];

            // Wait until the new image is fully decoded.
            await incoming.decode();

            // The card might have been removed by CSR filtering.
            if (!picture.isConnected) return;

            const outgoing = picture.querySelector(
                ".refine-card-image"
            );

            if (!outgoing) return;

            window.clearTimeout(timer);
            loader.hidden = true;

            const fallback = picture.querySelector(
                ".refine-image-fallback"
            );

            if (fallback) fallback.hidden = true;

            // The incoming photograph is placed above the current
            // one only AFTER it has finished loading.
            picture.append(incoming);

            const reducedMotion = window.matchMedia(
                "(prefers-reduced-motion: reduce)"
            ).matches;

            if (!reducedMotion) {
                const shift = target > gallery.index ? 100 : -100;

                const options = {
                    duration: 280,
                    easing: "cubic-bezier(0.22, 1, 0.36, 1)",
                    fill: "forwards"
                };

                oldAnimation = outgoing.animate([
                    { transform: "translateX(0)" },
                    { transform: `translateX(${-shift}%)` }
                ], options);

                newAnimation = incoming.animate([
                    { transform: `translateX(${shift}%)` },
                    { transform: "translateX(0)" }
                ], options);

                await Promise.all([
                    oldAnimation.finished,
                    newAnimation.finished
                ]);

                oldAnimation.cancel();
                newAnimation.cancel();
            }

            // The loaded photograph becomes the main image.
            outgoing.remove();
            incoming.classList.remove(
                "refine-carousel-incoming"
            );

            gallery.index = target;
            updateCarouselUI(picture, gallery);

        } catch (error) {
            // If the next photo fails, keep the old one visible.
            oldAnimation?.cancel();
            newAnimation?.cancel();
            incoming?.remove();

            if (picture.isConnected) {
                const message = node(
                    "div",
                    "refine-carousel-error",
                    "Photo unavailable"
                );

                picture.append(message);

                window.setTimeout(() => {
                    message.remove();
                }, 1500);
            }

        } finally {
            window.clearTimeout(timer);
            loader.remove();

            gallery.busy = false;
            picture.removeAttribute("aria-busy");
        }
    });

    // Enhance existing Go-rendered cards.
    // Can also be reused when SSR batches are appended later.
    function initializeSSRCarousels(root) {
        root.querySelectorAll(".refine-carousel-sources")
            .forEach(sources => {
                const picture = sources.closest(
                    ".refine-card-picture"
                );

                if (!picture) return;

                const images = imageURLs(
                    [...sources.querySelectorAll("[data-photo-url]")]
                        .map(element => element.dataset.photoUrl)
                );

                const title = picture.closest(".refine-card")
                    ?.querySelector(".refine-card-title")
                    ?.textContent?.trim() || "property";

                initializeCarousel(picture, images, title);
                sources.remove();
            });
    }

    function bookingURL(property) {
        const raw = property.booking_url;
        if (typeof raw !== "string" || !raw.trim()) return "";
        const domains = { "booking.com": "booking.com", "expedia": "expedia.com", "expedia.com": "expedia.com" };
        const domain = domains[String(property.provider_name || "").trim().toLowerCase()];
        if (!domain) return "";
        try {
            const url = new URL(raw.trim());
            const host = url.hostname.toLowerCase();
            if (url.protocol !== "https:" ||
                !(host === domain || host.endsWith(`.${domain}`)) ||
                url.username || url.password || url.port || url.pathname === "/") return "";
            return url.href;
        } catch { return ""; }
    }
    function propertyDetailsURL(property) {
        if (typeof property.slug !== "string" ||
            typeof property.id !== "string" ||
            !property.slug || !property.id) {
            return "";
        }

        return `/property/${encodeURIComponent(property.slug)}` +
            `/${encodeURIComponent(property.id)}`;
    }

    function propertyCard(property) {
        const card = node("article", "refine-card");
        const detailsURL = propertyDetailsURL(property);

        if (detailsURL) {
            card.dataset.detailsUrl = detailsURL;
        }

        const picture = node("div", "refine-card-picture");

        const photos = imageURLs(property.images);

        if (photos.length) {
            const img = node("img", "refine-card-image");

            img.src = photos[0];
            img.alt =
                `Illustrative accommodation photograph 1 of ` +
                `${photos.length} for ${property.name}`;

            img.loading = "lazy";
            img.decoding = "async";

            picture.append(img);

            initializeCarousel(picture, photos, property.name);
        } else {
            picture.append(
                node(
                    "div",
                    "refine-image-fallback",
                    "Photo unavailable"
                )
            );
        }

        picture.append(node("span", "refine-price", priceLabel(property) ? `From ${priceLabel(property)}` : "Check provider for price"));
        if (property.eco_friendly) picture.append(node("span", "refine-eco", "Eco"));
        picture.append(
            makeFavoriteButton(property.id, property.name)
        );

        const body = node("div", "refine-card-body");
        const meta = node("div", "refine-card-meta");
        const rating = node("span", "refine-rating");
        rating.append(node("span", "refine-rating-icon", "●"));
        rating.append(document.createTextNode(`${Number(property.rating || 0).toFixed(1)} (${property.review_count || 0} Reviews)`));
        meta.append(rating, node("span", "refine-property-type", property.property_type || ""));
        const title = node("h2", "refine-card-title");

        if (detailsURL) {
            const link = node(
                "a",
                "refine-details-link",
                property.name
            );

            link.href = detailsURL;
            title.append(link);
        } else {
            title.textContent = property.name;
        }

        body.append(meta, title);
        const details = [property.size_sqft > 0 ? `${property.size_sqft} ft²` : "", property.property_type,
            ...(Array.isArray(property.amenities) ? property.amenities : [])].filter(Boolean).join(" · ");
        body.append(node("p", "refine-card-details", details));
        const loc = property.location || {};
        body.append(node("p", "refine-card-location", [loc.city, loc.area, loc.state].filter(Boolean).join(" › ")));
        const bottom = node("div", "refine-card-bottom");
        bottom.append(node("span", "refine-provider", property.provider_name || ""));

        const providerURL = bookingURL(property);

        if (providerURL) {
            const link = node("a", "refine-availability", "View Availability");
            link.href = providerURL;
            link.rel = "noopener noreferrer";
            bottom.append(link);
        } else {
            bottom.append(
                node("span", "country-unavailable", "Provider link unavailable")
            );
        }
        body.append(bottom);
        card.append(picture, body);
        return card;
    }

    // UI-L5: preserve SSR HTML on country routes, and append CSR cards locally.
    const moreButton = node("button", "refine-page-button", "Load more properties");
    moreButton.type = "button";

    function findNextCountryURL(root) {
        const next = root?.querySelector('a.country-page-link[rel="next"]');
        if (!next) return "";

        try {
            const target = new URL(next.getAttribute("href"), location.href);
            return target.origin === location.origin &&
                target.pathname === location.pathname ? target.href : "";
        } catch {
            return "";
        }
    }

    function updateBatchControls() {
        if (!ui.paginationWrap) return;

        const total = isCountry
            ? Math.max(0, Number(app.dataset.total) || 0)
            : visibleResults.length;
        const first = isCountry ? countryPageStart + 1 : 1;
        const shown = isCountry
            ? countryPageStart + countryLoaded
            : state.visibleCount;

        ui.paginationWrap.hidden = total === 0;
        ui.pageSummary.textContent = total
            ? `Showing ${first}–${Math.min(shown, total)} of ${total} properties`
            : "";
        moreButton.hidden = isCountry
            ? !countryNextURL
            : shown >= total;
        moreButton.disabled = countryRequestInFlight || csrBatchPending;
        moreButton.textContent = countryRequestInFlight || csrBatchPending
            ? "Loading more properties..."
            : countryLoadError
                ? "Retry loading properties"
                : "Load more properties";
    }

    async function appendCSRBatch() {
        if (csrBatchPending || fetching || ui.grid.hidden ||
            state.visibleCount >= visibleResults.length) return;

        csrBatchPending = true;
        const started = performance.now();
        const resultsForBatch = visibleResults;
        updateBatchControls();

        try {
            await waitForBatchMinimum(started);

            // Do not append stale results if filters or search changed.
            if (fetching || ui.grid.hidden ||
                resultsForBatch !== visibleResults) return;

            const next = resultsForBatch.slice(
                state.visibleCount,
                state.visibleCount + PAGE_SIZE
            );

            const fragment = document.createDocumentFragment();
            next.forEach(property =>
                fragment.append(propertyCard(property))
            );

            ui.grid.append(fragment);
            state.visibleCount += next.length;

            ui.status.textContent =
                `Showing ${state.visibleCount} of ${resultsForBatch.length} matching properties`;
        } finally {
            csrBatchPending = false;
            updateBatchControls();
        }
    }

    async function appendCountryBatch() {
        if (!countryNextURL || countryRequestInFlight) return;
        countryRequestInFlight = true;
        countryLoadError = false;
        const started = performance.now();
        updateBatchControls();

        try {
            const nextURL = countryNextURL;
            const target = new URL(nextURL, location.href);
            if (target.origin !== location.origin ||
                target.pathname !== location.pathname) {
                throw new Error("Unexpected country page URL");
            }

            const response = await fetch(target.href, {
                headers: { Accept: "text/html" }
            });
            if (!response.ok) throw new Error(`HTTP ${response.status}`);

            const html = await response.text();
            const parsed = new DOMParser().parseFromString(html, "text/html");
            const pageApp = parsed.getElementById("country-results");
            const pageGrid = pageApp?.querySelector("#refine-grid");
            const expectedPage = Number(target.searchParams.get("page") || 1);

            if (!pageApp || !pageGrid ||
                pageApp.dataset.country !== country ||
                pageApp.dataset.siteKey !== app.dataset.siteKey ||
                Number(pageApp.dataset.page) !== expectedPage) {
                throw new Error("Unexpected country page response");
            }

            const cards = [...pageGrid.querySelectorAll(":scope > article.refine-card")];
            if (!cards.length || cards.length > PAGE_SIZE) {
                throw new Error("Unexpected country batch size");
            }

            const seen = new Set(
                [...ui.grid.querySelectorAll(".refine-favorite[data-property-id]")]
                    .map(button => button.dataset.propertyId)
            );
            for (const card of cards) {
                const id = card.querySelector(".refine-favorite")?.dataset.propertyId;
                if (!id || seen.has(id)) {
                    throw new Error("Duplicate or invalid country property");
                }
                seen.add(id);
            }

            const nextPageURL = findNextCountryURL(
                pageApp.querySelector("#refine-pagination")
            );
            if (nextPageURL === nextURL) {
                throw new Error("Country pagination did not advance");
            }

            await waitForBatchMinimum(started);

            const fragment = document.createDocumentFragment();
            cards.forEach(card => fragment.append(document.importNode(card, true)));
            ui.grid.append(fragment);
            initializeSSRCarousels(ui.grid);
            ui.grid.querySelectorAll("button.refine-favorite")
                .forEach(syncFavoriteButton);

            countryLoaded += cards.length;
            countryNextURL = nextPageURL;
            ui.status.textContent = `Showing ${countryLoaded} country properties`;
            ui.status.classList.remove("is-visible");
        } catch (error) {
            countryLoadError = true;
            ui.status.textContent =
                "Unable to load more properties. Select Retry loading properties.";
            ui.status.classList.add("is-visible");
        } finally {
            countryRequestInFlight = false;
            updateBatchControls();
        }
    }

    function loadNextBatch() {
        if (isCountry) void appendCountryBatch();
        else appendCSRBatch();
    }

    function render() {
        updatePills();
        const results = state.properties.filter(p => {
            const price = priceAmount(p);
            if (state.priceMin !== null && (!price || price < state.priceMin)) return false;
            if (state.priceMax !== null && (!price || price > state.priceMax)) return false;
            if (state.eco && !p.eco_friendly) return false;
            if (![...state.amenities].every(a => Array.isArray(p.amenities) && p.amenities.includes(a))) return false;
            return true;
        });
        if (ui.sort.value === "price_low") results.sort((a,b) => priceAmount(a) - priceAmount(b) || a.id.localeCompare(b.id));
        else if (ui.sort.value === "price_high") results.sort((a,b) => priceAmount(b) - priceAmount(a) || a.id.localeCompare(b.id));
        else results.sort((a,b) => Number(b.popularity_score) - Number(a.popularity_score) || a.id.localeCompare(b.id));

        visibleResults = results;
        state.visibleCount = Math.min(state.visibleCount, results.length);

        const fragment = document.createDocumentFragment();
        if (!results.length) {
            fragment.append(node(
                "p", "refine-empty", "No properties match your search or filters."
            ));
        } else {
            results.slice(0, state.visibleCount).forEach(property => {
                fragment.append(propertyCard(property));
            });
        }

        ui.grid.replaceChildren(fragment);
        updateBatchControls();
        ui.status.textContent = results.length
            ? `Showing 1 through ${state.visibleCount} of ${results.length} matching properties`
            : "No properties match your search or filters.";
        ui.status.classList.remove("is-visible");
    }

    // Only the CSR route requests property JSON. Country SSR never calls this.
    async function load() {
        if (isCountry) return;

        const request = ++latestRequest;
        updatePills();
        const ticket = showLoading();
        fetching = true;
        state.visibleCount = PAGE_SIZE;

        const params = new URLSearchParams({
            search: state.search,
            pax: String(state.pax),
            order: "1"
        });

        if (state.dateStart || state.dateEnd) {
            params.set("dateStart", state.dateStart);
            params.set("dateEnd", state.dateEnd);
        }

        try {
            const response = await fetch(
                `/api/properties?${params}`,
                { headers: { Accept: "application/json" } }
            );
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }
            if (!Array.isArray(data.properties)) {
                throw new Error("Unexpected API response");
            }

            await waitForLoading(ticket);
            if (request !== latestRequest || ticket.id !== loadingSequence) return;
            state.properties = data.properties;
            render();
        } catch (err) {
            await waitForLoading(ticket);
            if (request !== latestRequest || ticket.id !== loadingSequence) return;

            state.properties = [];
            ui.paginationWrap.hidden = true;
            ui.grid.replaceChildren(node(
                "p", "refine-empty", "Unable to load properties. Please try again."
            ));
            ui.status.textContent = `Unable to load properties: ${err.message}`;
            ui.status.classList.add("is-visible");
        } finally {
            if (request === latestRequest) {
                fetching = false;
                finishLoading(ticket);
            }
        }
    }

    function cloneApplied() {
        return { pax: state.pax, dateStart: state.dateStart, dateEnd: state.dateEnd,
            priceMin: state.priceMin, priceMax: state.priceMax,
            eco: state.eco, amenities: new Set(state.amenities) };
    }
    function amendGuest(delta) {
        const entered = Number(ui.guestNumber.value);
        const value = Number.isInteger(entered) && entered >= 1 && entered <= 20 ? entered : draft.pax;
        draft.pax = Math.max(1, Math.min(20, value + delta));
        ui.guestNumber.value = String(draft.pax);
    }
    function numericPrice(raw, bound, fallback) {
        if (String(raw).trim() === "") return fallback;
        const n = Number(raw);
        return Number.isFinite(n) ? Math.max(bound.low, Math.min(bound.high, Math.round(n))) : fallback;
    }
    function refreshPriceFields() {
        const bounds = priceBounds();
        const min = draft.priceMin === null ? bounds.low : Math.max(bounds.low, Math.min(bounds.high, draft.priceMin));
        const max = draft.priceMax === null ? bounds.high : Math.max(bounds.low, Math.min(bounds.high, draft.priceMax));
        const low = Math.min(min, max), high = Math.max(min, max);
        for (const slider of [ui.rangeMin, ui.rangeMax]) { slider.min = String(bounds.low); slider.max = String(bounds.high); slider.step = "1"; }
        ui.rangeMin.value = String(low); ui.rangeMax.value = String(high);
        ui.priceMin.value = String(low); ui.priceMax.value = String(high);
        ui.currencyMin.textContent = bounds.currency;
        ui.currencyMax.textContent = bounds.currency;
        const width = bounds.high - bounds.low;
        ui.rangeTrack.style.background = `linear-gradient(to right, #d9e2e6 0%, #d9e2e6 ${((low-bounds.low)/width)*100}%, #ff790b ${((low-bounds.low)/width)*100}%, #ff790b ${((high-bounds.low)/width)*100}%, #d9e2e6 ${((high-bounds.low)/width)*100}%)`;
    }
    function setPrice(kind, raw) {
        const bounds = priceBounds();
        const fallback = kind === "min" ? bounds.low : bounds.high;
        const value = numericPrice(raw, bounds, fallback);
        const other = kind === "min"
            ? (draft.priceMax ?? bounds.high) : (draft.priceMin ?? bounds.low);
        const adjusted = kind === "min" ? Math.min(value, other) : Math.max(value, other);
        if (kind === "min") draft.priceMin = adjusted <= bounds.low ? null : adjusted;
        else draft.priceMax = adjusted >= bounds.high ? null : adjusted;
        refreshPriceFields();
    }
    function availableAmenities() {
        if (isCountry) {
            return [...document.querySelectorAll(
                "#country-filter-amenities [data-amenity]"
            )].map(element => element.dataset.amenity);
        }
        return [...new Set(state.properties.flatMap(p => Array.isArray(p.amenities) ? p.amenities : []))]
            .filter(Boolean).sort((a,b) => a.localeCompare(b));
    }
    function populateAmenities() {
        const fragment = document.createDocumentFragment();
        for (const amenity of availableAmenities()) {
            const label = node("label", "refine-amenity");
            const check = node("input");
            check.type = "checkbox";
            check.checked = draft.amenities.has(amenity);
            check.addEventListener("change", () => {
                if (check.checked) draft.amenities.add(amenity);
                else draft.amenities.delete(amenity);
            });
            label.append(check, node("span", "", amenity));
            fragment.append(label);
        }
        ui.amenities.replaceChildren(fragment);
    }
    function paintDraft() {
        ui.eco.checked = draft.eco;
        ui.guestNumber.value = String(draft.pax);
        ui.modalDateSummary.textContent = draft.dateStart && draft.dateEnd
            ? `${shortDate(draft.dateStart)} – ${shortDate(draft.dateEnd)}` : "Select a date";
        refreshPriceFields();
        populateAmenities();
    }
    function openFilters(focus) {
        draft = cloneApplied();
        ui.guestSection.classList.toggle("is-highlighted", focus === "guests");
        ui.priceSection.classList.toggle("is-highlighted", focus === "price");
        paintDraft();
        ui.filter.showModal();
        if (focus === "guests") ui.guestNumber.focus();
        else if (focus === "price") ui.priceMax.focus();
        else if (focus === "dates") ui.modalDates.focus();
        else ui.filterClose.focus();
    }
    function applyDraft() {
        const pax = Number(ui.guestNumber.value);
        if (!Number.isInteger(pax) || pax < 1 || pax > 20) {
            ui.guestNumber.focus(); ui.guestNumber.reportValidity(); return;
        }
        draft.pax = pax;
        const mustFetch = draft.pax !== state.pax || draft.dateStart !== state.dateStart || draft.dateEnd !== state.dateEnd;
        Object.assign(state, { pax: draft.pax, dateStart: draft.dateStart, dateEnd: draft.dateEnd,
            priceMin: draft.priceMin, priceMax: draft.priceMax, eco: draft.eco, amenities: new Set(draft.amenities) });
        ui.filter.close();

        // Country SSR: submit the chosen filters by navigation.
        // Do not fetch or alter the server-rendered property grid.
        if (isCountry) {
            navigateWithLoading(buildRefineURL());
            return;
        }

        // Refine CSR: preserve existing filtering behavior.
        updateURL();

        if (mustFetch) {
            load();
        } else {
            state.visibleCount = PAGE_SIZE;
            renderWithLoading();
        }
    }

    function isoDate(year, month, day) {
        return `${year}-${String(month+1).padStart(2,"0")}-${String(day).padStart(2,"0")}`;
    }
    function todayISO() { const d = new Date(); return isoDate(d.getFullYear(),d.getMonth(),d.getDate()); }
    function shortDate(value) {
        const d = new Date(`${value}T00:00:00Z`);
        return Number.isNaN(d.getTime()) ? value : d.toLocaleDateString("en-US", { month:"short", day:"2-digit", timeZone:"UTC" });
    }
    function calendarPanel(year, month) {
        const box = node("section", "refine-month");
        box.append(node("h3", "", new Intl.DateTimeFormat("en-US", { month: "long", year: "numeric" }).format(new Date(year,month,1))));
        const grid = node("div", "refine-month-days");
        for (const day of ["Su","Mo","Tu","We","Th","Fr","Sa"]) grid.append(node("span", "refine-day-name", day));
        for (let i=0; i<new Date(year,month,1).getDay(); i++) grid.append(node("span"));
        for (let day=1; day<=new Date(year,month+1,0).getDate(); day++) {
            const iso = isoDate(year,month,day);
            const btn = node("button", "refine-day", day);
            btn.type = "button";
            btn.disabled = iso < todayISO();
            btn.setAttribute("aria-label", new Date(year, month, day).toDateString());
            if (iso === chosenStart || iso === chosenEnd) btn.classList.add("selected");
            else if (chosenStart && chosenEnd && iso>chosenStart && iso<chosenEnd) btn.classList.add("between");
            btn.addEventListener("click", () => {
                if (!chosenStart || chosenEnd || iso <= chosenStart) { chosenStart = iso; chosenEnd = ""; }
                else chosenEnd = iso;
                paintCalendar();
            });
            grid.append(btn);
        }
        box.append(grid);
        return box;
    }
    function paintCalendar() {
        const year = visibleMonth.getFullYear(), month = visibleMonth.getMonth();
        const second = new Date(year, month+1, 1);
        ui.calendarMonths.replaceChildren(calendarPanel(year, month), calendarPanel(second.getFullYear(),second.getMonth()));
        const today = new Date();
        ui.monthPrev.disabled = year < today.getFullYear() || (year === today.getFullYear() && month <= today.getMonth());
        ui.calendarHint.textContent = chosenStart && chosenEnd
            ? `${shortDate(chosenStart)} – ${shortDate(chosenEnd)}`
            : chosenStart ? "Choose checkout" : "Choose check-in and checkout";
        ui.calendarApply.disabled = Boolean(chosenStart && !chosenEnd);
    }
    function openCalendar(fromFilters) {
        calendarOrigin = fromFilters ? "filters" : "standalone";
        chosenStart = fromFilters ? draft.dateStart : state.dateStart;
        chosenEnd = fromFilters ? draft.dateEnd : state.dateEnd;
        const date = chosenStart ? new Date(`${chosenStart}T12:00:00`) : new Date();
        visibleMonth = new Date(date.getFullYear(),date.getMonth(),1);
        paintCalendar();
        ui.calendar.showModal();
    }
    function acceptDates() {
        if (chosenStart && !chosenEnd) return;
        if (calendarOrigin === "filters") {
            draft.dateStart = chosenStart;
            draft.dateEnd = chosenEnd;
            paintDraft();
        } else {
            state.dateStart = chosenStart;
            state.dateEnd = chosenEnd;
            updateURL(); updatePills(); load();
        }
        ui.calendar.close();
    }

    // Filters are intentionally one modal, not four independent <details> panels.
    document.querySelectorAll("[data-filter-open]").forEach(button => {
        button.addEventListener("click", () => {
            const type = button.dataset.filterOpen;
            if (type === "dates" && !isCountry) {
                openCalendar(false);
            } else {
                openFilters(type);
            }
        });
    });

    document.querySelectorAll("[data-filter-remove]").forEach(button => {
        button.addEventListener("click", event => {
            event.stopPropagation();

            const type = button.dataset.filterRemove;
            let needsFetch = false;

            switch (type) {
                case "dates":
                    state.dateStart = "";
                    state.dateEnd = "";
                    needsFetch = true;
                    break;

                case "price":
                    state.priceMin = null;
                    state.priceMax = null;
                    break;

                case "guests":
                    state.pax = 2;
                    needsFetch = true;
                    break;

                case "more":
                    state.eco = false;
                    state.amenities.clear();
                    break;

                default:
                    return;
            }

            if (needsFetch) {
                updateURL();
                updatePills();
                load();
            } else {
                state.visibleCount = PAGE_SIZE;
                updateURL();
                renderWithLoading();
            }

            button.parentElement
                .querySelector("[data-filter-open]")
                .focus();
        });
    });

    ui.filterClose.addEventListener("click", () => ui.filter.close());
    ui.filterClear.addEventListener("click", () => { draft = { pax:2, dateStart:"", dateEnd:"", priceMin:null,priceMax:null,eco:false,amenities:new Set() }; paintDraft(); });
    ui.filterSearch.addEventListener("click", applyDraft);
    ui.modalDates.addEventListener("click", () => openCalendar(true));
    ui.guestNumber.addEventListener("input", () => {
        const n = Number(ui.guestNumber.value);
        if (Number.isInteger(n) && n >= 1 && n <= 20) draft.pax = n;
    });
    ui.guestMinus.addEventListener("click", () => amendGuest(-1));
    ui.guestPlus.addEventListener("click", () => amendGuest(1));
    ui.rangeMin.addEventListener("input", () => setPrice("min", ui.rangeMin.value));
    ui.rangeMax.addEventListener("input", () => setPrice("max", ui.rangeMax.value));
    ui.priceMin.addEventListener("change", () => setPrice("min", ui.priceMin.value));
    ui.priceMax.addEventListener("change", () => setPrice("max", ui.priceMax.value));
    ui.eco.addEventListener("change", () => { draft.eco = ui.eco.checked; });
    ui.calendarClose.addEventListener("click", () => ui.calendar.close());
    ui.monthPrev.addEventListener("click", () => { visibleMonth.setMonth(visibleMonth.getMonth()-1); paintCalendar(); });
    ui.monthNext.addEventListener("click", () => { visibleMonth.setMonth(visibleMonth.getMonth()+1); paintCalendar(); });
    ui.calendarClear.addEventListener("click", () => { chosenStart=""; chosenEnd=""; paintCalendar(); });
    ui.calendarApply.addEventListener("click", acceptDates);
    if (ui.sort) {
        ui.sort.addEventListener("change", () => {
            state.visibleCount = PAGE_SIZE;
            updateURL();
            renderWithLoading();
        });
    }

    if (isCountry) {
        // Restore the original server HTML when returning from the back-forward cache.
        window.addEventListener("pageshow", event => {
            if (event.persisted && pendingNavigation) {
                finishLoading(pendingNavigation);
                pendingNavigation = null;
                ui.status.textContent = "";
            }
        });
    }

    window.addEventListener("popstate", () => {
        if (!isCountry) {
            parseURL();
            load();
        }
    });

    function alignWithPublisherHeader() {
        const navigation = document.getElementById("main-navigation");
        const inner = app.querySelector(".refine-inner");

        // Other publisher sites might use different header markup.
        // CSS provides a fallback when this navigation is absent.
        if (!navigation || !inner) return;

        const navRect = navigation.getBoundingClientRect();
        const pageRect = app.getBoundingClientRect();

        if (navRect.width <= 0 || pageRect.width <= 0) return;

        const left = Math.max(0, navRect.left - pageRect.left);
        const right = Math.max(0, pageRect.right - navRect.right);

        if (left + right >= pageRect.width) return;

        inner.style.width = "auto";
        inner.style.maxWidth = "none";
        inner.style.marginLeft = `${left}px`;
        inner.style.marginRight = `${right}px`;
        inner.style.paddingInline = "0";
    }

    window.addEventListener("resize", alignWithPublisherHeader);


    alignWithPublisherHeader();
    parseURL();

    // Enhance server-rendered photos and restore favorites.
    initializeSSRCarousels(ui.grid);
    ui.grid.querySelectorAll("button.refine-favorite")
        .forEach(syncFavoriteButton);

    if (ui.paginationWrap && ui.pagination) {
        if (isCountry) {
            countryNextURL = findNextCountryURL(ui.pagination);
            countryLoaded = ui.grid.querySelectorAll(
                ":scope > article.refine-card"
            ).length;
        }

        // Replace numbered links with an accessible incremental-load control.
        ui.pagination.replaceChildren(moreButton);
        moreButton.addEventListener("click", loadNextBatch);

        if ("IntersectionObserver" in window) {
            const observer = new IntersectionObserver(entries => {
                if (entries.some(entry => entry.isIntersecting) &&
                    !moreButton.hidden && !moreButton.disabled &&
                    !countryLoadError) {
                    loadNextBatch();
                }
            }, { rootMargin: "10px 0px" });
            observer.observe(ui.paginationWrap);
        }

        if (isCountry) updateBatchControls();
    }

    // Country routes retain their Go-rendered first batch.
    if (!isCountry) load();
})();

