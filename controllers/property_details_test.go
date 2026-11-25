package controllers

import (
	"reflect"
	"testing"

	"beego-multisite-renderer/models"
)

func nearbyTestProperty(
	id, city string, price int, popularity int,
) models.Property {
	return models.Property{
		ID: id,
		Location: models.PropertyLocation{
			City: city, State: city + " State", Country: "USA",
		},
		PriceFrom:       price,
		PopularityScore: popularity,
	}
}

func nearbyIDs(list []models.Property) []string {
	ids := make([]string, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	return ids
}

func TestNearbySortOptions(t *testing.T) {
	for raw, want := range map[string]string{
		"":           "popular",
		"popular":    "popular",
		"price_low":  "price_low",
		"price_high": "price_high",
		"rating_low": "popular",
		"<script>":   "popular",
	} {
		got, options := nearbySortOptions(raw)
		if got != want {
			t.Errorf("nearbySortOptions(%q) = %q; want %q", raw, got, want)
		}

		selected := 0
		for _, option := range options {
			if option.Selected {
				selected++
			}
		}
		if len(options) != 3 || selected != 1 {
			t.Errorf("nearbySortOptions(%q): %d options, %d selected", raw, len(options), selected)
		}
	}
}

func TestNearbySelectionAndSort(t *testing.T) {
	// Already in SearchCountry order: popularity descending.
	all := []models.Property{
		nearbyTestProperty("far-1", "New York", 890, 900),
		nearbyTestProperty("vegas-1", "Las Vegas", 200, 800),
		nearbyTestProperty("self", "Las Vegas", 360, 700),
		nearbyTestProperty("far-2", "Miami", 150, 600),
		nearbyTestProperty("vegas-2", "Las Vegas", 120, 500),
		nearbyTestProperty("far-3", "Orlando", 100, 400),
	}
	selected := all[2]

	for mode, want := range map[string][]string{
		// Same-city first, then two top-ups to reach four.
		"popular":    {"vegas-1", "vegas-2", "far-1", "far-2"},
		"price_low":  {"vegas-2", "vegas-1", "far-2", "far-1"},
		"price_high": {"vegas-1", "vegas-2", "far-1", "far-2"},
	} {
		got := selectNearby(all, selected)
		sortNearby(got, mode, selected)

		if ids := nearbyIDs(got); !reflect.DeepEqual(ids, want) {
			t.Errorf("%s: got %v; want %v", mode, ids, want)
		}
	}
}

func TestProviderLogo(t *testing.T) {
	for provider, want := range map[string]string{
		"Booking.com":  "/static/img/partners/booking.svg",
		" expedia ":    "/static/img/partners/expedia.svg",
		"Expedia.com":  "/static/img/partners/expedia.svg",
		"Other Travel": "",
	} {
		if got := providerLogo(provider); got != want {
			t.Errorf("providerLogo(%q) = %q; want %q", provider, got, want)
		}
	}
}

func TestSizedPhoto(t *testing.T) {
	cases := []struct{ raw, want string }{
		{
			"https://images.unsplash.com/photo-1",
			"https://images.unsplash.com/photo-1?auto=format&fit=crop&q=70&w=640",
		},
		{
			"https://images.unsplash.com/photo-2?auto=format&fit=crop&w=1000&q=75",
			"https://images.unsplash.com/photo-2?auto=format&fit=crop&q=70&w=640",
		},
		{
			"https://cdn.example.com/a.jpg?w=2000",
			"https://cdn.example.com/a.jpg?w=2000",
		},
	}

	for _, tc := range cases {
		if got := sizedPhoto(tc.raw, cardPhotoWidth); got != tc.want {
			t.Errorf("sizedPhoto(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
