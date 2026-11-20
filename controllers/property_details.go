package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"sparrow-ssr/models"
	"sparrow-ssr/services"

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

type propertyRelatedCard struct {
	Name     string
	URL      string
	Image    string
	Price    string
	Location string
	Rating   float64
	Reviews  int
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

	sort.SliceStable(nearby, func(i, j int) bool {
		return propertyRelatedRank(
			nearby[i], details.Property,
		) > propertyRelatedRank(
			nearby[j], details.Property,
		)
	})

	related := make([]propertyRelatedCard, 0, 4)

	for _, p := range nearby {
		if p.ID == details.Property.ID {
			continue
		}

		if len(related) == 4 {
			break
		}

		related = append(related, propertyRelatedCard{
			Name: p.Name,
			URL: "/property/" + url.PathEscape(p.Slug) +
				"/" + url.PathEscape(p.ID),
			Image:    safePhoto(p.Images),
			Price:    fmt.Sprintf("%s %d", p.Currency, p.PriceFrom),
			Location: p.Location.City + ", " + p.Location.State,
			Rating:   p.Rating,
			Reviews:  p.ReviewCount,
		})
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
	c.Data["IsDemo"] = strings.HasPrefix(details.Property.Slug, "demo-") &&
		strings.HasSuffix(details.Property.Name, " (Demo)")
	c.Data["Photos"] = safeGalleryPhotos(details.Property.Images)
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
	c.Data["Related"] = related

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
