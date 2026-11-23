package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strconv"
	"strings"

	"sparrow-ssr/services"
)

// flexInt accepts both 12 and "12", because publisher JSON is not
// consistent about number types.
type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	text := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if text == "" || text == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("not an integer: %s", data)
	}
	*f = flexInt(n)
	return nil
}

// flexString accepts both "3" and 3.
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		*f = ""
		return nil
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		text = unquoted
	}
	*f = flexString(text)
	return nil
}

// tilesComponent is one entry of a page's "tiles_component" array.
// Fields we do not use (bounding_box, mode, properties…) are ignored.
type tilesComponent struct {
	Keyword      string     `json:"keyword"`
	TilesPerPage flexInt    `json:"tiles_per_page"`
	TotalTiles   flexInt    `json:"total_tiles"`
	Order        flexString `json:"order"`
	BlockID      string     `json:"tiles_block_id"`
	Type         string     `json:"tiles_component_type"`
}

var tilesBlockIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// tilesBlock is the data for one rendered block.
type tilesBlock struct {
	Keyword string
	PerPage int
	Cards   []countryCard
}

// The same card markup as the /refine and "Vacation Rentals Near"
// cards (refine.css). Titles are <h4> because each home section
// already has an <h3> heading.
var tilesBlockTemplate = template.Must(template.New("tiles").Parse(
	`<div class="refine-grid sparrow-tiles-grid" data-tiles-keyword="{{.Keyword}}" data-per-page="{{.PerPage}}">` +
		`{{range .Cards}}` +
		`<article class="refine-card" data-details-url="{{.DetailsURL}}">` +
		`<div class="refine-card-picture">` +
		`{{if .Photos}}` +
		`<img class="refine-card-image" src="{{index .Photos 0}}" loading="lazy" decoding="async" alt="Illustrative accommodation photograph 1 of {{len .Photos}} for {{.Name}}">` +
		`<div class="refine-carousel-sources" hidden>{{range .Photos}}<span data-photo-url="{{.}}"></span>{{end}}</div>` +
		`{{else}}<div class="refine-image-fallback">Photo unavailable</div>{{end}}` +
		`<span class="refine-price">{{.Price}}</span>` +
		`{{if .Eco}}<span class="refine-eco">Eco</span>{{end}}` +
		`<button type="button" class="refine-favorite" data-property-id="{{.ID}}" data-property-name="{{.Name}}" aria-label="Add {{.Name}} to favorites" aria-pressed="false">` +
		`<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>` +
		`</button>` +
		`</div>` +
		`<div class="refine-card-body">` +
		`<div class="refine-card-meta">` +
		`<span class="refine-rating"><span class="refine-rating-icon" aria-hidden="true">●</span> {{printf "%.1f" .Rating}} ({{.Reviews}} Reviews)</span>` +
		`<span class="refine-property-type">{{.Type}}</span>` +
		`</div>` +
		`<h4 class="refine-card-title"><a class="refine-details-link" href="{{.DetailsURL}}">{{.Name}}</a></h4>` +
		`<p class="refine-card-details">{{.Amenities}}</p>` +
		`<p class="refine-card-location">{{.Location}}</p>` +
		`<div class="refine-card-bottom">` +
		`{{if .ProviderLogo}}<img class="refine-provider-logo" src="{{.ProviderLogo}}" alt="{{.Provider}}" loading="lazy" decoding="async">` +
		`{{else}}<span class="refine-provider">{{.Provider}}</span>{{end}}` +
		`{{if .BookingURL}}<a class="refine-availability" href="{{.BookingURL}}" rel="noopener noreferrer">{{if .IsDemo}}View Other Hotel{{else}}View Availability{{end}}</a>` +
		`{{else}}<span class="country-unavailable">Provider link unavailable</span>{{end}}` +
		`</div>` +
		`</div>` +
		`</article>` +
		`{{end}}` +
		`</div>`,
))

// buildTilesReplacements turns a page's tiles_component JSON into the
// new inner HTML of each block, keyed by tiles_block_id.
//
//   - No tiles_component (missing or null): nil, nil; the page is left as is.
//   - Unreadable tiles_component: an error; the caller leaves the page as is.
//   - A block whose query fails or finds nothing: "" (the dummy tiles are
//     removed and the block is left empty), and the problem is logged.
func buildTilesReplacements(
	siteDirectory string,
	raw json.RawMessage,
	search services.PropertyTilesSearcher,
) (map[string]string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}

	var components []tilesComponent
	if err := json.Unmarshal(raw, &components); err != nil {
		return nil, fmt.Errorf("decode tiles_component: %w", err)
	}

	// Two blocks with the same keyword, order and size share one query.
	results := make(map[services.TilesQuery][]countryCard)

	replacements := make(map[string]string, len(components))

	for _, component := range components {
		id := component.BlockID
		if !tilesBlockIDPattern.MatchString(id) {
			log.Printf("tiles: skipping invalid block id %q", id)
			continue
		}
		if _, seen := replacements[id]; seen {
			log.Printf("tiles: duplicate block id %q; first entry wins", id)
			continue
		}
		if component.Type != "" && component.Type != "property" {
			log.Printf("tiles: block %s has unsupported type %q", id, component.Type)
			continue
		}

		city, country, ok := services.ParseTilesKeyword(component.Keyword)
		if !ok || component.TotalTiles < 1 {
			log.Printf("tiles: block %s has keyword %q and total_tiles %d; leaving it empty",
				id, component.Keyword, component.TotalTiles)
			replacements[id] = ""
			continue
		}

		query := services.TilesQuery{
			City:    city,
			Country: country,
			Order:   services.ParseTilesOrder(string(component.Order)),
			Limit:   int(component.TotalTiles),
		}

		cards, done := results[query]
		if !done {
			properties, err := search.SearchTiles(siteDirectory, query)
			if err != nil {
				log.Printf("tiles: block %s (%q) query failed: %v", id, component.Keyword, err)
				properties = nil
			}
			cards = make([]countryCard, 0, len(properties))
			for _, p := range properties {
				cards = append(cards, newListingCard(p))
			}
			results[query] = cards
		}

		if len(cards) == 0 {
			log.Printf("tiles: block %s (%q) has no properties; leaving it empty", id, component.Keyword)
			replacements[id] = ""
			continue
		}

		var out bytes.Buffer
		if err := tilesBlockTemplate.Execute(&out, tilesBlock{
			Keyword: component.Keyword,
			PerPage: int(component.TilesPerPage),
			Cards:   cards,
		}); err != nil {
			return nil, fmt.Errorf("render tiles block %s: %w", id, err)
		}
		replacements[id] = out.String()
	}

	return replacements, nil
}
