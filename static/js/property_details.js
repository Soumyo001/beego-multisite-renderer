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
    const summary = document.getElementById("pd-calendar-summary");
    const done = document.getElementById("pd-calendar-done");
    const doneLabel = document.getElementById("pd-calendar-done-label");
    const nightsLabel = document.getElementById("pd-calendar-nights");
    const prev = document.getElementById("pd-calendar-prev");
    const next = document.getElementById("pd-calendar-next");
    const openFull = document.getElementById("pd-calendar-open");

    if (!checkInButton || !checkOutButton || !checkInText ||
        !checkOutText || !status || !calendar || !months ||
        !prompt || !summary || !done || !doneLabel || !nightsLabel ||
        !prev || !next || !openFull) return;

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

    function nightCount(start, end) {
        const a = fromISO(start);
        const b = fromISO(end);
        if (!a || !b) return 0;
        const first = Date.UTC(a.getFullYear(), a.getMonth(), a.getDate());
        const last = Date.UTC(b.getFullYear(), b.getMonth(), b.getDate());
        return Math.round((last - first) / 86400000);
    }

    function renderSelection() {
        checkInText.textContent = selectedStart ? displayDate(selectedStart) : "Choose date";
        checkOutText.textContent = selectedEnd ? displayDate(selectedEnd) : "Choose date";
        status.textContent = selectedStart && selectedEnd
            ? "Dates selected for reference only. Confirm availability and final price with the provider."
            : "Select dates or continue without dates. Availability is not verified here.";
    }

    function renderFooter() {
        const complete = Boolean(draftStart && draftEnd);
        const partial = Boolean(draftStart && !draftEnd);
        const nights = complete ? nightCount(draftStart, draftEnd) : 0;

        prompt.textContent = partial ? "Choose check-out" :
            complete ? "Selected stay" : "Choose check-in and check-out";
        summary.textContent = complete
            ? `${displayDate(draftStart, true)} – ${displayDate(draftEnd, true)}`
            : partial ? `${displayDate(draftStart, true)} – Choose check-out`
                : "No dates selected";
        done.disabled = partial;
        doneLabel.textContent = draftStart ? "Continue" : "Continue without dates";
        nightsLabel.textContent = complete
            ? `${nights} night${nights === 1 ? "" : "s"}` : "";
    }

    function renderCalendar() {
        months.replaceChildren();
        const days = ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"];

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
    document.getElementById("pd-calendar-cancel").addEventListener("click", () => calendar.close());
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