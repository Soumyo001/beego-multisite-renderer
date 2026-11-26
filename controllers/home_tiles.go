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
	"sync"
	texttemplate "text/template"

	"beego-multisite-renderer/services"
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
	SiteKey string
	PerPage int // cards shown per page, 1..len(Cards)
	Pages   int
	Columns int // grid columns on wide screens, 1..4
	Cards   []countryCard
}

// newTilesBlock applies tiles_per_page: an invalid value or one larger
// than the result count shows every card on a single page.
func newTilesBlock(keyword, siteKey string, perPage int, cards []countryCard) tilesBlock {
	if perPage < 1 || perPage > len(cards) {
		perPage = len(cards)
	}
	columns := min(perPage, 4)
	return tilesBlock{
		Keyword: keyword,
		SiteKey: siteKey,
		PerPage: perPage,
		Pages:   (len(cards) + perPage - 1) / perPage,
		Columns: columns,
		Cards:   cards,
	}
}

// The same card markup as the /refine and "Vacation Rentals Near"
// cards (refine.css). Titles are <h4> because each home section
// already has an <h3> heading.
var tilesBlockTemplate = template.Must(template.New("tiles").Parse(
	`<div class="bmr-tiles" data-site-key="{{.SiteKey}}" data-per-page="{{.PerPage}}">` +
		`<div class="refine-grid bmr-tiles-grid" data-tiles-keyword="{{.Keyword}}" style="--bmr-tiles-columns: {{.Columns}}">` +
		`{{range $i, $card := .Cards}}{{with $card}}` +
		`<article class="refine-card" data-details-url="{{.DetailsURL}}"{{if ge $i $.PerPage}} hidden{{end}}>` +
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
		`{{end}}{{end}}` +
		`</div>` +
		`{{if gt .Pages 1}}` +
		`<div class="bmr-tiles-pager">` +
		`<button type="button" class="refine-page-button" data-tiles-step="-1" aria-label="Previous properties" disabled>&lsaquo;</button>` +
		`<span class="bmr-tiles-status" aria-live="polite">1 / {{.Pages}}</span>` +
		`<button type="button" class="refine-page-button" data-tiles-step="1" aria-label="Next properties">&rsaquo;</button>` +
		`</div>` +
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
		block := newTilesBlock(
			component.Keyword, siteDirectory, int(component.TilesPerPage), cards,
		)
		if err := tilesBlockTemplate.Execute(&out, block); err != nil {
			return nil, fmt.Errorf("render tiles block %s: %w", id, err)
		}
		replacements[id] = out.String()
	}

	return replacements, nil
}

// tilesSearch is set once at start-up, before Beego serves requests.
var tilesSearch services.PropertyTilesSearcher

// SetTilesDependencies wires the database query used by tile blocks.
func SetTilesDependencies(search services.PropertyTilesSearcher) {
	tilesSearch = search
}

// pageBodyWithTiles fills the page's tile blocks from tiles_component.
// It returns the publisher body unchanged (and false) when the page has
// no tiles_component or anything goes wrong, so a tiles problem never
// breaks the page itself.
func pageBodyWithTiles(
	siteDirectory, body string,
	raw json.RawMessage,
	search services.PropertyTilesSearcher,
) (string, bool) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return body, false
	}
	if search == nil {
		logTilesOnce("tiles: no property search configured; keeping the publisher blocks")
		return body, false
	}

	replacements, err := buildTilesReplacements(siteDirectory, raw, search)
	if err != nil {
		log.Printf("tiles: %v; keeping the publisher blocks", err)
		return body, false
	}
	if len(replacements) == 0 {
		return body, false
	}

	out, missing, err := services.ReplaceElementContents(body, replacements)
	if err != nil {
		log.Printf("tiles: %v; keeping the publisher blocks", err)
		return body, false
	}
	if len(missing) > 0 {
		logTilesOnce(fmt.Sprintf(
			"tiles: %s: tiles_block_id not found in the page: %s",
			siteDirectory, strings.Join(missing, ", "),
		))
	}
	if len(missing) == len(replacements) {
		return body, false
	}

	return out, true
}

// buildTileSlots runs buildTilesReplacements and returns its result keyed
// by tiles_block_id, ready to bind to the marked page body. Blocks without
// an entry (or no tiles_component at all) keep the publisher's tiles.
func buildTileSlots(
	siteDirectory string,
	raw json.RawMessage,
	search services.PropertyTilesSearcher,
) map[string]string {
	slots := make(map[string]string)
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return slots
	}

	if search == nil {
		logTilesOnce("tiles: no property search configured; keeping the publisher blocks")
		return slots
	}

	replacements, err := buildTilesReplacements(siteDirectory, raw, search)
	if err != nil {
		log.Printf("tiles: %v; keeping the publisher blocks", err)
		return slots
	}

	return replacements
}

// Delimiters of the tile markers that scripts/tilemarkers writes into the
// page HTML. They look like HTML comments, so a marked body is still valid
// HTML and cannot clash with the publisher's markup or scripts.
const (
	tilesLeftDelim  = "<!--{{"
	tilesRightDelim = "}}-->"
)

// bodyTemplate is one page body, parsed once.
type bodyTemplate struct {
	source string                 // the body it was parsed from
	tmpl   *texttemplate.Template // nil: no markers, serve source as is
}

// bodyTemplates caches parsed page bodies by page JSON path.
var bodyTemplates sync.Map

// pageBodyTemplate returns the parsed body for key, parsing it only when it
// is new or its text changed. The loader returns the same string each
// time, so the comparison with the cached source is a pointer check.
func pageBodyTemplate(key, body string) *bodyTemplate {
	if cached, ok := bodyTemplates.Load(key); ok {
		if entry := cached.(*bodyTemplate); entry.source == body {
			return entry
		}
	}

	entry := &bodyTemplate{source: body}
	if strings.Contains(body, tilesLeftDelim) {
		tmpl, err := texttemplate.New(key).
			Delims(tilesLeftDelim, tilesRightDelim).
			Parse(body)
		if err != nil {
			log.Printf("tiles: %s: tile markers cannot be parsed; showing the publisher blocks: %v", key, err)
		} else {
			entry.tmpl = tmpl
		}
	}

	bodyTemplates.Store(key, entry)
	return entry
}

// renderMarkedBody executes the marked page body with the tile HTML bound
// by block id. A block whose id is missing or empty in tiles shows the
// publisher's original tiles. An unmarked body is returned as it is.
func renderMarkedBody(pageKey, body string, tiles map[string]string) string {
	entry := pageBodyTemplate(pageKey, body)
	if entry.tmpl == nil {
		if len(tiles) > 0 {
			logTilesOnce(fmt.Sprintf(
				"tiles: %s has no tile markers; run: go run ./scripts/tilemarkers -write <page.json>",
				pageKey,
			))
		}
		return body
	}

	size := len(body)
	for _, html := range tiles {
		size += len(html)
	}

	var out strings.Builder
	out.Grow(size)
	data := struct{ Tiles map[string]string }{Tiles: tiles}
	if err := entry.tmpl.Execute(&out, data); err != nil {
		// The marked body itself shows the publisher's tiles.
		log.Printf("tiles: %s: render page body: %v", pageKey, err)
		return body
	}
	return out.String()
}

// A block id missing from the page would otherwise be logged on every
// request; log each distinct message once per process.
var tilesLogged sync.Map

func logTilesOnce(message string) {
	if _, seen := tilesLogged.LoadOrStore(message, struct{}{}); !seen {
		log.Print(message)
	}
}
