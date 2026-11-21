(() => {
    "use strict";

    const checkInButton = document.getElementById("pd-checkin");
    const checkOutButton = document.getElementById("pd-checkout");
    const checkInText = document.getElementById("pd-checkin-text");
    const checkOutText = document.getElementById("pd-checkout-text");
    const status = document.getElementById("pd-date-status");
    const calendar = document.getElementById("pd-calendar-dialog");
    const months = document.getElementById("pd-calendar-months");
    const prompt = document.getElementById("pd-calendar-prompt");
    const done = document.getElementById("pd-calendar-done");
    const prev = document.getElementById("pd-calendar-prev");
    const next = document.getElementById("pd-calendar-next");
    const openFull = document.getElementById("pd-calendar-open");

    if (!checkInButton || !checkOutButton || !checkInText ||
        !checkOutText || !status || !calendar || !months ||
        !prompt || !done || !prev || !next || !openFull) return;

    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const currentMonth = new Date(today.getFullYear(), today.getMonth(), 1);
    let firstMonth = new Date(currentMonth);
    let selectedStart = "";
    let selectedEnd = "";
    let draftStart = "";
    let draftEnd = "";
    let openingButton = null;

    function iso(date) {
        return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
    }

    function fromISO(value) {
        if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return null;
        const [year, month, day] = value.split("-").map(Number);
        const date = new Date(year, month - 1, day);
        return date.getFullYear() === year &&
            date.getMonth() === month - 1 && date.getDate() === day
            ? date : null;
    }

    function displayDate(value, includeYear = false) {
        const date = fromISO(value);
        if (!date) return "Choose date";
        return date.toLocaleDateString("en-US", {
            month: "short", day: "numeric",
            ...(includeYear ? { year: "numeric" } : {})
        });
    }

    function renderSelection() {
        checkInText.textContent = selectedStart ? displayDate(selectedStart) : "Choose date";
        checkOutText.textContent = selectedEnd ? displayDate(selectedEnd) : "Choose date";
        status.textContent = selectedStart && selectedEnd
            ? "Dates selected for reference only. Confirm availability and final price with the provider."
            : "Select dates or continue without dates. Availability is not verified here.";
    }

    function renderFooter() {
        prompt.textContent = draftStart && draftEnd
            ? `${displayDate(draftStart)} – ${displayDate(draftEnd)}`
            : draftStart ? "Choose checkout" : "Choose check-in and checkout";
        done.disabled = Boolean(draftStart && !draftEnd);
    }

    function renderCalendar() {
        months.replaceChildren();
        const days = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];

        for (let offset = 0; offset < 2; offset++) {
            const base = new Date(firstMonth.getFullYear(), firstMonth.getMonth() + offset, 1);
            const panel = document.createElement("section");
            panel.className = "pd-month";
            const title = document.createElement("h3");
            title.textContent = base.toLocaleDateString("en-US", {
                month: "long", year: "numeric"
            });
            panel.append(title);

            const grid = document.createElement("div");
            grid.className = "pd-month-grid";
            for (const name of days) {
                const label = document.createElement("span");
                label.className = "pd-weekday";
                label.textContent = name;
                grid.append(label);
            }
            for (let blank = 0; blank < base.getDay(); blank++) {
                const placeholder = document.createElement("span");
                placeholder.setAttribute("aria-hidden", "true");
                grid.append(placeholder);
            }

            const count = new Date(base.getFullYear(), base.getMonth() + 1, 0).getDate();
            for (let day = 1; day <= count; day++) {
                const date = new Date(base.getFullYear(), base.getMonth(), day);
                const value = iso(date);
                const button = document.createElement("button");
                button.type = "button";
                button.className = "pd-calendar-day";
                button.textContent = String(day);
                button.dataset.date = value;
                button.disabled = date < today;
                button.setAttribute("aria-label", date.toLocaleDateString("en-US", {
                    weekday: "long", month: "long", day: "numeric", year: "numeric"
                }));
                const chosen = value === draftStart || value === draftEnd;
                if (chosen) button.classList.add("is-selected");
                else if (draftStart && draftEnd && value > draftStart && value < draftEnd) {
                    button.classList.add("is-between");
                }
                button.setAttribute("aria-pressed", String(chosen));
                grid.append(button);
            }
            panel.append(grid);
            months.append(panel);
        }
        prev.disabled = firstMonth.getTime() <= currentMonth.getTime();
        renderFooter();
    }

    function openCalendar(trigger, editCheckout = false) {
        openingButton = trigger;
        draftStart = selectedStart;
        draftEnd = editCheckout && selectedStart ? "" : selectedEnd;
        const focusDate = fromISO(draftStart);
        firstMonth = focusDate && focusDate >= today
            ? new Date(focusDate.getFullYear(), focusDate.getMonth(), 1)
            : new Date(currentMonth);
        renderCalendar();
        trigger.setAttribute("aria-expanded", "true");
        calendar.showModal();
    }

    checkInButton.addEventListener("click", () => openCalendar(checkInButton));
    checkOutButton.addEventListener("click", () => openCalendar(checkOutButton, true));
    openFull.addEventListener("click", () => openCalendar(openFull));

    months.addEventListener("click", event => {
        const button = event.target instanceof Element
            ? event.target.closest("button[data-date]") : null;
        if (!button || button.disabled) return;
        const chosen = button.dataset.date;
        if (!draftStart || draftEnd || chosen <= draftStart) {
            draftStart = chosen;
            draftEnd = "";
        } else {
            draftEnd = chosen;
        }
        renderCalendar();
    });

    prev.addEventListener("click", () => {
        if (firstMonth <= currentMonth) return;
        firstMonth = new Date(firstMonth.getFullYear(), firstMonth.getMonth() - 1, 1);
        renderCalendar();
    });
    next.addEventListener("click", () => {
        firstMonth = new Date(firstMonth.getFullYear(), firstMonth.getMonth() + 1, 1);
        renderCalendar();
    });
    document.getElementById("pd-calendar-clear").addEventListener("click", () => {
        draftStart = "";
        draftEnd = "";
        renderCalendar();
    });
    done.addEventListener("click", () => {
        if (done.disabled) return;
        selectedStart = draftStart;
        selectedEnd = draftEnd;
        renderSelection();
        calendar.close();
    });
    document.getElementById("pd-calendar-close").addEventListener("click", () => calendar.close());
    calendar.addEventListener("close", () => {
        if (openingButton) {
            openingButton.setAttribute("aria-expanded", "false");
            openingButton.focus();
            openingButton = null;
        }
    });

    // The server-validated provider href is unchanged. Selected dates are not
    // appended because the provider's URL/date parameter contract is unknown.
    const photoDialog = document.getElementById("pd-photos-dialog");
    const photoOpen = document.getElementById("pd-photos-open");
    const photoClose = document.getElementById("pd-photos-close");
    if (photoDialog && photoOpen && photoClose) {
        photoOpen.addEventListener("click", () => photoDialog.showModal());
        photoClose.addEventListener("click", () => photoDialog.close());
    }

    renderSelection();
})();


