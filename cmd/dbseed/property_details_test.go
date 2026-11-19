package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestPropertyDetailsForAll48Seeds(t *testing.T) {
	originals, err := gallerySeedProperties()
	if err != nil {
		t.Fatal(err)
	}

	demos, err := additionalDemoSeedProperties(originals)
	if err != nil {
		t.Fatal(err)
	}

	all := append(
		append([]seedProperty(nil), originals...),
		demos...,
	)

	if len(all) != 48 {
		t.Fatalf("want 48, got %d", len(all))
	}

	for i, p := range all {
		spec, err := propertyDetailsForSeed(p)
		if err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}

		if !spec.Details.Valid ||
			!strings.Contains(spec.Details.String, p.Name) ||
			!spec.Bedrooms.Valid ||
			spec.Bedrooms.Int64 < 1 ||
			!spec.Bathrooms.Valid ||
			spec.Bathrooms.Float64 < 1 {
			t.Errorf(
				"%s has incomplete details seed", p.ID,
			)
		}

		// Original named hotels: unknown pet policy.
		// Explicit demos: illustrative pet policy.
		if (i >= 15) != spec.PetFriendly.Valid {
			t.Errorf(
				"%s has unexpected pet-policy validity",
				p.ID,
			)
		}
	}
}

func TestClassifyPropertyDetails(t *testing.T) {
	expected := propertyDetailsSeed{
		Details: sql.NullString{
			String: "sample",
			Valid:  true,
		},
		Bedrooms: sql.NullInt64{
			Int64: 2,
			Valid: true,
		},
		Bathrooms: sql.NullFloat64{
			Float64: 1,
			Valid:   true,
		},
	}

	if got := classifyPropertyDetails(
		propertyDetailsSeed{}, expected,
	); got != "pending" {
		t.Fatalf("null row: got %s", got)
	}

	if got := classifyPropertyDetails(
		expected, expected,
	); got != "unchanged" {
		t.Fatalf("matching row: got %s", got)
	}

	partial := propertyDetailsSeed{
		Details: sql.NullString{
			String: "custom",
			Valid:  true,
		},
	}

	if got := classifyPropertyDetails(
		partial, expected,
	); got != "conflict" {
		t.Fatalf("customized row: got %s", got)
	}
}
