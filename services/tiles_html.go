package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Block ids are used inside a CSS selector, so only simple ids are
// accepted. The publisher's ids look like "iqskfb" or "id2lqh".
var elementIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ReplaceElementContents parses the page HTML into a DOM, finds each
// element whose id is a key of replacements, and replaces that
// element's inner HTML with the value. The element itself (its tag,
// classes and data attributes) is kept. It returns the ids that were
// not found in the document.
//
// The document is publisher HTML; replacement values must already be
// safe HTML (rendered by html/template).
func ReplaceElementContents(
	document string, replacements map[string]string,
) (string, []string, error) {
	if len(replacements) == 0 {
		return document, nil, nil
	}

	// 1. Parse the HTML text into a DOM. The fragment ends up inside
	//    the <body> of a generated <html><head><body> document.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(document))
	if err != nil {
		return "", nil, fmt.Errorf("parse page html: %w", err)
	}

	ids := make([]string, 0, len(replacements))
	for id := range replacements {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	missing := make([]string, 0)
	replaced := 0

	for _, id := range ids {
		if !elementIDPattern.MatchString(id) {
			return "", nil, fmt.Errorf("invalid element id %q", id)
		}

		// 2. Query the element. [id="…"] also works for ids that
		//    start with a digit, which "#…" does not.
		block := doc.Find(`[id="` + id + `"]`)
		if block.Length() == 0 {
			missing = append(missing, id)
			continue
		}

		// 3. Replace its contents.
		block.SetHtml(replacements[id])
		replaced++
	}

	// Nothing matched: return the original text untouched.
	if replaced == 0 {
		return document, missing, nil
	}

	// 4. Serialize the DOM back to HTML (the <body> contents only).
	out, err := doc.Find("body").Html()
	if err != nil {
		return "", nil, fmt.Errorf("render page html: %w", err)
	}

	return out, missing, nil
}
