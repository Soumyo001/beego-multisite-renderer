package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"beego-multisite-renderer/models"
	"beego-multisite-renderer/services"

	beego "github.com/beego/beego/v2/server/web"
)

type propertyDetailsDependencies struct {
	reader services.PropertyDetailsReader
	search services.PropertySearcher
}

var detailsDependencies propertyDetailsDependencies

func SetPropertyDetailsDependencies(
	reader services.PropertyDetailsReader,
	search services.PropertySearcher,
) {
	detailsDependencies = propertyDetailsDependencies{
		reader: reader,
		search: search,
	}
}

type PropertyDetailsController struct {
	beego.Controller
}

// propertyCrumb is one breadcrumb link above the property title.
type propertyCrumb struct {
	Label string
	URL   string
}

// refineSearchURL builds the existing /refine search URL
// used by the header search popup.
func refineSearchURL(term string) string {
	query := url.Values{}
	query.Set("search", term)
	query.Set("pax", "2")
	query.Set("order", "1")
	return "/refine?" + query.Encode()
}

// propertyCrumbs mirrors the reference breadcrumb:
// "<Area> Rentals > Country > State > City > Area".
// Country links to the SSR country page; the other
// levels link to the existing /refine search.
func propertyCrumbs(location models.PropertyLocation) []propertyCrumb {
	root := location.Area
	if root == "" {
		root = location.City
	}

	crumbs := make([]propertyCrumb, 0, 5)
	if root != "" {
		crumbs = append(crumbs, propertyCrumb{
			Label: root + " Rentals",
			URL:   refineSearchURL(root),
		})
	}

	if location.Country != "" {
		slug := strings.ToLower(
			strings.Join(strings.Fields(location.Country), "-"),
		)
		crumbURL := refineSearchURL(location.Country)
		if _, ok := countryFromSlug(slug); ok {
			crumbURL = "/all/" + slug
		}
		crumbs = append(crumbs, propertyCrumb{
			Label: location.Country,
			URL:   crumbURL,
		})
	}

	seen := map[string]bool{location.Country: true}
	for _, label := range []string{
		location.State, location.City, location.Area,
	} {
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		crumbs = append(crumbs, propertyCrumb{
			Label: label,
			URL:   refineSearchURL(label),
		})
	}

	return crumbs
}

// formatNightlyPrice renders "USD 1,206" for the booking sidebar.
func formatNightlyPrice(currency string, amount int) string {
	if amount <= 0 {
		return ""
	}

	digits := strconv.Itoa(amount)
	var grouped strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(digit)
	}

	return strings.TrimSpace(currency + " " + grouped.String())
}

// providerLogo returns the local partner logo for the two
// providers accepted by safeBooking.
func providerLogo(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "booking.com":
		return "/static/img/partners/booking.svg"
	case "expedia", "expedia.com":
		return "/static/img/partners/expedia.svg"
	default:
		return ""
	}
}

// newListingCard builds the shared /refine-style SSR card.
// It is used by /all/:country and the property-details
// "Vacation Rentals Near" section, so both render identical cards.
func newListingCard(p models.Property) countryCard {
	detailParts := make([]string, 0, len(p.Amenities)+1)

	if p.SizeSqFt > 0 {
		detailParts = append(
			detailParts,
			fmt.Sprintf("%d ft²", p.SizeSqFt),
		)
	}

	detailParts = append(detailParts, p.Amenities...)

	price := "Check provider for price"
	if p.PriceFrom > 0 {
		price = fmt.Sprintf(
			"From %s %d", p.Currency, p.PriceFrom,
		)
	}

	return countryCard{
		ID:         p.ID,
		DetailsURL: "/property/" + url.PathEscape(p.Slug) + "/" + url.PathEscape(p.ID),
		Name:       p.Name,
		Type:       p.PropertyType,
		Photos:     sizedPhotos(safeGalleryPhotos(p.Images), cardPhotoWidth),
		Price:      price,
		Location: strings.Join(
			nonempty(
				p.Location.City,
				p.Location.Area,
				p.Location.State,
			),
			" › ",
		),
		Amenities: strings.Join(
			detailParts, " · ",
		),
		Provider:     p.ProviderName,
		ProviderLogo: providerLogo(p.ProviderName),
		IsDemo: strings.HasPrefix(p.Slug, "demo-") &&
			strings.HasSuffix(p.Name, " (Demo)"),
		BookingURL: safeBooking(
			p.ProviderName, p.BookingURL,
		),
		Rating:  p.Rating,
		Reviews: p.ReviewCount,
		Eco:     p.EcoFriendly,
	}
}

