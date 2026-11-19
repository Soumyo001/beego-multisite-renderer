package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"sparrow-ssr/models"

	"github.com/beego/beego/v2/client/orm"
)

// PropertyDetails contains the existing public property fields
// plus the additional nullable details stored in PostgreSQL.
//
// It is a server-side view model, not the /api/properties
// JSON response.
type PropertyDetails struct {
	Property    models.Property
	Details     sql.NullString
	Bedrooms    sql.NullInt64
	Bathrooms   sql.NullFloat64
	PetFriendly sql.NullBool
}

// PropertyDetailsReader defines the capability required
// by the property-details SSR controller.
//
// The existing property-listing consumers do not need
// to implement this interface.
type PropertyDetailsReader interface {
	FindDetails(siteDirectory, slug, propertyID string) (PropertyDetails, bool, error)
}

var propertyDetailsSlugPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*$`,
)

var propertyDetailsIDPattern = regexp.MustCompile(
	`^[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*$`,
)

func validPropertyDetailsIdentity(slug, propertyID string) bool {
	return len(slug) > 0 &&
		len(slug) <= 255 &&
		len(propertyID) > 0 &&
		len(propertyID) <= 100 &&
		propertyDetailsSlugPattern.MatchString(slug) &&
		propertyDetailsIDPattern.MatchString(propertyID)
}

// FindDetails retrieves one property strictly inside
// the publisher's database scope.
//
// found=false represents a missing property or an invalid
// slug/ID combination. Unexpected database problems are
// returned as errors.
func (s *DatabasePropertySearchService) FindDetails(
	siteDirectory, slug, propertyID string,
) (PropertyDetails, bool, error) {
	if strings.TrimSpace(siteDirectory) == "" {
		return PropertyDetails{}, false,
			fmt.Errorf("site directory must not be empty")
	}

	if !validPropertyDetailsIdentity(slug, propertyID) {
		return PropertyDetails{}, false, nil
	}

	var record models.PropertyRecord

	err := orm.NewOrm().
		QueryTable(new(models.PropertyRecord)).
		Filter("SiteKey", siteDirectory).
		Filter("Slug", slug).
		Filter("PropertyID", propertyID).
		RelatedSel("Location").
		One(&record)

	if errors.Is(err, orm.ErrNoRows) {
		return PropertyDetails{}, false, nil
	}

	if err != nil {
		return PropertyDetails{}, false, fmt.Errorf(
			"lookup property details: %w", err,
		)
	}

	details, err := propertyDetailsFromRecord(record)
	if err != nil {
		return PropertyDetails{}, false, err
	}

	return details, true, nil
}

// Convert the database record into a server-side model.
// Keep the public Property JSON shape unchanged.
func propertyDetailsFromRecord(
	row models.PropertyRecord,
) (PropertyDetails, error) {
	if row.Location == nil {
		return PropertyDetails{}, fmt.Errorf(
			"property %s has no location",
			row.PropertyID,
		)
	}

	var amenities []string

	if err := json.Unmarshal([]byte(row.AmenitiesJSON), &amenities); err != nil {
		return PropertyDetails{}, fmt.Errorf(
			"decode amenities for %s: %w",
			row.PropertyID, err,
		)
	}

	var images []string

	if err := json.Unmarshal([]byte(row.ImageJSON), &images); err != nil {
		return PropertyDetails{}, fmt.Errorf(
			"decode images for %s: %w",
			row.PropertyID, err,
		)
	}

	property := models.Property{
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
	}

	return PropertyDetails{
		Property:    property,
		Details:     row.Details,
		Bedrooms:    row.Bedrooms,
		Bathrooms:   row.Bathrooms,
		PetFriendly: row.PetFriendly,
	}, nil
}

var _ PropertyDetailsReader = (*DatabasePropertySearchService)(nil)
