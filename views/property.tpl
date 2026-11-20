<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Property.Name}} | {{.SiteName}}</title>
  <link rel="canonical" href="{{.CanonicalURL}}">

  {{if .GlobalCSS}}
  <link rel="stylesheet" href="{{.GlobalCSS}}">
  {{end}}
  {{if .TemplateCSS}}
  <link rel="stylesheet" href="{{.TemplateCSS}}">
  {{end}}

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

      <h1>
        {{.Property.Name}} |
        {{.Property.PropertyType}} in {{.Property.Location.City}}
      </h1>

      <div class="pd-facts">
        <span>
          {{.Property.Rating}} ({{.Property.ReviewCount}} reviews)
        </span>
        {{if .PetFriendly}}<span>Pet Friendly</span>{{end}}
        {{if .HasBedrooms}}<span>{{.Bedrooms}} bedrooms</span>{{end}}
        {{if .HasBathrooms}}<span>{{.Bathrooms}} bathrooms</span>{{end}}
        <span>{{.Property.MaxGuests}} guests</span>
      </div>

      <div class="pd-columns">
        <div class="pd-primary">
          <section class="pd-gallery" aria-label="Property photographs">
            {{range $i, $photo := .Photos}}
              <img src="{{$photo}}"
                   alt="Illustrative property photograph"
                   {{if eq $i 0}}
                   loading="eager" fetchpriority="high"
                   {{else}}
                   loading="lazy"
                   {{end}}>
            {{else}}
              <p>Photographs are not available.</p>
            {{end}}
          </section>

          <section class="pd-description">
            <h2>About {{.Property.Name}}</h2>

            {{if .HasDetails}}
              <p>{{.Details}}</p>
            {{else}}
              <p>Property details are not available.</p>
            {{end}}

            {{if .Property.Amenities}}
              <h2>Amenities</h2>
              <ul>
                {{range .Property.Amenities}}
                  <li>{{.}}</li>
                {{end}}
              </ul>
            {{end}}
          </section>
        </div>

        <aside class="pd-booking"
               aria-label="Property availability information">
          <h2>Check Availability</h2>

          <p>
            From
            <strong>
              {{.Property.Currency}} {{.Property.PriceFrom}}
            </strong>
            per night
          </p>

          <p>
            Rates are illustrative. Confirm dates,
            availability and final pricing with the provider.
          </p>

          {{if .BookingURL}}
            <a href="{{.BookingURL}}"
               rel="noopener noreferrer"
               class="pd-action">
              View provider availability
            </a>
          {{else}}
            <p>
              Live booking is not connected for this
              demonstration listing.
            </p>
          {{end}}
        </aside>
      </div>

      <section class="pd-nearby"
               aria-label="Nearby vacation rentals">
        <h2>Vacation Rentals Near This Location</h2>

        <div class="pd-related">
          {{range .Related}}
            <article class="pd-card">
              <a href="{{.URL}}">
                {{if .Image}}
                  <img src="{{.Image}}"
                       alt="Illustrative photograph of {{.Name}}"
                       loading="lazy">
                {{end}}
                <h3>{{.Name}}</h3>
              </a>

              <p>{{.Rating}} ({{.Reviews}} reviews)</p>
              <p>{{.Location}}</p>
              <p>From {{.Price}} per night</p>
            </article>
          {{else}}
            <p>No nearby properties are currently listed.</p>
          {{end}}
        </div>
      </section>
    </div>
  </main>

  {{.FooterHTML}}

  {{if .GlobalJS}}
  <script src="{{.GlobalJS}}" defer></script>
  {{end}}
  <script src="/static/js/property_search_popup.js" defer></script>
</body>
</html>