// nearbySortOption is one choice in the "Sort by" select.
type nearbySortOption struct {
	Value    string
	Label    string
	Selected bool
}

// The same three choices, values and labels
var nearbySortChoices = [][2]string{
	{"popular", "Most Popular"},
	{"price_low", "Price: Low to High"},
	{"price_high", "Price: High to Low"},
}

// nearbySortOptions validates ?sort=. Unknown values fall back
// to "popular" instead of failing the whole page.
func nearbySortOptions(raw string) (string, []nearbySortOption) {
	selected := "popular"
	for _, choice := range nearbySortChoices {
		if raw == choice[0] {
			selected = raw
		}
	}

	options := make([]nearbySortOption, 0, len(nearbySortChoices))
	for _, choice := range nearbySortChoices {
		options = append(options, nearbySortOption{
			Value:    choice[0],
			Label:    choice[1],
			Selected: choice[0] == selected,
		})
	}

	return selected, options
}

const minimumNearby = 4

// selectNearby keeps every other property in the same city.
// Only when that gives fewer than four does it top up with the
// closest remaining same-country properties.
func selectNearby(
	all []models.Property, selected models.Property,
) []models.Property {
	ranked := make([]models.Property, 0, len(all))
	for _, p := range all {
		if p.ID != selected.ID {
			ranked = append(ranked, p)
		}
	}

	// Stable: SearchCountry's popularity order is kept within a rank.
	sort.SliceStable(ranked, func(i, j int) bool {
		return propertyRelatedRank(ranked[i], selected) >
			propertyRelatedRank(ranked[j], selected)
	})

	nearby := make([]models.Property, 0, len(ranked))
	for _, p := range ranked {
		if propertyRelatedRank(p, selected) >= 1 ||
			len(nearby) < minimumNearby {
			nearby = append(nearby, p)
		}
	}

	return nearby
}

// sortNearby applies the visitor's choice on the server.
// Same-city properties always stay ahead of top-up properties
// from other cities; the choice orders each group. Stable
// sorting keeps popularity order between equal values.
func sortNearby(
	list []models.Property, mode string, selected models.Property,
) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		aLocal := propertyRelatedRank(a, selected) >= 1
		bLocal := propertyRelatedRank(b, selected) >= 1
		if aLocal != bLocal {
			return aLocal
		}
		switch mode {
		case "price_high":
			return a.PriceFrom > b.PriceFrom
		case "price_low":
			return a.PriceFrom < b.PriceFrom
		default:
			return a.PopularityScore > b.PopularityScore
		}
	})
}

func propertyRelatedRank(
	p, selected models.Property,
) int {
	if p.Location.Country != selected.Location.Country {
		return -1
	}
	if p.Location.State == selected.Location.State &&
		p.Location.City == selected.Location.City &&
		p.Location.Area == selected.Location.Area {
		return 2
	}
	if p.Location.State == selected.Location.State &&
		p.Location.City == selected.Location.City {
		return 1
	}
	return 0
}

