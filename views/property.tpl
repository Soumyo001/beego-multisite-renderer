<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Property.Name}} | {{.SiteName}}</title>
  <link rel="canonical" href="{{.CanonicalURL}}">

  {{if .GlobalCSS}}<link rel="stylesheet" href="{{.GlobalCSS}}">{{end}}
  {{if .TemplateCSS}}<link rel="stylesheet" href="{{.TemplateCSS}}">{{end}}
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Marcellus&display=swap">
  <link rel="stylesheet" href="/static/css/property_details.css">
  <link rel="stylesheet" href="/static/css/property_search_popup.css">
  {{.WidgetHTML}}
</head>
<body class="property-route">
  {{.HeaderHTML}}

  <main class="property-details" data-site-key="{{.SiteKey}}">
    <div class="pd-container">
      <nav class="pd-crumbs" aria-label="Breadcrumb">
        <ol>
          {{range .Crumbs}}<li><a href="{{.URL}}">{{.Label}}</a></li>{{end}}
        </ol>
      </nav>

      <h1>{{.Property.Name}} | {{.Property.PropertyType}} in {{.Property.Location.City}}</h1>

      <div class="pd-facts">
        <span class="pd-fact pd-fact-rating">
          <span class="pd-rating-badge" aria-hidden="true">
            <svg viewBox="0 0 24 24" focusable="false"><path d="M7 10v11M15 5.9 14 10h5.8a2 2 0 0 1 1.9 2.5l-2.3 8A2 2 0 0 1 17.4 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.8a2 2 0 0 0 1.8-1.1L12 2a3.1 3.1 0 0 1 3 3.9z"/></svg>
          </span>
          <strong>{{printf "%.1f" .Property.Rating}}</strong>
          ({{.Property.ReviewCount}} Reviews)
        </span>
        <span class="pd-fact-divider" aria-hidden="true">|</span>
        {{if .HasBedrooms}}
          <span class="pd-fact">
            <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M2 4v16M2 8h18a2 2 0 0 1 2 2v10M2 17h20M6 8v9"/></svg>
            {{.Bedrooms}} {{if eq .Bedrooms 1}}Bedroom{{else}}Bedrooms{{end}}
          </span>
        {{end}}
        {{if .HasBathrooms}}
          <span class="pd-fact">
            <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M9 6 6.5 3.5a1.5 1.5 0 0 0-1-.5C4.7 3 4 3.7 4 4.5V17a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-5M2 12h20M7 19v2M17 19v2"/></svg>
            {{.Bathrooms}} {{if le .Bathrooms 1.0}}Bathroom{{else}}Bathrooms{{end}}
          </span>
        {{end}}
        <span class="pd-fact">
          <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M22 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/><circle cx="9" cy="7" r="4"/></svg>
          {{.Property.MaxGuests}} Guests
        </span>
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
              <button type="button" class="pd-favorite" id="pd-favorite"
                      data-property-id="{{.Property.ID}}"
                      data-property-name="{{.Property.Name}}"
                      aria-pressed="false"
                      aria-label="Add {{.Property.Name}} to favorites">
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
              </button>
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

          <section class="pd-description" aria-labelledby="pd-description-title">
            <h2 id="pd-description-title">
              {{if .HasBedrooms}}{{.Bedrooms}} Bedroom {{end}}{{.Property.PropertyType}} in {{if .Property.Location.Area}}{{.Property.Location.Area}}, {{end}}{{.Property.Location.City}}
            </h2>
            <div class="pd-description-body" id="pd-description-body">
              {{if .HasDetails}}
                <p>{{.Details}}</p>
              {{else}}
                <p>Property details are not available.</p>
              {{end}}
              {{if or .Property.Amenities .PetFriendly}}
                <p class="pd-amenities-title">Amenities included with rental:</p>
                <ul class="pd-amenities">
                  {{range .Property.Amenities}}<li>{{.}}</li>{{end}}
                  {{if .PetFriendly}}<li>Pet Friendly</li>{{end}}
                </ul>
              {{end}}
            </div>
            <button type="button" class="pd-show-more" id="pd-show-more"
                    aria-controls="pd-description-body"
                    aria-expanded="true" hidden>Show more</button>
          </section>
        </div>

        <div class="pd-sidebar">
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

          <div class="pd-rate" id="pd-rate-from">
            <span class="pd-rate-label">Nightly rates from:</span>
            <strong class="pd-rate-price">{{if .PriceText}}{{.PriceText}}{{else}}Check provider{{end}}</strong>
          </div>

          {{if gt .Property.PriceFrom 0}}
            <dl class="pd-rate-rows" id="pd-rate-rows" hidden
                data-price="{{.Property.PriceFrom}}"
                data-currency="{{.Property.Currency}}">
              <div><dt>Price per night</dt><dd id="pd-rate-night"></dd></div>
              <div><dt>Total Price</dt><dd id="pd-rate-total"></dd></div>
            </dl>
          {{end}}

          <div class="pd-rate-links">
            <button type="button" class="pd-price-details" id="pd-price-open"
                    aria-haspopup="dialog" aria-controls="pd-price-dialog">Price Details</button>
          </div>

          {{if .BookingURL}}
            <a id="pd-check-provider" href="{{.BookingURL}}"
               class="pd-action" rel="noopener noreferrer">{{if .IsDemo}}View Other Hotel{{else}}Check Availability{{end}}</a>
            {{if .ProviderLogo}}
              <p class="pd-redirect-note">
                You will be redirected to
                <img class="pd-provider-logo" src="{{.ProviderLogo}}" alt="{{.Property.ProviderName}}">
              </p>
            {{end}}
          {{else}}
            <p class="pd-booking-unavailable">Provider link unavailable</p>
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

        {{if .FitLabel}}
          <p class="pd-fit">
            <span>Not the right fit? Check out our other properties in <a href="{{.FitURL}}">{{.FitLabel}}</a></span>
            <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><circle cx="14" cy="9" r="6"/><path d="M9.8 13.2 3 20M14 12.2s-2.4-2.1-2.4-3.6a2.4 2.4 0 0 1 4.8 0c0 1.5-2.4 3.6-2.4 3.6z"/></svg>
          </p>
        {{end}}

        <dialog id="pd-price-dialog" class="pd-price-dialog" aria-labelledby="pd-price-title">
          <div class="pd-price-heading">
            <h2 id="pd-price-title">Price Details</h2>
            <button type="button" id="pd-price-close" aria-label="Close price details">×</button>
          </div>
          <p>Price excludes fees and taxes. <button type="button" class="pd-enter-dates" data-open-calendar>Enter dates</button>, then confirm the full price with the provider.</p>
        </dialog>
        </div>
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