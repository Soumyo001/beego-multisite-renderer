package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"beego-multisite-renderer/models"
	"beego-multisite-renderer/services"
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

		// tiles_per_page 4 of 12: cards 5-12 start hidden, 3 pages.
		if n := strings.Count(block, `" hidden><div class="refine-card-picture">`); n != 8 {
			t.Errorf("%s has %d hidden cards; want 8", id, n)
		}
		if !strings.Contains(block, `data-site-key="site"`) ||
			!strings.Contains(block, `--bmr-tiles-columns: 4`) ||
			!strings.Contains(block, `>1 / 3</span>`) {
			t.Errorf("%s pager markup is unexpected:\n%.600s", id, block)
		}
	}
}

func TestNewTilesBlock(t *testing.T) {
	cards := make([]countryCard, 5)
	for _, tc := range []struct{ perPage, wantPer, wantPages, wantCols int }{
		{4, 4, 2, 4},
		{2, 2, 3, 2},
		{0, 5, 1, 4},  // invalid: everything on one page
		{12, 5, 1, 4}, // more than the results
	} {
		b := newTilesBlock("k", "s", tc.perPage, cards)
		if b.PerPage != tc.wantPer || b.Pages != tc.wantPages || b.Columns != tc.wantCols {
			t.Errorf("perPage %d: got %d/%d/%d; want %d/%d/%d", tc.perPage,
				b.PerPage, b.Pages, b.Columns, tc.wantPer, tc.wantPages, tc.wantCols)
		}
	}
}

func TestPageBodyWithTiles(t *testing.T) {
	body := `<div class="pres__tiles-container" id="id2lqh"><p>dummy tile</p></div><p id="x">keep</p>`

	got, ok := pageBodyWithTiles("site", body, json.RawMessage(tilesJSON), &fakeTilesSearcher{})
	if !ok || strings.Contains(got, "dummy tile") ||
		!strings.Contains(got, `<p id="x">keep</p>`) ||
		strings.Count(got, `<article class="refine-card"`) != 12 {
		t.Fatalf("ok=%v body:\n%.500s", ok, got)
	}

	// No tiles_component, no searcher, bad JSON, or no block found:
	// the publisher body is returned unchanged.
	for name, tc := range map[string]struct {
		raw    string
		search services.PropertyTilesSearcher
		body   string
	}{
		"missing":   {"", &fakeTilesSearcher{}, body},
		"null":      {"null", &fakeTilesSearcher{}, body},
		"no search": {tilesJSON, nil, body},
		"bad json":  {`{"x":1}`, &fakeTilesSearcher{}, body},
		"no blocks": {tilesJSON, &fakeTilesSearcher{}, `<p id="x">keep</p>`},
	} {
		got, ok := pageBodyWithTiles("site", tc.body, json.RawMessage(tc.raw), tc.search)
		if ok || got != tc.body {
			t.Errorf("%s: ok=%v, body changed to %.200s", name, ok, got)
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
