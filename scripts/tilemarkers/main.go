// Command tilemarkers prepares a publisher page JSON for property tiles.
//
// For every block listed in the page's tiles_component, it wraps the
// block's existing inner HTML (the publisher's dummy tiles) in template
// markers, so the server can show its own cards or the original tiles
// without searching the page at request time:
//
//	<div id="iqskfb" ...><!--{{if index .Tiles "iqskfb"}}-->
//	<!--{{index .Tiles "iqskfb"}}--><!--{{else}}-->
//	...original inner HTML...<!--{{end}}--></div>
//
// The server binds the tile HTML by block id; an id that is missing or
// empty shows the original inner HTML.
//
// Only body.content changes; every other byte of the HTML and every other
// JSON field is kept. Running it again on a marked file changes nothing.
//
// Usage (from the project root):
//
//	go run ./scripts/tilemarkers content/beta/preview/test-travel-templates/home.json
//	go run ./scripts/tilemarkers -write content/beta/preview/test-travel-templates/home.json
//
// Without -write it only reports what it would do.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/net/html"
)

// Must match tilesLeftDelim / tilesRightDelim in controllers/home_tiles.go.
const (
	leftDelim  = "<!--{{"
	rightDelim = "}}-->"
)

var blockIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func main() {
	write := flag.Bool("write", false, "write the marked file (default: report only)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/tilemarkers [-write] page.json ...")
		os.Exit(2)
	}

	failed := false
	for _, file := range flag.Args() {
		if err := processFile(file, *write); err != nil {
			log.Printf("%s: %v", file, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func processFile(file string, write bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	ids, err := tileBlockIDs(data)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		log.Printf("%s: no tiles_component blocks; nothing to do", file)
		return nil
	}

	content, err := bodyContent(data)
	if err != nil {
		return err
	}

	result, err := markBlocks(content, ids)
	if err != nil {
		return err
	}
	for _, id := range result.Marked {
		log.Printf("%s: block %s marked", file, id)
	}
	for _, id := range result.Already {
		log.Printf("%s: block %s already marked", file, id)
	}
	for _, id := range result.Missing {
		log.Printf("%s: block %s is in tiles_component but not in the HTML; skipped", file, id)
	}

	if err := verify(content, result.Content, result.Marked); err != nil {
		return fmt.Errorf("verification failed, file not written: %w", err)
	}

	if len(result.Marked) == 0 {
		log.Printf("%s: unchanged", file)
		return nil
	}
	if !write {
		log.Printf("%s: dry run; add -write to save", file)
		return nil
	}

	updated, err := replaceBodyContent(data, result.Content)
	if err != nil {
		return err
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, updated, info.Mode().Perm()); err != nil {
		return err
	}
	log.Printf("%s: written", file)
	return nil
}

// tileBlockIDs returns the distinct, valid tiles_block_id values of
// property tile blocks, in file order.
func tileBlockIDs(data []byte) ([]string, error) {
	var page struct {
		Tiles []struct {
			BlockID string `json:"tiles_block_id"`
			Type    string `json:"tiles_component_type"`
		} `json:"tiles_component"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("read tiles_component: %w", err)
	}

	seen := make(map[string]bool)
	ids := make([]string, 0, len(page.Tiles))
	for _, tile := range page.Tiles {
		id := tile.BlockID
		switch {
		case tile.Type != "" && tile.Type != "property":
		case !blockIDPattern.MatchString(id):
			log.Printf("skipping invalid tiles_block_id %q", id)
		case !seen[id]:
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func bodyContent(data []byte) (string, error) {
	var page struct {
		Body struct {
			Content *string `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return "", fmt.Errorf("read body.content: %w", err)
	}
	if page.Body.Content == nil {
		return "", errors.New("body.content is missing")
	}
	return *page.Body.Content, nil
}

// markResult is the outcome of markBlocks.
type markResult struct {
	Content string
	Marked  []string // wrapped now
	Already []string // wrapped by an earlier run
	Missing []string // no element with this id
}

// innerSpan is the byte range of one element's inner HTML.
type innerSpan struct {
	id         string
	start, end int
}

// markBlocks wraps the inner HTML of each element whose id is in ids.
func markBlocks(content string, ids []string) (markResult, error) {
	spans, err := findInnerSpans(content, ids)
	if err != nil {
		return markResult{}, err
	}

	var result markResult
	var out strings.Builder
	last := 0

	for _, id := range ids {
		span, found := spans[id]
		if !found {
			result.Missing = append(result.Missing, id)
			continue
		}
		if strings.HasPrefix(content[span.start:], leftDelim) {
			result.Already = append(result.Already, id)
			continue
		}
		result.Marked = append(result.Marked, id)
	}

	// Write the spans in document order.
	for _, span := range orderedSpans(spans, result.Marked) {
		tile := fmt.Sprintf(`index .Tiles %q`, span.id)
		out.WriteString(content[last:span.start])
		out.WriteString(leftDelim + "if " + tile + rightDelim)
		out.WriteString(leftDelim + tile + rightDelim)
		out.WriteString(leftDelim + "else" + rightDelim)
		out.WriteString(content[span.start:span.end])
		out.WriteString(leftDelim + "end" + rightDelim)
		last = span.end
	}
	out.WriteString(content[last:])

	result.Content = out.String()
	return result, nil
}

func orderedSpans(spans map[string]innerSpan, ids []string) []innerSpan {
	list := make([]innerSpan, 0, len(ids))
	for _, id := range ids {
		list = append(list, spans[id])
	}
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].start < list[j-1].start; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
	return list
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// findInnerSpans tokenizes the HTML once and returns, for each wanted id,
// the byte range between the element's start tag and its matching end
// tag. It counts the exact bytes of every token, so the ranges point into
// the original text.
func findInnerSpans(content string, ids []string) (map[string]innerSpan, error) {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}

	spans := make(map[string]innerSpan)
	z := html.NewTokenizer(strings.NewReader(content))
	offset := 0

	type open struct {
		id, tag string
		start   int
		depth   int
	}
	var current *open

	for {
		tokenType := z.Next()
		tokenStart := offset
		offset += len(z.Raw())

		switch tokenType {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				return nil, fmt.Errorf("tokenize HTML: %w", z.Err())
			}
			if current != nil {
				return nil, fmt.Errorf("block %s: end tag </%s> not found", current.id, current.tag)
			}
			if offset != len(content) {
				return nil, fmt.Errorf("tokenizer covered %d of %d bytes", offset, len(content))
			}
			return spans, nil

		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			tag := string(name)

			if current != nil {
				if tokenType == html.StartTagToken && !voidElements[tag] {
					current.depth++
				}
				continue
			}

			id := idAttribute(z, hasAttr)
			if !wanted[id] {
				continue
			}
			if _, dup := spans[id]; dup {
				return nil, fmt.Errorf("block %s: id appears more than once", id)
			}
			if tokenType == html.SelfClosingTagToken || voidElements[tag] {
				return nil, fmt.Errorf("block %s: <%s> cannot contain HTML", id, tag)
			}
			current = &open{id: id, tag: tag, start: offset, depth: 1}

		case html.EndTagToken:
			if current == nil {
				continue
			}
			current.depth--
			if current.depth > 0 {
				continue
			}
			if name, _ := z.TagName(); string(name) != current.tag {
				return nil, fmt.Errorf("block %s: <%s> closed by </%s>; HTML is not well formed",
					current.id, current.tag, name)
			}
			spans[current.id] = innerSpan{id: current.id, start: current.start, end: tokenStart}
			current = nil
		}
	}
}