// PD-2A: favorite heart and collapsible description.
// Kept separate from the calendar code above so that
// neither feature depends on the other's elements.
(() => {
    "use strict";

    const page = document.querySelector("main.property-details");
    if (!page) return;

    // Same storage key and format as static/js/refine.js,
    // so a heart saved here also shows on /refine and /all/:country.
    const favoritesKey =
        `sparrow:favorites:v1:${page.dataset.siteKey || location.host}`;
    const heart = document.getElementById("pd-favorite");

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

    function syncHeart(selected) {
        const name = heart.dataset.propertyName || "property";
        heart.setAttribute("aria-pressed", String(selected));
        heart.setAttribute("aria-label", selected
            ? `Remove ${name} from favorites`
            : `Add ${name} to favorites`);
    }

    if (heart && heart.dataset.propertyId) {
        const id = heart.dataset.propertyId;
        syncHeart(readFavorites().has(id));

        heart.addEventListener("click", () => {
            const favorites = readFavorites();
            if (favorites.has(id)) favorites.delete(id);
            else favorites.add(id);
            syncHeart(favorites.has(id));
            try {
                localStorage.setItem(favoritesKey, JSON.stringify([...favorites]));
            } catch {
                // Storage is optional; the button state still updates.
            }
        });
    }

    // The server always sends the full description. JavaScript only
    // collapses it when it is taller than the collapsed height.
    const section = page.querySelector(".pd-description");
    const body = document.getElementById("pd-description-body");
    const toggle = document.getElementById("pd-show-more");
    if (!section || !body || !toggle) return;

    section.classList.add("is-collapsed");
    if (body.scrollHeight <= body.clientHeight + 4) {
        section.classList.remove("is-collapsed");
        return;
    }

    toggle.hidden = false;
    toggle.setAttribute("aria-expanded", "false");

    toggle.addEventListener("click", () => {
        const expand = section.classList.contains("is-collapsed");
        section.classList.toggle("is-collapsed", !expand);
        toggle.setAttribute("aria-expanded", String(expand));
        toggle.textContent = expand ? "Show less" : "Show more";
    });
})();