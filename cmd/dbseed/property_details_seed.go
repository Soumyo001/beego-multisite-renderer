package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// propertyDetailsSeed is assessment data, not verified hotel metadata.
type propertyDetailsSeed struct {
	Details     sql.NullString
	Bedrooms    sql.NullInt64
	Bathrooms   sql.NullFloat64
	PetFriendly sql.NullBool
}

func propertyDetailsForSeed(
	p seedProperty,
) (propertyDetailsSeed, error) {
	if p.ID == "" || p.Name == "" || p.Type == "" ||
		p.Location.City == "" || p.Location.State == "" ||
		p.Location.Area == "" || p.Location.Country == "" ||
		p.MaxGuests < 1 || len(p.Amenities) == 0 {
		return propertyDetailsSeed{}, fmt.Errorf(
			"incomplete property details source for %q", p.ID,
		)
	}

	bedrooms := int64((p.MaxGuests + 1) / 2)
	bathrooms := 1.0

	if bedrooms >= 3 {
		bathrooms = 2.0
	}
	if bedrooms >= 5 {
		bathrooms = 3.0
	}

	description := fmt.Sprintf(
		"%s is an illustrative %s listing in %s, %s, %s. "+
			"For this assessment, the sample accommodation is shown for up to %d guests, "+
			"with amenities including %s. The gallery, room configuration, nightly price, "+
			"and property facts are demonstration data. They are not verified descriptions "+
			"of the named accommodation, and do not establish live booking availability.",
		p.Name,
		strings.ToLower(p.Type),
		p.Location.Area,
		p.Location.City,
		p.Location.State,
		p.MaxGuests,
		strings.Join(p.Amenities, ", "),
	)

	spec := propertyDetailsSeed{
		Details: sql.NullString{
			String: description,
			Valid:  true,
		},
		Bedrooms: sql.NullInt64{
			Int64: bedrooms,
			Valid: true,
		},
		Bathrooms: sql.NullFloat64{
			Float64: bathrooms,
			Valid:   true,
		},
	}

	// Only explicitly named demo listings receive a
	// fictional pet policy. Original named hotels remain unknown.
	if strings.HasSuffix(p.Name, " (Demo)") &&
		strings.HasPrefix(p.Slug, "demo-") {

		number, err := strconv.Atoi(
			strings.TrimPrefix(p.ID, "prop-"),
		)
		if err != nil {
			return propertyDetailsSeed{}, fmt.Errorf(
				"demo property %q has invalid ID: %w",
				p.ID, err,
			)
		}

		spec.PetFriendly = sql.NullBool{
			Bool:  number%3 == 0,
			Valid: true,
		}
	}

	return spec, nil
}

func classifyPropertyDetails(
	actual, expected propertyDetailsSeed,
) string {
	if actual == expected {
		return "unchanged"
	}
	if actual == (propertyDetailsSeed{}) {
		return "pending"
	}
	return "conflict"
}
