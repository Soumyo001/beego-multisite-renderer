// Shared listing-card behaviour for /refine, /all/:country and
// the property-details "Vacation Rentals Near" section.
//
// The carousel code is moved unchanged from static/js/refine.js
// so every page has the identical photo carousel.
(() => {
    "use strict";

    function node(tag, className = "", text = undefined) {
        const n = document.createElement(tag);
        if (className) n.className = className;
        if (text !== undefined) n.textContent = String(text);
        return n;
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

    // One listener per grid supports SSR, CSR and future appended cards.
    const boundGrids = new WeakSet();

    function bindCarousel(grid) {
        if (!grid || boundGrids.has(grid)) return;
        boundGrids.add(grid);

        grid.addEventListener("click", async event => {
            const control = event.target instanceof Element
                ? event.target.closest(
                    "button[data-carousel-direction], button[data-carousel-go]"
                )
                : null;

            if (!control || !grid.contains(control)) return;

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
    }

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

    // Local partner logos for the two providers allowed by
    // bookingURL() in refine.js and safeBooking() in Go.
    const PROVIDER_LOGOS = {
        "booking.com": "/static/img/partners/booking.svg",
        "expedia": "/static/img/partners/expedia.svg",
        "expedia.com": "/static/img/partners/expedia.svg"
    };

    // Returns a logo <img> for a known provider, otherwise the
    // provider name as text (the previous behaviour).
    function providerLabel(providerName) {
        const name = String(providerName || "").trim();
        const logo = PROVIDER_LOGOS[name.toLowerCase()];

        if (!logo) return node("span", "refine-provider", name);

        const img = node("img", "refine-provider-logo");
        img.src = logo;
        img.alt = name;
        img.loading = "lazy";
        img.decoding = "async";
        return img;
    }

    window.SparrowCards = Object.freeze({
        imageURLs,
        initializeCarousel,
        initializeSSRCarousels,
        bindCarousel,
        providerLabel
    });
})();
