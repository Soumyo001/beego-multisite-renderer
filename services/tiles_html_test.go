package services

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestReplaceElementContents(t *testing.T) {
	doc := `<main><div id="a" class="x"><div><div>old</div></div><img src="i.png"></div>` +
		`<section id="b"><p>keep me</p></section><div id="c"><br>text</div></main>`

	got, missing, err := ReplaceElementContents(doc, map[string]string{
		"a": "<b>new A</b>",
		"c": "",
		"z": "<i>nowhere</i>",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := `<main><div id="a" class="x"><b>new A</b></div>` +
		`<section id="b"><p>keep me</p></section><div id="c"></div></main>`
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if !reflect.DeepEqual(missing, []string{"z"}) {
		t.Fatalf("missing = %v; want [z]", missing)
	}
}

func TestReplaceElementContentsNoMatchKeepsText(t *testing.T) {
	doc := `<div id="a"><img src="x.png"/></div>`
	got, missing, err := ReplaceElementContents(doc, map[string]string{"zz": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != doc || !reflect.DeepEqual(missing, []string{"zz"}) {
		t.Fatalf("got %q, %v; want the original text and [zz]", got, missing)
	}
}

func TestReplaceElementContentsRejectsUnsafeID(t *testing.T) {
	if _, _, err := ReplaceElementContents(
		`<div id="a"></div>`, map[string]string{`a"],*[id="b`: ""},
	); err == nil {
		t.Fatal("expected an error for an id that is not a simple name")
	}
}

// Runs against the real publisher page: the two blocks that exist are
// emptied of their 12 dummy tiles, i79o0m is reported missing, and the
// rest of the page (sections, ids, other components) is still there.
func TestReplaceElementContentsHomeJSON(t *testing.T) {
	raw, err := os.ReadFile("../content/beta/preview/test-travel-templates/home.json")
	if err != nil {
		t.Skipf("home.json not available: %v", err)
	}
	var page struct {
		Body struct {
			Content string `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	doc := page.Body.Content
	if strings.Count(doc, "js-property-tile") != 24 {
		t.Skip("home.json no longer has the 24 dummy tiles this test expects")
	}

	got, missing, err := ReplaceElementContents(doc, map[string]string{
		"iqskfb": "<p>DHAKA</p>",
		"id2lqh": "<p>TORONTO</p>",
		"i79o0m": "<p>NOWHERE</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(missing, []string{"i79o0m"}) {
		t.Fatalf("missing = %v; want [i79o0m]", missing)
	}
	if n := strings.Count(got, "js-property-tile"); n != 0 {
		t.Fatalf("%d dummy tiles remain", n)
	}
	for _, s := range []string{
		`id="iqskfb"><p>DHAKA</p></div>`,
		`id="id2lqh"><p>TORONTO</p></div>`,
		`id="hero-main-heading"`, // before the blocks
		`id="tab_view_button_wrapper"`,
		`id="pt_tiles_cta_wrapper"`,
		`id="accordion-feeds"`, // after the blocks
		`id="property-search-section"`,
	} {
		if !strings.Contains(got, s) {
			t.Errorf("result lacks %s", s)
		}
	}

	// Every element id outside the two blocks survives, the same number
	// of times: ids in the original minus ids inside the dummy blocks.
	want := idCounts(t, doc, "")
	for id, n := range idCounts(t, doc, `[id="iqskfb"] [id], [id="id2lqh"] [id]`) {
		want[id] -= n
	}
	for id, n := range idCounts(t, got, "") {
		if want[id] != n {
			t.Errorf("id %q appears %d times; want %d", id, n, want[id])
		}
		delete(want, id)
	}
	for id, n := range want {
		if n != 0 {
			t.Errorf("id %q is missing (%d expected)", id, n)
		}
	}
}

// idCounts counts element ids in the HTML, optionally only those
// matched by selector.
func idCounts(t *testing.T, html, selector string) map[string]int {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if selector == "" {
		selector = "[id]"
	}
	counts := map[string]int{}
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		id, _ := s.Attr("id")
		counts[id]++
	})
	return counts
}
