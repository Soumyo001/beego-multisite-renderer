package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"sparrow-ssr/models"
	"sparrow-ssr/services"
)

// fakeTilesSearcher records queries and returns n properties per query.
type fakeTilesSearcher struct {
	queries []services.TilesQuery
	fail    bool
}

func (f *fakeTilesSearcher) SearchTiles(
	site string, q services.TilesQuery,
) ([]models.Property, error) {
	f.queries = append(f.queries, q)
	if f.fail {
		return nil, errors.New("database down")
	}
	if strings.EqualFold(q.City, "Atlantis") {
		return nil, nil
	}
	out := make([]models.Property, 0, q.Limit)
	for i := 0; i < q.Limit; i++ {
		out = append(out, models.Property{
			ID:           fmt.Sprintf("prop-%03d", i+1),
			Slug:         fmt.Sprintf("demo-%s-%d", strings.ToLower(q.City), i),
			Name:         fmt.Sprintf("%s <Stay> %d (Demo)", q.City, i),
			PropertyType: "Hotel",
			Location:     models.PropertyLocation{City: q.City, Area: "Centre", State: "S", Country: q.Country},
			PriceFrom:    100 + i, Currency: "USD", Rating: 8.5, ReviewCount: 10,
			Amenities:    []string{"Pool"},
			ProviderName: "Booking.com",
			BookingURL:   "https://www.booking.com/hotel/ca/example.html",
			Images:       []string{"https://images.unsplash.com/photo-1"},
		})
	}
	return out, nil
}

const tilesJSON = `[
 {"keyword":"Toronto, ON, Canada","tiles_per_page":4,"total_tiles":12,"order":"3","tiles_block_id":"i79o0m","tiles_component_type":"property"},
 {"keyword":"Toronto, ON, Canada","tiles_per_page":4,"total_tiles":12,"order":"3","tiles_block_id":"id2lqh","tiles_component_type":"property"},
 {"keyword":"Dhaka, Bangladesh","tiles_per_page":"4","total_tiles":"12","order":2,"tiles_block_id":"iqskfb","tiles_component_type":"property"}
]`

func TestBuildTilesReplacements(t *testing.T) {
	search := &fakeTilesSearcher{}
	got, err := buildTilesReplacements("site", json.RawMessage(tilesJSON), search)
	if err != nil {
		t.Fatal(err)
	}

	// The two identical Toronto blocks share one query.
	want := []services.TilesQuery{
		{City: "Toronto", Country: "Canada", Order: services.TilesOrderPriceHigh, Limit: 12},
		{City: "Dhaka", Country: "Bangladesh", Order: services.TilesOrderPriceLow, Limit: 12},
	}
	if fmt.Sprint(search.queries) != fmt.Sprint(want) {
		t.Fatalf("queries = %v; want %v", search.queries, want)
	}

	for _, id := range []string{"i79o0m", "id2lqh", "iqskfb"} {
		block := got[id]
		if n := strings.Count(block, `<article class="refine-card"`); n != 12 {
			t.Errorf("%s has %d cards; want 12", id, n)
		}
		if !strings.Contains(block, `data-per-page="4"`) ||
			!strings.Contains(block, "View Other Hotel") ||
			!strings.Contains(block, `href="/property/demo-`) ||
			// Names are escaped, never raw HTML.
			strings.Contains(block, "<Stay>") ||
			!strings.Contains(block, "&lt;Stay&gt;") {
			t.Errorf("%s block markup is unexpected:\n%.400s", id, block)
		}
	}
}

func TestBuildTilesReplacementsEmptyCases(t *testing.T) {
	// No tiles_component at all: leave the page alone.
	for _, raw := range []string{"", "null"} {
		got, err := buildTilesReplacements("site", json.RawMessage(raw), &fakeTilesSearcher{})
		if err != nil || got != nil {
			t.Errorf("%q: got %v, %v; want nil, nil", raw, got, err)
		}
	}

	// Unreadable JSON is an error; the caller keeps the page as is.
	if _, err := buildTilesReplacements("site", json.RawMessage(`{"x":1}`), &fakeTilesSearcher{}); err == nil {
		t.Error("expected a decode error")
	}

	// Failed or empty queries and bad entries give empty blocks or are skipped.
	raw := `[
	 {"keyword":"Atlantis","total_tiles":12,"tiles_block_id":"none"},
	 {"keyword":"","total_tiles":12,"tiles_block_id":"nokey"},
	 {"keyword":"Dhaka","total_tiles":12,"tiles_block_id":"bad id!"},
	 {"keyword":"Dhaka","total_tiles":12,"tiles_block_id":"other","tiles_component_type":"blog"}
	]`
	got, err := buildTilesReplacements("site", json.RawMessage(raw), &fakeTilesSearcher{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["none"] != "" || got["nokey"] != "" {
		t.Errorf("got %v; want only empty none and nokey", got)
	}

	failing := &fakeTilesSearcher{fail: true}
	got, err = buildTilesReplacements("site", json.RawMessage(tilesJSON), failing)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["iqskfb"] != "" {
		t.Errorf("failed queries should give 3 empty blocks, got %v", got)
	}
}
