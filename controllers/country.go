package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sparrow-ssr/services"

	beego "github.com/beego/beego/v2/server/web"
)

var countrySearcher services.PropertySearcher

func SetCountrySearchDependencies(
	search services.PropertySearcher,
) {
	countrySearcher = search
}

type CountryController struct {
	beego.Controller
}

type countryCard struct {
	Name       string
	Type       string
	Photo      string
	Price      string
	Location   string
	Amenities  string
	Provider   string
	BookingURL string
	Rating     float64
	Reviews    int
	Eco        bool
}

const countryPageSize = 8

type countryPageLink struct {
	Number  int
	URL     string
	Current bool
}

// Calculate a server-rendered page of country listings.
func countryPageBounds(total int, raw string) (
	page, pageCount, start, end int, ok bool,
) {
	pageCount = (total + countryPageSize - 1) / countryPageSize
	if pageCount == 0 {
		pageCount = 1
	}

	page = 1

	if raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		page = parsed
	}

	// Clamp numeric page requests to the available range.
	if page < 1 {
		page = 1
	}
	if page > pageCount {
		page = pageCount
	}

	start = (page - 1) * countryPageSize
	end = start + countryPageSize

	if end > total {
		end = total
	}

	return page, pageCount, start, end, true
}

var countrySlugPattern = regexp.MustCompile(
	`^[a-z]+(?:-[a-z]+)*$`,
)

func countryFromSlug(slug string) (string, bool) {
	if len(slug) < 2 ||
		len(slug) > 80 ||
		!countrySlugPattern.MatchString(slug) {
		return "", false
	}

	if slug == "usa" {
		return "USA", true
	}

	words := strings.Split(slug, "-")
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}

	return strings.Join(words, " "), true
}

func safePhoto(images []string) string {
	if len(images) == 0 {
		return ""
	}

	u, err := url.Parse(images[0])
	if err != nil ||
		(u.Scheme != "https" && u.Scheme != "http") ||
		u.Host == "" ||
		u.User != nil {
		return ""
	}

	return u.String()
}

// Match the provider restrictions from static/js/refine.js.
func safeBooking(provider, raw string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))

	domain := ""

	switch provider {
	case "booking.com":
		domain = "booking.com"
	case "expedia", "expedia.com":
		domain = "expedia.com"
	default:
		return ""
	}

	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil ||
		u.Scheme != "https" ||
		u.User != nil ||
		u.Port() != "" ||
		u.Path == "" ||
		u.Path == "/" {
		return ""
	}

	host := strings.ToLower(u.Hostname())

	if host != domain &&
		!strings.HasSuffix(host, "."+domain) {
		return ""
	}

	return u.String()
}

