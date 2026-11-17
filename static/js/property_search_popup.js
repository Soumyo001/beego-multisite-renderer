(() => {
    "use strict";

    const popup = document.querySelector(
        ".property-search-popup-container"
    );

    if (!popup ||
        popup.dataset.sparrowSearchReady === "true") {
        return;
    }

    const destination = popup.querySelector("#location_search");
    const dates = popup.querySelector("#date_picker");
    const guestDisplay = popup.querySelector("#guest_picker");
    const guestPanel = popup.querySelector("#search-guest-panel");
    const searchButton = popup.querySelector(".search-property-btn");

    const originalGuestCount =
        guestPanel?.querySelector(".guest-count");

    if (!destination || !dates || !guestDisplay ||
        !guestPanel || !originalGuestCount || !searchButton) {
        return;
    }

    const dateWrap = dates.closest(".datepicker-wrap");
    const guestWrap = guestDisplay.closest(".guest-container");

    if (!dateWrap || !guestWrap) return;

    popup.dataset.sparrowSearchReady = "true";
    popup.classList.add("sparrow-search-ready");

    let selectedStart = "";
    let selectedEnd = "";

    // Preserve the publisher's readonly date summary field.
    // A native modal hosts an accessible two-month range calendar.
    dates.setAttribute("aria-haspopup", "dialog");
    dates.setAttribute("aria-expanded", "false");
    dates.setAttribute("role", "button");
    dates.tabIndex = 0;

    const calendar = document.createElement("dialog");
    calendar.className = "sparrow-range-dialog";
    calendar.setAttribute("aria-labelledby", "sparrow-range-title");
    calendar.innerHTML = `
        <div class="sparrow-range-surface">
            <div class="sparrow-range-header">
                <h2 id="sparrow-range-title">When do you want to travel?</h2>
                <button type="button" class="sparrow-range-close"
                        aria-label="Close date calendar">×</button>
            </div>
            <div class="sparrow-range-toolbar">
                <button type="button" class="sparrow-range-prev"
                        aria-label="Previous month">‹</button>
                <div class="sparrow-range-months"></div>
                <button type="button" class="sparrow-range-next"
                        aria-label="Next month">›</button>
            </div>
            <div class="sparrow-range-footer">
                <span class="sparrow-range-summary" role="status"
                      aria-live="polite"></span>
                <div class="sparrow-range-actions">
                    <button type="button" class="sparrow-range-clear">Clear</button>
                    <button type="button" class="sparrow-range-apply">Continue</button>
                </div>
            </div>
        </div>
    `;
    document.body.append(calendar);

    const monthsRoot = calendar.querySelector(".sparrow-range-months");
    const previousMonth = calendar.querySelector(".sparrow-range-prev");
    const nextMonth = calendar.querySelector(".sparrow-range-next");
    const summary = calendar.querySelector(".sparrow-range-summary");
    const applyDates = calendar.querySelector(".sparrow-range-apply");
    const clearDates = calendar.querySelector(".sparrow-range-clear");

    const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    let pendingStart = "";
    let pendingEnd = "";
    let visibleMonth = new Date();
    visibleMonth.setDate(1);

    function isoDate(year, month, day) {
        return `${year}-${String(month + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
    }

    function todayISO() {
        const today = new Date();
        return isoDate(today.getFullYear(), today.getMonth(), today.getDate());
    }

    function utcDayCount(start, end) {
        const utc = (iso) => {
            const [year, month, day] = iso.split("-").map(Number);
            return Date.UTC(year, month - 1, day);
        };
        return Math.round((utc(end) - utc(start)) / 86400000);
    }

    function monthPanel(year, month) {
        const section = document.createElement("section");
        section.className = "sparrow-range-month";

        const title = document.createElement("h3");
        title.textContent = new Intl.DateTimeFormat("en-US", {
            month: "long", year: "numeric"
        }).format(new Date(year, month, 1));
        section.append(title);

        const grid = document.createElement("div");
        grid.className = "sparrow-range-days";
        section.append(grid);

        for (const weekday of weekdays) {
            const label = document.createElement("span");
            label.className = "sparrow-range-weekday";
            label.textContent = weekday;
            grid.append(label);
        }

        const firstWeekday = new Date(year, month, 1).getDay();
        for (let offset = 0; offset < firstWeekday; offset++) {
            const empty = document.createElement("span");
            empty.setAttribute("aria-hidden", "true");
            grid.append(empty);
        }

        const days = new Date(year, month + 1, 0).getDate();
        for (let day = 1; day <= days; day++) {
            const iso = isoDate(year, month, day);
            const button = document.createElement("button");
            button.type = "button";
            button.className = "sparrow-range-day";
            button.textContent = String(day);
            button.dataset.date = iso;
            button.disabled = iso < todayISO();
            button.setAttribute("aria-label", new Intl.DateTimeFormat(
                "en-US", { weekday: "long", month: "long", day: "numeric", year: "numeric" }
            ).format(new Date(year, month, day)));
            button.setAttribute("aria-pressed", String(
                iso === pendingStart || iso === pendingEnd
            ));

            if (iso === pendingStart || iso === pendingEnd) {
                button.classList.add("sparrow-range-selected");
            } else if (pendingStart && pendingEnd &&
                       iso > pendingStart && iso < pendingEnd) {
                button.classList.add("sparrow-range-between");
            }

            button.addEventListener("click", () => {
                if (!pendingStart || pendingEnd || iso <= pendingStart) {
                    pendingStart = iso;
                    pendingEnd = "";
                } else {
                    pendingEnd = iso;
                }
                renderCalendar(iso);
            });
            grid.append(button);
        }
        return section;
    }

    function renderCalendar(focusISO = "") {
        const year = visibleMonth.getFullYear();
        const month = visibleMonth.getMonth();
        const second = new Date(year, month + 1, 1);
        monthsRoot.replaceChildren(
            monthPanel(year, month),
            monthPanel(second.getFullYear(), second.getMonth())
        );

        const today = new Date();
        previousMonth.disabled =
            year < today.getFullYear() ||
            (year === today.getFullYear() && month <= today.getMonth());

        applyDates.disabled = !pendingStart || !pendingEnd;

        if (pendingStart && pendingEnd) {
            const nights = utcDayCount(pendingStart, pendingEnd);
            summary.textContent =
                `${pendingStart} – ${pendingEnd} · ` +
                `${nights} ${nights === 1 ? "night" : "nights"}`;
        } else {
            summary.textContent = pendingStart
                ? "Now choose your checkout date"
                : "Choose your check-in and checkout dates";
        }

        if (focusISO) {
            const chosen = monthsRoot.querySelector(
                `.sparrow-range-day[data-date="${focusISO}"]`
            );
            chosen?.focus();
        }
    }

    function openDates() {
        const today = new Date();
        pendingStart = selectedStart;
        pendingEnd = selectedEnd;
        visibleMonth = pendingStart
            ? new Date(`${pendingStart}T12:00:00`)
            : new Date(today.getFullYear(), today.getMonth(), 1);
        visibleMonth.setDate(1);
        renderCalendar();
        dates.setAttribute("aria-expanded", "true");
        calendar.showModal();
        calendar.querySelector(".sparrow-range-close").focus();
    }

    previousMonth.addEventListener("click", () => {
        visibleMonth.setMonth(visibleMonth.getMonth() - 1);
        renderCalendar();
    });
    nextMonth.addEventListener("click", () => {
        visibleMonth.setMonth(visibleMonth.getMonth() + 1);
        renderCalendar();
    });
    calendar.querySelector(".sparrow-range-close")
        .addEventListener("click", () => calendar.close());

    // Prevent publisher's document-level click/escape listeners
    // from closing the underlying search popup while using the modal.
    calendar.addEventListener("click", (event) => {
        event.stopPropagation();
        if (event.target === calendar) calendar.close();
    });
    calendar.addEventListener("keydown", event => event.stopPropagation());
    calendar.addEventListener("close", () => {
        dates.setAttribute("aria-expanded", "false");
    });

    applyDates.addEventListener("click", () => {
        if (!pendingStart || !pendingEnd || pendingEnd <= pendingStart) return;
        selectedStart = pendingStart;
        selectedEnd = pendingEnd;
        dates.value = `${formatDate(selectedStart)} – ${formatDate(selectedEnd)}`;
        setError("");
        calendar.close();
    });

    clearDates.addEventListener("click", () => {
        selectedStart = "";
        selectedEnd = "";
        dates.value = "";
        setError("");
        calendar.close();
    });

    // Retain the original count span for compatibility
    // with publisher code, but show an editable input.
    const guestNumber = document.createElement("input");

    guestNumber.type = "number";
    guestNumber.className = "sparrow-guest-number";
    guestNumber.min = "1";
    guestNumber.max = "20";
    guestNumber.step = "1";
    guestNumber.value = "2";
    guestNumber.setAttribute(
        "aria-label",
        "Number of guests"
    );

    originalGuestCount.hidden = true;
    originalGuestCount.insertAdjacentElement(
        "afterend",
        guestNumber
    );

    const decrease = guestPanel.querySelector(".decrease-nav");
    const increase = guestPanel.querySelector(".increase-nav");
    const guestClear = guestPanel.querySelector(".guest-clear-btn");
    const guestSave = guestPanel.querySelector(".guest-save-btn");

    if (!decrease || !increase || !guestClear || !guestSave) {
        return;
    }

    // The original publisher CSS draws the +/- symbols.
    // We do not add duplicate symbols using textContent.

    // Keep validation feedback outside the horizontal
    // search row to prevent it becoming another flex item.
    const error = document.createElement("p");
    error.className = "sparrow-search-error";
    error.setAttribute("role", "alert");
    error.hidden = true;

    popup.querySelector(".property-search-content")?.insertAdjacentElement(
        "afterend", error
    );

    function setError(message) {
        error.textContent = message;
        error.hidden = !message;
    }

    function closeDates() {
        if (calendar.open) calendar.close();
        dates.setAttribute("aria-expanded", "false");
    }

    function closeGuests() {
        guestPanel.classList.remove("sparrow-open");
        guestDisplay.setAttribute("aria-expanded", "false");
    }

    function guestCount() {
        const count = Number(guestNumber.value);

        if (!Number.isInteger(count) ||
            count < 1 || count > 20) {
            return null;
        }

        return count;
    }

    function setGuests(count) {
        const value = Math.max(1, Math.min(20, count));

        guestNumber.value = String(value);

        guestDisplay.value =
            `${value} ${value === 1 ? "Person" : "Persons"}`;

        setError("");
    }

    function formatDate(value) {
        const date = new Date(`${value}T00:00:00Z`);

        return date.toLocaleDateString("en-US", {
            month: "short",
            day: "2-digit",
            timeZone: "UTC"
        });
    }

    function validateDates(start, end) {
        if ((start && !end) || (!start && end)) {
            return "Choose both check-in and checkout dates.";
        }

        if (start && end <= start) {
            return "Checkout must be after check-in.";
        }

        return "";
    }

    // Open the range calendar from the publisher's readonly input.
    dates.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopImmediatePropagation();
        closeGuests();
        if (!calendar.open) openDates();
    }, true);

    dates.addEventListener("keydown", (event) => {
        if ((event.key !== "Enter" && event.key !== " ") ||
            event.isComposing || event.repeat) {
            return;
        }

        event.preventDefault();
        event.stopPropagation();

        // Enter submits when a complete date range is selected.
        // Space always allows keyboard users to reopen the calendar.
        if (event.key === "Enter" && selectedStart && selectedEnd) {
            submitSearch();
            return;
        }

        dates.click();
    });

    // Open the existing guest panel.
    guestDisplay.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopImmediatePropagation();

        const shouldOpen =
            !guestPanel.classList.contains("sparrow-open");

        closeDates();

        guestPanel.classList.toggle(
            "sparrow-open",
            shouldOpen
        );

        guestDisplay.setAttribute(
            "aria-expanded",
            String(shouldOpen)
        );

        if (shouldOpen) guestNumber.focus();
    }, true);

    guestDisplay.addEventListener("keydown", (event) => {
        if ((event.key !== "Enter" && event.key !== " ") ||
            event.isComposing || event.repeat) {
            return;
        }

        event.preventDefault();
        event.stopImmediatePropagation();

        // Enter submits through the existing search validation.
        // Space remains available to open the guest picker.
        if (event.key === "Enter") {
            submitSearch();
        } else {
            guestDisplay.click();
        }
    }, true);

    // Connect the original +/- buttons to the number input.
    for (const [button, amount] of [
        [decrease, -1],
        [increase, 1]
    ]) {
        button.addEventListener("click", (event) => {
            event.preventDefault();
            event.stopImmediatePropagation();

            setGuests((guestCount() || 2) + amount);

            // After a pointer selection, focus the guest display
            // so Enter searches without opening a mobile keyboard.
            // Keyboard activation retains the +/- button behavior.
            if (event.detail > 0) {
                guestDisplay.focus({ preventScroll: true });
            }
        }, true);
    }

    guestNumber.addEventListener("change", () => {
        const value = guestCount();

        if (value === null) {
            setError(
                "Guests must be a whole number between 1 and 20."
            );
            return;
        }

        setGuests(value);
    });

    guestClear.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopImmediatePropagation();

        // Restore the original default of two guests.
        setGuests(2);
    }, true);

    guestSave.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopImmediatePropagation();

        const value = guestCount();

        if (value === null) {
            setError(
                "Guests must be a whole number between 1 and 20."
            );
            return;
        }

        setGuests(value);
        closeGuests();

        // Return focus to the visible guest field.
        // Enter can now submit the complete search.
        guestDisplay.focus({ preventScroll: true });
    }, true);

    // UI-L2: visual transition before navigating to the existing CSR page.
    let searchNavigationPending = false;
    let searchLoadingOverlay = null;

    function navigateWithSearchLoading(search, targetURL) {
        if (searchNavigationPending) return;
        searchNavigationPending = true;
        searchButton.disabled = true;
        searchButton.setAttribute("aria-busy", "true");

        const overlay = document.createElement("section");
        overlay.className = "sparrow-search-transition";
        overlay.setAttribute("aria-label", "Loading search results");

        const inner = document.createElement("div");
        inner.className = "sparrow-search-transition-inner";
        inner.innerHTML = `
            <p class="sparrow-search-transition-status" role="status">
                Preparing your search...
            </p>
            <h2 class="sparrow-search-transition-heading"></h2>
            <div class="sparrow-search-transition-grid" aria-hidden="true"></div>
        `;
        inner.querySelector(".sparrow-search-transition-heading").textContent =
            `Finding a Place to Stay in ${search}`;

        const skeletonCard = `
            <div class="sparrow-search-transition-card">
                <div class="sparrow-search-transition-photo sparrow-search-transition-shimmer"></div>
                <div class="sparrow-search-transition-details">
                    <div class="sparrow-search-transition-line sparrow-search-transition-shimmer" style="width:70%"></div>
                    <div class="sparrow-search-transition-line sparrow-search-transition-shimmer" style="width:90%"></div>
                    <div class="sparrow-search-transition-line sparrow-search-transition-shimmer" style="width:83%"></div>
                    <div class="sparrow-search-transition-line sparrow-search-transition-shimmer" style="width:42%"></div>
                </div>
            </div>
        `;
        inner.querySelector(".sparrow-search-transition-grid").innerHTML =
            skeletonCard.repeat(8);
        overlay.append(inner);

        // Preserve the publisher header and its centered content width.
        const nav = document.getElementById("main-navigation");
        if (nav) {
            const rect = nav.getBoundingClientRect();
            if (rect.width > 0 && rect.bottom > 0) {
                overlay.style.top = `${Math.max(0, rect.bottom)}px`;
                inner.style.width = `${rect.width}px`;
                inner.style.maxWidth = "none";
                inner.style.marginLeft = `${rect.left}px`;
                inner.style.marginRight = "0";
                inner.style.paddingInline = "0";
            }
        }

        document.body.append(overlay);
        searchLoadingOverlay = overlay;

        // Permit a paint; the brief minimum is visual feedback, not server time.
        if (document.visibilityState === "hidden") {
            window.location.assign(targetURL);
        } else {
            requestAnimationFrame(() => {
                window.setTimeout(() => window.location.assign(targetURL), 180);
            });
        }
    }

    // Restore the old page if a visitor returns via the back-forward cache.
    window.addEventListener("pageshow", event => {
        if (!event.persisted || !searchNavigationPending) return;
        searchLoadingOverlay?.remove();
        searchLoadingOverlay = null;
        searchNavigationPending = false;
        searchButton.disabled = false;
        searchButton.removeAttribute("aria-busy");
    });

    // Convert selections into the exact URL format
    // expected by the existing /refine page.
    function submitSearch() {
        const count = guestCount();

        if (count === null) {
            setError(
                "Guests must be a whole number between 1 and 20."
            );
            return;
        }

        const dateError = validateDates(
            selectedStart,
            selectedEnd
        );

        if (dateError) {
            setError(dateError);
            return;
        }

        const search = destination.value.trim();

        if (!search) {
            setError("Enter a destination to search.");
            destination.focus();
            return;
        }

        const params = new URLSearchParams({
            search: search.toLowerCase(),
            pax: String(count),
            order: "1"
        });

        if (selectedStart && selectedEnd) {
            params.set("dateStart", selectedStart);
            params.set("dateEnd", selectedEnd);
        }

        // Relative URL: preserve the current host and port.
        navigateWithSearchLoading(
            search,
            `/refine?${params.toString()}`
        );
    }

    searchButton.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopImmediatePropagation();

        submitSearch();
    }, true);

    // Enter in an editable search field uses the same
    // validation and loading flow as the Search button.
    //
    // The readonly date and guest display fields retain
    // their existing Enter-key behavior for opening pickers.
    for (const input of [destination, guestNumber]) {
        input.addEventListener("keydown", (event) => {
            if (event.key !== "Enter" ||
                event.isComposing || event.repeat) {
                return;
            }

            event.preventDefault();
            event.stopPropagation();

            submitSearch();
        });
    }

    // Close pickers when clicking elsewhere.
    document.addEventListener("click", (event) => {
        if (!dateWrap.contains(event.target)) {
            closeDates();
        }

        if (!guestWrap.contains(event.target)) {
            closeGuests();
        }
    });

    document.addEventListener("keydown", (event) => {
        if (event.key === "Escape") {
            closeDates();
            closeGuests();
        }
    });

    setGuests(2);
})();