func idAttribute(z *html.Tokenizer, hasAttr bool) string {
	for hasAttr {
		var key, value []byte
		key, value, hasAttr = z.TagAttr()
		if string(key) == "id" {
			return string(value)
		}
	}
	return ""
}

// render parses a marked body exactly like the server and executes it
// with the tile HTML bound by block id.
func render(content string, tiles map[string]string) (string, error) {
	tmpl, err := template.New("page").Delims(leftDelim, rightDelim).Parse(content)
	if err != nil {
		return "", fmt.Errorf("markers do not parse: %w", err)
	}
	var out strings.Builder
	err = tmpl.Execute(&out, struct{ Tiles map[string]string }{tiles})
	return out.String(), err
}

// verify checks that the marked HTML renders the original page when no
// tiles are shown, and our HTML in every newly marked block when they are.
func verify(original, marked string, ids []string) error {
	// An original marked by an earlier run is compared in its rendered form.
	want, err := render(original, nil)
	if err != nil {
		return err
	}
	plain, err := render(marked, nil)
	if err != nil {
		return err
	}
	if plain != want {
		return errors.New("without tiles the page differs from the original")
	}

	tiles := make(map[string]string, len(ids))
	for _, id := range ids {
		tiles[id] = "<!--tile " + id + "-->"
	}
	shown, err := render(marked, tiles)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if !strings.Contains(shown, "<!--tile "+id+"-->") {
			return fmt.Errorf("block %s does not render", id)
		}
	}
	return nil
}

// replaceBodyContent returns data with body.content replaced. The other
// fields keep their original bytes and order.
func replaceBodyContent(data []byte, content string) ([]byte, error) {
	top, err := readObject(data)
	if err != nil {
		return nil, err
	}

	for i := range top {
		if top[i].key != "body" {
			continue
		}
		body, err := readObject(top[i].value)
		if err != nil {
			return nil, fmt.Errorf("body: %w", err)
		}
		for j := range body {
			if body[j].key == "content" {
				encoded, err := encodeString(content)
				if err != nil {
					return nil, err
				}
				body[j].value = encoded
				top[i].value = writeObject(body)
				return writeObject(top), nil
			}
		}
	}
	return nil, errors.New("body.content is missing")
}

type field struct {
	key   string
	value json.RawMessage
}

func readObject(data []byte) ([]field, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if token, err := dec.Token(); err != nil || token != json.Delim('{') {
		return nil, errors.New("expected a JSON object")
	}

	var fields []field
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := token.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		fields = append(fields, field{key: key, value: value})
	}
	return fields, nil
}

// writeObject uses the same ", " and ": " separators as the publisher files.
func writeObject(fields []field) []byte {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			out.WriteString(", ")
		}
		key, _ := encodeString(f.key)
		out.Write(key)
		out.WriteString(": ")
		out.Write(f.value)
	}
	out.WriteByte('}')
	return out.Bytes()
}

func encodeString(s string) ([]byte, error) {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return bytes.TrimRight(out.Bytes(), "\n"), nil
}