func (c *CountryController) Get() {
	if dependencies.catalog == nil ||
		dependencies.loader == nil ||
		countrySearcher == nil {
		c.writeCountryError(
			http.StatusInternalServerError,
			"Application is not initialized",
		)
		return
	}

	location, found := dependencies.catalog.Resolve(
		c.Ctx.Request.Host,
		"/",
	)
	if !found {
		c.writeCountryError(
			http.StatusNotFound,
			"Site not found",
		)
		return
	}

	slug := c.Ctx.Input.Param(":country")
	country, valid := countryFromSlug(slug)
	if !valid {
		c.writeCountryError(
			http.StatusNotFound,
			"Country not found",
		)
		return
	}

	properties, err := countrySearcher.SearchCountry(
		location.ContentDirectory, country,
	)
	if err != nil {
		log.Printf("Country property search failed: %v", err)
		c.writeCountryError(
			http.StatusInternalServerError,
			"Failed to search country properties",
		)
		return
	}

	// Build filter metadata from all country properties,
	// before slicing the server-rendered page of eight.
	filterLow, filterHigh := 0, 0
	priceSeen := false
	filterCurrency := "USD"

	currencies := make(map[string]struct{})
	amenitySet := make(map[string]struct{})

	for _, property := range properties {
		if property.PriceFrom > 0 {
			if !priceSeen || property.PriceFrom < filterLow {
				filterLow = property.PriceFrom
			}

			if property.PriceFrom > filterHigh {
				filterHigh = property.PriceFrom
			}

			priceSeen = true

			if property.Currency != "" {
				currencies[property.Currency] = struct{}{}
			}
		}

		for _, amenity := range property.Amenities {
			if name := strings.TrimSpace(amenity); name != "" {
				amenitySet[name] = struct{}{}
			}
		}
	}

	if !priceSeen {
		filterLow, filterHigh = 0, 100
	} else if filterHigh <= filterLow {
		filterHigh = filterLow + 1
	}

	if len(currencies) == 1 {
		for currency := range currencies {
			filterCurrency = currency
		}
	}

	filterAmenities := make([]string, 0, len(amenitySet))
	for amenity := range amenitySet {
		filterAmenities = append(filterAmenities, amenity)
	}

	sort.Strings(filterAmenities)

	total := len(properties)

	page, pageCount, start, end, ok := countryPageBounds(
		total,
		c.Ctx.Request.URL.Query().Get("page"),
	)

	if !ok {
		c.writeCountryError(
			http.StatusNotFound,
			"Page not found",
		)
		return
	}

	content, err := dependencies.loader.Load(location)
	if err != nil {
		log.Printf(
			"Country page content loading failed: %v", err,
		)
		c.writeCountryError(
			http.StatusInternalServerError,
			"Failed to load site content",
		)
		return
	}

	cards := make([]countryCard, 0, end-start)

	for _, p := range properties[start:end] {
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

		cards = append(cards, countryCard{
			Name:  p.Name,
			Type:  p.PropertyType,
			Photo: safePhoto(p.Images),
			Price: price,
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
			Provider: p.ProviderName,
			BookingURL: safeBooking(
				p.ProviderName, p.BookingURL,
			),
			Rating:  p.Rating,
			Reviews: p.ReviewCount,
			Eco:     p.EcoFriendly,
		})
	}

	language := content.Header.Language
	if language == "" {
		language = "en"
	}

	c.Data["Language"] = language
	c.Data["SiteName"] = content.Header.SiteName
	c.Data["Country"] = country
	c.Data["IsCountry"] = true

	c.Data["FilterPriceLow"] = filterLow
	c.Data["FilterPriceHigh"] = filterHigh
	c.Data["FilterCurrency"] = filterCurrency
	c.Data["FilterAmenities"] = filterAmenities

	c.Data["Total"] = total
	c.Data["Cards"] = cards
	c.Data["Page"] = page
	c.Data["PageCount"] = pageCount

	if total > 0 {
		c.Data["PageStart"] = start + 1
		c.Data["PageEnd"] = end
	}

	pageLinks := make([]countryPageLink, 0, pageCount)

	for n := 1; n <= pageCount; n++ {
		path := "/all/" + slug

		if n > 1 {
			path += "?page=" + strconv.Itoa(n)
		}

		pageLinks = append(pageLinks, countryPageLink{
			Number:  n,
			URL:     path,
			Current: n == page,
		})
	}

	c.Data["PageLinks"] = pageLinks

	if page > 1 {
		c.Data["PrevURL"] = pageLinks[page-2].URL
	}

	if page < pageCount {
		c.Data["NextURL"] = pageLinks[page].URL
	}

	c.Data["CanonicalURL"] =
		"https://" + location.Domain + "/all/" + slug

	c.Data["RefineURL"] =
		"/refine?search=" +
			url.QueryEscape(country) + "&pax=2&order=1"

	c.Data["GlobalCSS"] = content.Header.CSSFile
	c.Data["TemplateCSS"] =
		content.Header.TemplateCSSFile
	c.Data["GlobalJS"] = content.Header.JSFile

	// Trusted publisher-provided fragments only.
	c.Data["HeaderHTML"] = template.HTML(
		content.Header.HeaderHTML,
	)
	c.Data["FooterHTML"] = template.HTML(
		content.Footer.FooterHTML,
	)
	c.Data["WidgetHTML"] = template.HTML(
		content.Header.WidgetCode,
	)

	c.TplName = "refine.tpl"

	if err := c.Render(); err != nil {
		log.Printf("Country page rendering failed: %v", err)
		c.writeCountryError(
			http.StatusInternalServerError,
			"Failed to render country page",
		)
		return
	}

	c.EnableRender = false
}

func nonempty(values ...string) []string {
	result := make([]string, 0, len(values))

	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}

	return result
}

func (c *CountryController) writeCountryError(
	status int,
	message string,
) {
	c.EnableRender = false
	c.Ctx.Output.SetStatus(status)
	_ = c.Ctx.Output.Body([]byte(message))
}
