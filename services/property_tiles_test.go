package services

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/joho/godotenv"
)

func TestParseTilesKeyword(t *testing.T) {
	for _, tc := range []struct {
		keyword, city, country string
		ok                     bool
	}{
		{"Toronto, ON, Canada", "Toronto", "Canada", true},
		{"Dhaka, Bangladesh", "Dhaka", "Bangladesh", true},
		{"  Canada ", "Canada", "", true},
		{"Toronto,, Canada,", "Toronto", "Canada", true},
		{" , ", "", "", false},
		{"", "", "", false},
	} {
		city, country, ok := ParseTilesKeyword(tc.keyword)
		if city != tc.city || country != tc.country || ok != tc.ok {
			t.Errorf("ParseTilesKeyword(%q) = %q, %q, %v; want %q, %q, %v",
				tc.keyword, city, country, ok, tc.city, tc.country, tc.ok)
		}
	}
}

func TestParseTilesOrder(t *testing.T) {
	for raw, want := range map[string]int{
		"1": TilesOrderPopular, "2": TilesOrderPriceLow, " 3 ": TilesOrderPriceHigh,
		"": TilesOrderPopular, "0": TilesOrderPopular, "4": TilesOrderPopular, "x": TilesOrderPopular,
	} {
		if got := ParseTilesOrder(raw); got != want {
			t.Errorf("ParseTilesOrder(%q) = %d; want %d", raw, got, want)
		}
	}
}

func TestTilesOrderBy(t *testing.T) {
	for order, want := range map[int][]string{
		TilesOrderPopular:   {"-PopularityScore", "PropertyID"},
		TilesOrderPriceLow:  {"PriceFrom", "PropertyID"},
		TilesOrderPriceHigh: {"-PriceFrom", "PropertyID"},
	} {
		if got := tilesOrderBy(order); !reflect.DeepEqual(got, want) {
			t.Errorf("tilesOrderBy(%d) = %v; want %v", order, got, want)
		}
	}
}

// Run explicitly with SPARROW_TEST_NEON=1. Read-only.
func TestSearchTilesSeededNeon(t *testing.T) {
	if os.Getenv("SPARROW_TEST_NEON") != "1" {
		t.Skip("set SPARROW_TEST_NEON=1 to run the read-only Neon test")
	}
	if err := godotenv.Load("../.env"); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load environment: %v", err)
	}
	databaseURL := os.Getenv("DATABASE_URL_DIRECT")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if err := InitPropertyDatabase(databaseURL); err != nil {
		t.Fatalf("initialize database: %v", err)
	}

	const site = "beta/preview/test-travel-templates"
	search := NewDatabasePropertySearchService()

	for _, tc := range []struct {
		name        string
		query       TilesQuery
		want        int
		first, last int // expected first/last price; 0 = not checked
		wantCountry string
	}{
		{"toronto_high", TilesQuery{"Toronto", "Canada", TilesOrderPriceHigh, 12}, 12, 620, 180, "Canada"},
		{"dhaka_low", TilesQuery{"Dhaka", "Bangladesh", TilesOrderPriceLow, 12}, 12, 40, 185, "Bangladesh"},
		{"limit_4", TilesQuery{"Dhaka", "Bangladesh", TilesOrderPopular, 4}, 4, 0, 0, "Bangladesh"},
		{"case_insensitive", TilesQuery{"toronto", "CANADA", TilesOrderPriceLow, 12}, 12, 180, 620, "Canada"},
		{"wrong_country", TilesQuery{"Toronto", "Bangladesh", TilesOrderPopular, 12}, 0, 0, 0, ""},
		{"country_only", TilesQuery{"Canada", "", TilesOrderPopular, 48}, 12, 0, 0, "Canada"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := search.SearchTiles(site, tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.want {
				t.Fatalf("want %d properties, got %d", tc.want, len(got))
			}
			for _, p := range got {
				if p.Location.Country != tc.wantCountry {
					t.Errorf("%s is in %q", p.ID, p.Location.Country)
				}
			}
			if tc.first != 0 && (got[0].PriceFrom != tc.first ||
				got[len(got)-1].PriceFrom != tc.last) {
				t.Errorf("prices run %d..%d; want %d..%d",
					got[0].PriceFrom, got[len(got)-1].PriceFrom, tc.first, tc.last)
			}
		})
	}
}
