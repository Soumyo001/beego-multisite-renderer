package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestInternationalDemoSeedProperties(t *testing.T) {
	international, err := internationalDemoSeedProperties()
	if err != nil {
		t.Fatal(err)
	}
	if len(international) != 24 {
		t.Fatalf("want 24 international properties, got %d", len(international))
	}
	if international[0].ID != "prop-049" ||
		international[23].ID != "prop-072" {
		t.Fatalf("unexpected ID range %s..%s",
			international[0].ID, international[23].ID)
	}

	pool := make(map[string]bool)
	for _, id := range verifiedPhotoPool() {
		pool[id] = true
	}

	perCity := map[string]int{}
	for _, p := range international {
		key := p.Location.City + "/" + p.Location.Country
		perCity[key]++

		if !strings.HasSuffix(p.Name, " (Demo)") ||
			!strings.HasPrefix(p.Slug, "demo-") {
			t.Errorf("%s lost its demo identity", p.ID)
		}
		if p.Currency != "USD" || p.ProviderName != "Booking.com" {
			t.Errorf("%s: currency %q provider %q", p.ID, p.Currency, p.ProviderName)
		}

		// Each listing links to a hotel page in its own country.
		u, err := url.Parse(p.BookingURL)
		want := map[string]string{"Canada": "/hotel/ca/", "Bangladesh": "/hotel/bd/"}[p.Location.Country]
		if err != nil || u.Scheme != "https" || u.Host != "www.booking.com" ||
			want == "" || !strings.HasPrefix(u.Path, want) {
			t.Errorf("%s: unexpected booking URL %q", p.ID, p.BookingURL)
		}

		// Five distinct, already-verified photos.
		seen := map[string]bool{}
		for _, img := range p.Images {
			id := strings.TrimPrefix(strings.SplitN(img, "?", 2)[0],
				"https://images.unsplash.com/")
			if !pool[id] || seen[id] {
				t.Errorf("%s: unverified or repeated photo %q", p.ID, img)
			}
			seen[id] = true
		}
		if len(p.Images) != 5 {
			t.Errorf("%s has %d images", p.ID, len(p.Images))
		}

		if _, err := propertyDetailsForSeed(p); err != nil {
			t.Errorf("%s: %v", p.ID, err)
		}
	}

	if perCity["Toronto/Canada"] != 12 || perCity["Dhaka/Bangladesh"] != 12 {
		t.Fatalf("unexpected city split: %v", perCity)
	}
}

func TestAll72SeedPropertiesValid(t *testing.T) {
	original, err := gallerySeedProperties()
	if err != nil {
		t.Fatal(err)
	}
	demos, err := additionalDemoSeedProperties(original)
	if err != nil {
		t.Fatal(err)
	}
	international, err := internationalDemoSeedProperties()
	if err != nil {
		t.Fatal(err)
	}

	all := append(append(append([]seedProperty(nil), original...), demos...), international...)
	if len(all) != 72 {
		t.Fatalf("want 72, got %d", len(all))
	}
	// Also rejects duplicate IDs and slugs across all three groups.
	if err := validateSeedData(all); err != nil {
		t.Fatal(err)
	}
}
