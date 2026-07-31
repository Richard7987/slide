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

func TestProcess_NumbersByFirstAppearance(t *testing.T) {
	slides := []string{
		"intro slide, no citations",
		"see [@doe2019] and also [@smith2020]",
		"cited again: [@doe2019]",
	}
	out := Process(slides, testEntries())

	if len(out) != len(slides)+1 {
		t.Fatalf("got %d slides, want %d (+1 for references)", len(out), len(slides)+1)
	}
	if out[0] != slides[0] {
		t.Errorf("slide 0 should be untouched, got %q", out[0])
	}
	if !strings.Contains(out[1], "[[1](#doe2019)]") || !strings.Contains(out[1], "[[2](#smith2020)]") {
		t.Errorf("slide 1 = %q, want [[1]] (doe2019, first seen) and [[2]] (smith2020)", out[1])
	}
	if !strings.Contains(out[2], "[[1](#doe2019)]") {
		t.Errorf("slide 2 = %q, want repeated citation to reuse [[1]]", out[2])
	}

	refs := out[len(out)-1]
	if !strings.HasPrefix(refs, "## References") {
		t.Errorf("references slide should start with heading, got %q", refs)
	}
	if !strings.Contains(refs, "[1] J. Doe") {
		t.Errorf("references slide missing entry 1 (doe2019): %q", refs)
	}
	if !strings.Contains(refs, "[2] J. Smith") {
		t.Errorf("references slide missing entry 2 (smith2020): %q", refs)
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
