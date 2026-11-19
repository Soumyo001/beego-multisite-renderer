package services

import (
	"database/sql"
	"strings"
	"testing"

	"sparrow-ssr/models"
)

func TestPropertyDetailsIdentity(t *testing.T) {
	tests := []struct {
		slug string
		id   string
		want bool
	}{
		{
			"the-plaza-a-fairmont-hotel",
			"prop-001",
			true,
		},
		{
			"demo-midtown-garden-suites",
			"prop-016",
			true,
		},
		{
			"blue-heron-house",
			"BC-3156886",
			true,
		},
		{
			"Wrong-Slug",
			"prop-001",
			false,
		},
		{
			"the-plaza",
			"../prop-001",
			false,
		},
		{
			"the-plaza",
			"",
			false,
		},
		{
			"",
			"prop-001",
			false,
		},
	}

	for _, tc := range tests {
		got := validPropertyDetailsIdentity(
			tc.slug, tc.id,
		)

		if got != tc.want {
			t.Errorf(
				"identity(%q, %q) = %v; want %v",
				tc.slug, tc.id, got, tc.want,
			)
		}
	}
}

func testPropertyDetailsRecord() models.PropertyRecord {
	return models.PropertyRecord{
		SiteKey:    "beta/preview/test-travel-templates",
		PropertyID: "prop-001",
		Slug:       "the-plaza-a-fairmont-hotel",
		Name:       "The Plaza, A Fairmont Hotel",

		PropertyType: "Hotel",
		MaxGuests:    2,
		PriceFrom:    890,
		Currency:     "USD",

		Location: &models.LocationRecord{
			City:    "New York",
			State:   "New York",
			Area:    "Midtown Manhattan",
			Country: "USA",
		},

		AmenitiesJSON: `["Free Wifi","Fitness center"]`,
		ImageJSON: `[
			"https://images.example.com/photo-1.jpg",
			"https://images.example.com/photo-2.jpg"
		]`,

		Details: sql.NullString{
			String: "Illustrative property description",
			Valid:  true,
		},
		Bedrooms: sql.NullInt64{
			Int64: 1,
			Valid: true,
		},
		Bathrooms: sql.NullFloat64{
			Float64: 1.0,
			Valid:   true,
		},

		// Unknown policy for an original named hotel.
		PetFriendly: sql.NullBool{
			Valid: false,
		},
	}
}

func TestPropertyDetailsFromRecord(t *testing.T) {
	record := testPropertyDetailsRecord()

	got, err := propertyDetailsFromRecord(record)
	if err != nil {
		t.Fatal(err)
	}

	if got.Property.ID != "prop-001" ||
		got.Property.Name != record.Name ||
		got.Property.Location.Country != "USA" {
		t.Fatalf(
			"incorrect property identity: %+v",
			got.Property,
		)
	}

	if len(got.Property.Images) != 2 ||
		len(got.Property.Amenities) != 2 {
		t.Fatal("gallery or amenities missing")
	}

	if !got.Details.Valid ||
		!got.Bedrooms.Valid ||
		got.Bedrooms.Int64 != 1 ||
		!got.Bathrooms.Valid ||
		got.Bathrooms.Float64 != 1.0 {
		t.Fatal("property details were not preserved")
	}

	if got.PetFriendly.Valid {
		t.Fatal(
			"unknown pet policy became a known value",
		)
	}
}

func TestPropertyDetailsFromRecordErrors(t *testing.T) {
	t.Run("missing location", func(t *testing.T) {
		record := testPropertyDetailsRecord()
		record.Location = nil

		_, err := propertyDetailsFromRecord(record)

		if err == nil ||
			!strings.Contains(err.Error(), "location") {
			t.Fatal("expected missing location error")
		}
	})

	t.Run("invalid amenities", func(t *testing.T) {
		record := testPropertyDetailsRecord()
		record.AmenitiesJSON = "{invalid}"

		_, err := propertyDetailsFromRecord(record)

		if err == nil ||
			!strings.Contains(err.Error(), "amenities") {
			t.Fatal("expected amenities decoding error")
		}
	})

	t.Run("invalid images", func(t *testing.T) {
		record := testPropertyDetailsRecord()
		record.ImageJSON = "{invalid}"

		_, err := propertyDetailsFromRecord(record)

		if err == nil ||
			!strings.Contains(err.Error(), "images") {
			t.Fatal("expected gallery decoding error")
		}
	})
}
