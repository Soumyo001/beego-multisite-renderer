package main

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"text/template"
)

func TestMarkBlocks(t *testing.T) {
	content := `<section><div id="a" class="x"><div><img src="1.png"><br/>old A</div>` +
		`<script>if (x < 1) { document.write("</div>") }</script></div>` +
		`<p id="keep">keep</p><div id="b"></div></section>`

	got, err := markBlocks(content, []string{"a", "b", "zz"})
	if err != nil {
		t.Fatal(err)
	}

	slot := func(id string) string {
		return `<!--{{if index .Tiles "` + id + `"}}--><!--{{index .Tiles "` + id + `"}}--><!--{{else}}-->`
	}
	want := `<section><div id="a" class="x">` + slot("a") +
		`<div><img src="1.png"><br/>old A</div>` +
		`<script>if (x < 1) { document.write("</div>") }</script><!--{{end}}--></div>` +
		`<p id="keep">keep</p><div id="b">` + slot("b") + `<!--{{end}}--></div></section>`

	if got.Content != want {
		t.Fatalf("got\n%s\nwant\n%s", got.Content, want)
	}
	if !reflect.DeepEqual(got.Marked, []string{"a", "b"}) ||
		!reflect.DeepEqual(got.Missing, []string{"zz"}) || got.Already != nil {
		t.Fatalf("marked %v, missing %v, already %v", got.Marked, got.Missing, got.Already)
	}
	if err := verify(content, got.Content, got.Marked); err != nil {
		t.Fatal(err)
	}

	// Running again changes nothing.
	again, err := markBlocks(got.Content, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Content != got.Content || !reflect.DeepEqual(again.Already, []string{"a", "b"}) {
		t.Fatalf("second run changed the content or marked again: %+v", again)
	}
}

func TestMarkBlocksRejectsBadHTML(t *testing.T) {
	for name, content := range map[string]string{
		"unclosed":  `<div id="a"><div>text</div>`,
		"mismatch":  `<div id="a"><span>text</div></span>`,
		"void":      `<img id="a" src="x.png">`,
		"duplicate": `<div id="a"></div><div id="a"></div>`,
	} {
		if _, err := markBlocks(content, []string{"a"}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// Marks the real publisher home page and checks the written JSON.
func TestHomeJSON(t *testing.T) {
	data, err := os.ReadFile("../../content/beta/preview/test-travel-templates/home.json")
	if err != nil {
		t.Skipf("home.json not available: %v", err)
	}

	ids, err := tileBlockIDs(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"i79o0m", "id2lqh", "iqskfb"}) {
		t.Fatalf("ids = %v", ids)
	}

	content, err := bodyContent(data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, leftDelim) {
		t.Skip("home.json is already marked")
	}

	result, err := markBlocks(content, ids)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Marked, []string{"id2lqh", "iqskfb"}) ||
		!reflect.DeepEqual(result.Missing, []string{"i79o0m"}) {
		t.Fatalf("marked %v, missing %v", result.Marked, result.Missing)
	}
	if err := verify(content, result.Content, result.Marked); err != nil {
		t.Fatal(err)
	}

	// With both blocks shown, none of the 24 dummy tiles remain.
	shown, err := render(result.Content, map[string]string{
		"id2lqh": "<p>TORONTO</p>",
		"iqskfb": "<p>DHAKA</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(shown, "js-property-tile"); n != 0 {
		t.Fatalf("%d dummy tiles remain", n)
	}
	for _, s := range []string{`id="iqskfb"><p>DHAKA</p></div>`, `id="id2lqh"><p>TORONTO</p></div>`} {
		if !strings.Contains(shown, s) {
			t.Errorf("rendered page lacks %s", s)
		}
	}

	// Only body.content changed in the JSON.
	updated, err := replaceBodyContent(data, result.Content)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]any
	if err := json.Unmarshal(data, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(updated, &after); err != nil {
		t.Fatal(err)
	}
	if got := after["body"].(map[string]any)["content"]; got != result.Content {
		t.Fatal("body.content was not replaced")
	}
	after["body"].(map[string]any)["content"] = content
	if !reflect.DeepEqual(before, after) {
		t.Fatal("other JSON fields changed")
	}
	if !strings.HasPrefix(string(updated), `{"head": {"title": `) {
		t.Fatalf("field order or format changed: %.60s", updated)
	}
}

func BenchmarkRenderMarkedHome(b *testing.B) {
	data, err := os.ReadFile("../../content/beta/preview/test-travel-templates/home.json")
	if err != nil {
		b.Skip(err)
	}
	ids, _ := tileBlockIDs(data)
	content, _ := bodyContent(data)
	result, err := markBlocks(content, ids)
	if err != nil {
		b.Fatal(err)
	}
	tiles := map[string]string{
		"id2lqh": strings.Repeat("x", 20000),
		"iqskfb": strings.Repeat("y", 20000),
	}
	// Parsed once, like the server; only execution is measured.
	tmpl := mustParse(b, result.Content)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out strings.Builder
		out.Grow(len(content) + 40000)
		if err := tmpl.Execute(&out, struct{ Tiles map[string]string }{tiles}); err != nil {
			b.Fatal(err)
		}
	}
}

func mustParse(b *testing.B, content string) interface {
	Execute(w io.Writer, data any) error
} {
	b.Helper()
	tmpl, err := template.New("page").Delims(leftDelim, rightDelim).Parse(content)
	if err != nil {
		b.Fatal(err)
	}
	return tmpl
}
