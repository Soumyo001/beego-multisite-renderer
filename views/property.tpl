<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Property.Name}} | {{.SiteName}}</title>
  <link rel="canonical" href="{{.CanonicalURL}}">

  {{if .GlobalCSS}}<link rel="stylesheet" href="{{.GlobalCSS}}">{{end}}
  {{if .TemplateCSS}}<link rel="stylesheet" href="{{.TemplateCSS}}">{{end}}
  <link rel="stylesheet" href="/static/css/property_details.css">
  <link rel="stylesheet" href="/static/css/property_search_popup.css">
  {{.WidgetHTML}}
</head>
<body class="property-route">
  {{.HeaderHTML}}

  <main class="property-details">
    <div class="pd-container">
      <nav class="pd-crumbs" aria-label="Breadcrumb">
        <a href="/">Vacation Rentals</a> ›
        <span>{{.Property.Location.Country}}</span> ›
        <span>{{.Property.Location.State}}</span> ›
        <span>{{.Property.Location.City}}</span>
      </nav>

      <h1>{{.Property.Name}} | {{.Property.PropertyType}} in {{.Property.Location.City}}</h1>

      <div class="pd-facts">
        <span>{{printf "%.1f" .Property.Rating}} ({{.Property.ReviewCount}} reviews)</span>
        {{if .PetFriendly}}<span>Pet Friendly</span>{{end}}
        {{if .HasBedrooms}}<span>{{.Bedrooms}} Bedrooms</span>{{end}}
        {{if .HasBathrooms}}<span>{{.Bathrooms}} Bathrooms</span>{{end}}
        <span>{{.Property.MaxGuests}} Guests</span>
      </div>

      <div class="pd-columns">
        <div class="pd-primary">
          <section class="pd-gallery" aria-label="Property photographs">
            {{range $i, $photo := .Photos}}
              <img src="{{$photo}}"
                   alt="Illustrative property photograph"
                   {{if eq $i 0}}loading="eager" fetchpriority="high"{{else}}loading="lazy"{{end}}>
            {{else}}
              <p>Photographs are not available.</p>
            {{end}}
            {{if .Photos}}
              <button type="button" class="pd-photos-open" id="pd-photos-open">
                View More Photos
              </button>
            {{end}}
          </section>

          {{if .Photos}}
            <dialog id="pd-photos-dialog" class="pd-photo-dialog" aria-labelledby="pd-photos-title">
              <div class="pd-dialog-bar">
                <h2 id="pd-photos-title">Photos of {{.Property.Name}}</h2>
                <button type="button" id="pd-photos-close">Close</button>
              </div>
              <div class="pd-photo-grid">
                {{range .Photos}}
                  <img src="{{.}}" alt="Illustrative property photograph" loading="lazy">
                {{end}}
              </div>
            </dialog>
          {{end}}

          <section class="pd-description">
            <h2>
              {{if .HasBedrooms}}{{.Bedrooms}} Bedroom {{end}}{{.Property.PropertyType}} in {{.Property.Location.City}}
            </h2>
            {{if .HasDetails}}
              <p>{{.Details}}</p>
            {{else}}
              <p>Property details are not available.</p>
            {{end}}
            {{if .Property.Amenities}}
              <h2>Amenities</h2>
              <ul>
                {{range .Property.Amenities}}<li>{{.}}</li>{{end}}
              </ul>
            {{end}}
          </section>
        </div>

        <aside class="pd-booking" aria-label="Property availability information">
          <h2>Check {{.Property.PropertyType}} Availability</h2>

          <div class="pd-date-grid" role="group" aria-label="Select stay dates">
            <div class="pd-date-field">
              <span id="pd-checkin-label">Check-in</span>
              <button type="button" id="pd-checkin" class="pd-date-trigger"
                      aria-labelledby="pd-checkin-label pd-checkin-text"
                      aria-haspopup="dialog" aria-controls="pd-calendar-dialog"
                      aria-expanded="false">
                <span id="pd-checkin-text">Choose date</span>
              </button>
            </div>
            <div class="pd-date-field">
              <span id="pd-checkout-label">Check-out</span>
              <button type="button" id="pd-checkout" class="pd-date-trigger"
                      aria-labelledby="pd-checkout-label pd-checkout-text"
                      aria-haspopup="dialog" aria-controls="pd-calendar-dialog"
                      aria-expanded="false">
                <span id="pd-checkout-text">Choose date</span>
              </button>
            </div>
          </div>
          <p id="pd-date-status" class="pd-date-status" role="status" aria-live="polite">
            Select dates or continue without dates.
          </p>
          <button type="button" class="pd-calendar-button" id="pd-calendar-open">
            Show Full Calendar
          </button>

          {{if gt .Property.PriceFrom 0}}
            <p class="pd-rate">
              <span>Illustrative nightly rate from</span>
              <strong>{{.Property.Currency}} {{.Property.PriceFrom}}</strong>
            </p>
          {{else}}
            <p class="pd-rate">Confirm price with the provider.</p>
          {{end}}

          <p class="pd-note">Prices and property facts are illustrative; the dates above do not check live inventory.</p>

          {{if .BookingURL}}
            <a id="pd-check-provider" href="{{.BookingURL}}"
               class="pd-action" rel="noopener noreferrer">{{if .IsDemo}}View Other Hotel{{else}}Check Availability{{end}}</a>
            {{if .IsDemo}}
              <p class="pd-redirect-note">This is an illustrative demo listing. The link opens a different, real hotel in the same destination. The displayed photos, details, and prices do not describe that hotel.</p>
            {{else}}
              <p class="pd-redirect-note">You will be redirected to the provider to confirm dates, availability and final pricing. Selected dates are not sent automatically.</p>
            {{end}}
          {{else}}
            <p class="pd-booking-unavailable">Live booking information is not connected for this demonstration listing.</p>
          {{end}}

          <dialog id="pd-calendar-dialog" class="pd-calendar-dialog"
                  aria-labelledby="pd-calendar-title">
            <div class="pd-calendar-heading">
              <h2 id="pd-calendar-title">Select your dates</h2>
              <button type="button" id="pd-calendar-close"
                      aria-label="Close calendar">×</button>
            </div>
            <div class="pd-calendar-content">
              <div class="pd-calendar-toolbar">
                <button type="button" id="pd-calendar-prev"
                        aria-label="Previous month">‹</button>
                <strong id="pd-calendar-prompt" aria-live="polite">
                  Choose check-in and checkout
                </strong>
                <button type="button" id="pd-calendar-next"
                        aria-label="Next month">›</button>
              </div>
              <div id="pd-calendar-months" class="pd-calendar-months"></div>
            </div>
            <div class="pd-calendar-footer">
              <button type="button" id="pd-calendar-clear">Clear</button>
              <button type="button" id="pd-calendar-done">Apply dates</button>
            </div>
          </dialog>
        </aside>
      </div>

      <section class="pd-nearby" aria-label="Nearby vacation rentals">
        <h2>Vacation Rentals Near {{.Property.Location.City}}</h2>
        <div class="pd-related">
          {{range .Related}}
            <article class="pd-card">
              <a class="pd-card-main" href="{{.URL}}">
                <div class="pd-card-picture">
                  {{if .Image}}
                    <img src="{{.Image}}" alt="Illustrative photograph of {{.Name}}" loading="lazy">
                  {{end}}
                  <span class="pd-card-price">From {{.Price}}</span>
                </div>
                <div class="pd-card-content">
                  <p class="pd-card-rating">{{printf "%.1f" .Rating}} ({{.Reviews}} reviews)</p>
                  <h3>{{.Name}}</h3>
                  <p>{{.Location}}</p>
                  <span class="pd-card-link">View details</span>
                </div>
              </a>
            </article>
          {{else}}
            <p>No nearby properties are currently listed.</p>
          {{end}}
        </div>
      </section>
    </div>
  </main>

  {{.FooterHTML}}
  {{if .GlobalJS}}<script src="{{.GlobalJS}}" defer></script>{{end}}
  <script src="/static/js/property_search_popup.js" defer></script>
  <script src="/static/js/property_details.js" defer></script>
</body>
</html>