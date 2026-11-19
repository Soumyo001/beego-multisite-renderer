package main

import (
	"testing"

	"sparrow-ssr/models"
)

func TestSeedData(t *testing.T) {
	if err := validateSeedData(seedProperties); err != nil {
		t.Fatalf("invalid seed data: %v", err)
	}
	if got := len(seedProperties); got != 15 {
		t.Fatalf("want 15 properties, got %d", got)
	}
	locations := make(map[models.PropertyLocation]bool)
	ids := make(map[string]bool)
	for _, property := range seedProperties {
		locations[property.Location] = true
		ids[property.ID] = true
	}
	if got := len(locations); got != 8 {
		t.Fatalf("want 8 distinct locations, got %d", got)
	}
	if !ids["prop-001"] || !ids["prop-015"] {
		t.Fatal("expected first and last property IDs")
	}
}

func TestRejectDuplicateSeedID(t *testing.T) {
	duplicate := append([]seedProperty(nil), seedProperties...)
	duplicate = append(duplicate, seedProperties[0])
	if err := validateSeedData(duplicate); err == nil {
		t.Fatal("expected duplicate ID to be rejected")
	}
}

func TestRejectDuplicateSeedSlug(t *testing.T) {
	duplicate := append([]seedProperty(nil), seedProperties...)
	duplicate[1].Slug = duplicate[0].Slug

	if err := validateSeedData(duplicate); err == nil {
		t.Fatal("expected duplicate slug to be rejected")
	}
}

func TestRejectInvalidSeedSlug(t *testing.T) {
	invalid := append([]seedProperty(nil), seedProperties...)
	invalid[0].Slug = "Not Valid"

	if err := validateSeedData(invalid); err == nil {
		t.Fatal("expected invalid slug to be rejected")
	}
}

func TestAdditionalDemoSeedProperties(t *testing.T) {
	// First build the original five-image gallery dataset.
	original, err := gallerySeedProperties()
	if err != nil {
		t.Fatalf("prepare original gallery: %v", err)
	}

	if len(original) != 15 {
		t.Fatalf(
			"expected 15 original properties, got %d",
			len(original),
		)
	}

	demos, err := additionalDemoSeedProperties(original)
	if err != nil {
		t.Fatalf("prepare demo properties: %v", err)
	}

	if len(demos) != 33 {
		t.Fatalf(
			"expected 33 demo properties, got %d",
			len(demos),
		)
	}

	// Validate original and demo properties together.
	combined := append(
		append([]seedProperty(nil), original...),
		demos...,
	)

	if len(combined) != 48 {
		t.Fatalf(
			"expected 48 total properties, got %d",
			len(combined),
		)
	}

	if err := validateSeedData(combined); err != nil {
		t.Fatalf(
			"expanded seed dataset is invalid: %v",
			err,
		)
	}

	if demos[0].ID != "prop-016" ||
		demos[len(demos)-1].ID != "prop-048" {
		t.Fatal("unexpected demo property ID range")
	}

	// All demonstration properties must belong to USA,
	// carry five images, and remain clearly identified.
	locations := make(map[models.PropertyLocation]bool)

	for _, property := range combined {
		locations[property.Location] = true
	}

	if len(locations) != 8 {
		t.Fatalf(
			"expected 8 original locations, got %d",
			len(locations),
		)
	}

	for _, property := range demos {
		if property.Location.Country != "USA" {
			t.Errorf(
				"%s has unexpected country %q",
				property.ID,
				property.Location.Country,
			)
		}

		if len(property.Images) != 5 {
			t.Errorf(
				"%s has %d images, expected 5",
				property.ID,
				len(property.Images),
			)
		}

		if property.ProviderName != "Demo Listing" {
			t.Errorf(
				"%s has unexpected provider %q",
				property.ID,
				property.ProviderName,
			)
		}

		if property.BookingURL !=
			"https://example.com/demo/"+property.ID {
			t.Errorf(
				"%s has unexpected demo booking URL",
				property.ID,
			)
		}
	}

	// Confirm the new properties do not share image slice
	// storage with the original gallery properties.
	firstOriginalImage := original[0].Images[0]

	demos[0].Images[0] = "temporary-test-value"

	if original[0].Images[0] != firstOriginalImage {
		t.Fatal(
			"demo property changed original image data",
		)
	}
}
