package bib

import (
	"strings"
	"testing"
)

func testEntries() map[string]Entry {
	return map[string]Entry{
		"smith2020": {Type: "article", Key: "smith2020", Fields: map[string]string{
			"author": "Smith, John", "title": "First Paper", "year": "2020",
		}},
		"doe2019": {Type: "article", Key: "doe2019", Fields: map[string]string{
			"author": "Doe, Jane", "title": "Second Paper", "year": "2019",
		}},
	}
}

func TestProcess_NumbersAlphabetically(t *testing.T) {
	slides := []string{
		"intro slide, no citations",
		"see [@smith2020] and also [@doe2019]", // Smith cited first
		"cited again: [@smith2020]",
	}
	out := Process(slides, testEntries())

	if len(out) != len(slides)+1 {
		t.Fatalf("got %d slides, want %d (+1 for references)", len(out), len(slides)+1)
	}
	if out[0] != slides[0] {
		t.Errorf("slide 0 should be untouched, got %q", out[0])
	}
	// Doe sorts before Smith, so Doe is [1] and Smith is [2] even though
	// Smith is cited first.
	if !strings.Contains(out[1], "[[2](#smith2020)]") || !strings.Contains(out[1], "[[1](#doe2019)]") {
		t.Errorf("slide 1 = %q, want smith=[[2]] doe=[[1]] (alphabetical)", out[1])
	}
	if !strings.Contains(out[2], "[[2](#smith2020)]") {
		t.Errorf("slide 2 = %q, want repeated citation to reuse [[2]]", out[2])
	}

	refs := out[len(out)-1]
	if !strings.HasPrefix(refs, "## References") {
		t.Errorf("references slide should start with heading, got %q", refs)
	}
	if !strings.Contains(refs, "[1] J. Doe") || !strings.Contains(refs, "[2] J. Smith") {
		t.Errorf("references not numbered alphabetically: %q", refs)
	}
	if strings.Index(refs, "[1] J. Doe") > strings.Index(refs, "[2] J. Smith") {
		t.Errorf("references slide should list [1] before [2]: %q", refs)
	}
}

func TestProcess_UnknownKeyLeftLiteral(t *testing.T) {
	slides := []string{"cites [@missing] key"}
	out := Process(slides, testEntries())

	if len(out) != 1 {
		t.Fatalf("no known citations, should not append a references slide; got %d slides", len(out))
	}
	if out[0] != slides[0] {
		t.Errorf("unresolved citation should be left untouched, got %q", out[0])
	}
}

func TestProcess_NoCitations(t *testing.T) {
	slides := []string{"just text", "more text"}
	out := Process(slides, testEntries())
	if len(out) != len(slides) {
		t.Fatalf("no citations found, slide count should be unchanged")
	}
}