func (c *PropertyDetailsController) Get() {
	if dependencies.catalog == nil ||
		dependencies.loader == nil ||
		detailsDependencies.reader == nil ||
		detailsDependencies.search == nil {
		c.writeDetailsError(
			http.StatusInternalServerError,
			"Application is not initialized",
		)
		return
	}

	// Resolve the actual request Host, never a URL-supplied site key.
	location, ok := dependencies.catalog.Resolve(
		c.Ctx.Request.Host, "/",
	)
	if !ok {
		c.writeDetailsError(http.StatusNotFound, "Site not found")
		return
	}

	slug := c.Ctx.Input.Param(":slug")
	propertyID := c.Ctx.Input.Param(":propertyID")

	details, found, err := detailsDependencies.reader.FindDetails(
		location.ContentDirectory, slug, propertyID,
	)
	if err != nil {
		log.Printf("Property lookup failed: %v", err)
		c.writeDetailsError(
			http.StatusInternalServerError,
			"Failed to load property",
		)
		return
	}
	if !found {
		c.writeDetailsError(http.StatusNotFound, "Property not found")
		return
	}

	content, err := dependencies.loader.Load(location)
	if err != nil {
		log.Printf("Property publisher content failed: %v", err)
		c.writeDetailsError(
			http.StatusInternalServerError,
			"Failed to load site content",
		)
		return
	}

	// Related listings are still fetched from PostgreSQL.
	// SearchCountry already restricts results to the site key.
	nearby, err := detailsDependencies.search.SearchCountry(
		location.ContentDirectory,
		details.Property.Location.Country,
	)
	if err != nil {
		log.Printf("Related properties failed: %v", err)
		c.writeDetailsError(
			http.StatusInternalServerError,
			"Failed to load nearby properties",
		)
		return
	}

	sortMode, sortOptions := nearbySortOptions(c.GetString("sort"))
	selectedNearby := selectNearby(nearby, details.Property)
	sortNearby(selectedNearby, sortMode, details.Property)

	nearbyCards := make([]countryCard, 0, len(selectedNearby))
	for _, p := range selectedNearby {
		nearbyCards = append(nearbyCards, newListingCard(p))
	}

	nearbyLabel := details.Property.Location.City
	if area := details.Property.Location.Area; area != "" && area != nearbyLabel {
		nearbyLabel = area + ", " + nearbyLabel
	}

	language := content.Header.Language
	if language == "" {
		language = "en"
	}

	c.Data["Language"] = language
	c.Data["SiteName"] = content.Header.SiteName
	c.Data["CanonicalURL"] = "https://" + location.Domain +
		"/property/" + url.PathEscape(slug) +
		"/" + url.PathEscape(propertyID)

	c.Data["GlobalCSS"] = content.Header.CSSFile
	c.Data["TemplateCSS"] = content.Header.TemplateCSSFile
	c.Data["GlobalJS"] = content.Header.JSFile

	// Trusted publisher fragments, not database description text.
	c.Data["HeaderHTML"] = template.HTML(content.Header.HeaderHTML)
	c.Data["FooterHTML"] = template.HTML(content.Footer.FooterHTML)
	c.Data["WidgetHTML"] = template.HTML(content.Header.WidgetCode)

	c.Data["Property"] = details.Property
	c.Data["SiteKey"] = location.ContentDirectory
	c.Data["Crumbs"] = propertyCrumbs(details.Property.Location)
	c.Data["IsDemo"] = strings.HasPrefix(details.Property.Slug, "demo-") &&
		strings.HasSuffix(details.Property.Name, " (Demo)")
	c.Data["Photos"] = sizedPhotos(
		safeGalleryPhotos(details.Property.Images), galleryPhotoWidth,
	)
	c.Data["HasDetails"] = details.Details.Valid
	c.Data["Details"] = details.Details.String
	c.Data["HasBedrooms"] = details.Bedrooms.Valid
	c.Data["Bedrooms"] = details.Bedrooms.Int64
	c.Data["HasBathrooms"] = details.Bathrooms.Valid
	c.Data["Bathrooms"] = details.Bathrooms.Float64
	c.Data["PetFriendly"] =
		details.PetFriendly.Valid && details.PetFriendly.Bool

	c.Data["BookingURL"] = safeBooking(
		details.Property.ProviderName,
		details.Property.BookingURL,
	)
	c.Data["ProviderLogo"] = providerLogo(details.Property.ProviderName)
	c.Data["PriceText"] = formatNightlyPrice(
		details.Property.Currency,
		details.Property.PriceFrom,
	)

	// "Not the right fit?" links to the same area search
	// as the first breadcrumb.
	fitLabel := details.Property.Location.Area
	if fitLabel == "" {
		fitLabel = details.Property.Location.City
	}
	c.Data["FitLabel"] = fitLabel
	c.Data["FitURL"] = refineSearchURL(fitLabel)
	c.Data["NearbyCards"] = nearbyCards
	c.Data["NearbyLabel"] = nearbyLabel
	c.Data["SortOptions"] = sortOptions
	c.Data["SeeMoreURL"] = refineSearchURL(details.Property.Location.City)

	c.TplName = "property.tpl"

	if err := c.Render(); err != nil {
		log.Printf("Property rendering failed: %v", err)
		c.writeDetailsError(
			http.StatusInternalServerError,
			"Failed to render property",
		)
		return
	}

	c.EnableRender = false
}

func (c *PropertyDetailsController) writeDetailsError(
	status int, message string,
) {
	c.EnableRender = false
	c.Ctx.Output.SetStatus(status)
	_ = c.Ctx.Output.Body([]byte(message))
}
