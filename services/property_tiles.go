package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sparrow-ssr/models"

	"github.com/beego/beego/v2/client/orm"
)

// Sort orders used by the publisher's tiles_component "order" field.
// Confirmed from the dummy tiles in home.json: the "order": "2" block
// lists prices rising, the "order": "3" block lists prices falling.
const (
	TilesOrderPopular   = 1 // most popular first
	TilesOrderPriceLow  = 2 // price low to high
	TilesOrderPriceHigh = 3 // price high to low
)

// maxTiles bounds one block's query, whatever total_tiles says.
const maxTiles = 48

// TilesQuery is one tiles_component block's database request.
type TilesQuery struct {
	City    string
	Country string
	Order   int
	Limit   int
}

// PropertyTilesSearcher is the capability needed to fill
// home-page tiles blocks. It is separate from PropertySearcher so
// existing test fixtures do not have to implement it.
type PropertyTilesSearcher interface {
	SearchTiles(siteDirectory string, query TilesQuery) ([]models.Property, error)
}

// ParseTilesKeyword reads a publisher keyword such as
// "Toronto, ON, Canada" or "Dhaka, Bangladesh". The first part is
// the city and the last part the country; middle parts (state or
// province codes) are ignored. A single part, such as "Canada", is
// returned as city with an empty country and matches either field.
func ParseTilesKeyword(keyword string) (city, country string, ok bool) {
	parts := make([]string, 0, 3)
	for _, part := range strings.Split(keyword, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}

	switch len(parts) {
	case 0:
		return "", "", false
	case 1:
		return parts[0], "", true
	default:
		return parts[0], parts[len(parts)-1], true
	}
}

// ParseTilesOrder converts the "order" string. Unknown or missing
// values fall back to most popular instead of failing the page.
func ParseTilesOrder(raw string) int {
	order, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || order < TilesOrderPopular || order > TilesOrderPriceHigh {
		return TilesOrderPopular
	}
	return order
}

// tilesOrderBy returns the ORM ordering for a tiles order.
// PropertyID is the tie-breaker so equal prices stay stable.
func tilesOrderBy(order int) []string {
	switch order {
	case TilesOrderPriceLow:
		return []string{"PriceFrom", "PropertyID"}
	case TilesOrderPriceHigh:
		return []string{"-PriceFrom", "PropertyID"}
	default:
		return []string{"-PopularityScore", "PropertyID"}
	}
}

// SearchTiles returns at most query.Limit properties of one site in
// the given city and country, in the requested order.
func (s *DatabasePropertySearchService) SearchTiles(
	siteDirectory string, query TilesQuery,
) ([]models.Property, error) {
	if strings.TrimSpace(siteDirectory) == "" {
		return nil, fmt.Errorf("site directory must not be empty")
	}

	city := strings.TrimSpace(query.City)
	country := strings.TrimSpace(query.Country)
	if city == "" {
		return nil, fmt.Errorf("tiles city must not be empty")
	}

	limit := query.Limit
	if limit < 1 {
		return []models.Property{}, nil
	}
	if limit > maxTiles {
		limit = maxTiles
	}

	condition := orm.NewCondition().And("SiteKey", siteDirectory)
	if country != "" {
		condition = condition.
			And("Location__City__iexact", city).
			And("Location__Country__iexact", country)
	} else {
		condition = condition.AndCond(orm.NewCondition().
			Or("Location__City__iexact", city).
			Or("Location__Country__iexact", city))
	}

	var rows []models.PropertyRecord
	_, err := orm.NewOrm().QueryTable(new(models.PropertyRecord)).
		SetCond(condition).
		RelatedSel("Location").
		OrderBy(tilesOrderBy(query.Order)...).
		Limit(limit).
		All(&rows)
	if err != nil {
		return nil, fmt.Errorf("query tiles properties: %w", err)
	}

	properties := make([]models.Property, 0, len(rows))
	for _, row := range rows {
		property, err := propertyFromRecord(row)
		if err != nil {
			return nil, err
		}
		properties = append(properties, property)
	}

	return properties, nil
}

// propertyFromRecord converts one database row into the public
// Property shape used by the API, cards and templates.
func propertyFromRecord(row models.PropertyRecord) (models.Property, error) {
	if row.Location == nil {
		return models.Property{}, fmt.Errorf(
			"property %s has no location", row.PropertyID,
		)
	}

	var amenities []string
	if err := json.Unmarshal([]byte(row.AmenitiesJSON), &amenities); err != nil {
		return models.Property{}, fmt.Errorf(
			"decode amenities for %s: %w", row.PropertyID, err,
		)
	}

	var images []string
	if err := json.Unmarshal([]byte(row.ImageJSON), &images); err != nil {
		return models.Property{}, fmt.Errorf(
			"decode images for %s: %w", row.PropertyID, err,
		)
	}

	return models.Property{
		ID:           row.PropertyID,
		Slug:         row.Slug,
		Name:         row.Name,
		PropertyType: row.PropertyType,
		Location: models.PropertyLocation{
			City:    row.Location.City,
			State:   row.Location.State,
			Area:    row.Location.Area,
			Country: row.Location.Country,
		},
		MaxGuests:       row.MaxGuests,
		SizeSqFt:        row.SizeSqFt,
		Rating:          row.Rating,
		ReviewCount:     row.ReviewCount,
		PriceFrom:       row.PriceFrom,
		Currency:        row.Currency,
		Amenities:       amenities,
		ProviderName:    row.ProviderName,
		BookingURL:      row.BookingURL,
		PopularityScore: row.PopularityScore,
		EcoFriendly:     row.EcoFriendly,
		Images:          images,
	}, nil
}

var _ PropertyTilesSearcher = (*DatabasePropertySearchService)(nil)
