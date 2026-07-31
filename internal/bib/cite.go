package bib

import (
	"fmt"
	"regexp"
	"strings"
)

var citeRe = regexp.MustCompile(`\[@([A-Za-z0-9_:.-]+)\]`)

// Process scans slides in order for [@key] citations, numbers them by
// first appearance, rewrites each [@key] into a [n] markdown link to a
// #key anchor (it never resolves anywhere -- it's just how glamour picks
// up the theme's link_text color without hardcoding one), and appends a
// References slide listing every cited entry in citation-number order.
// Citation keys not found in entries are left as literal [@key] text.
func Process(slides []string, entries map[string]Entry) []string {
	numbers := make(map[string]int)
	var order []string

	for _, slide := range slides {
		for _, m := range citeRe.FindAllStringSubmatch(slide, -1) {
			key := m[1]
			if _, ok := entries[key]; !ok {
				continue
			}
			if _, seen := numbers[key]; !seen {
				numbers[key] = len(order) + 1
				order = append(order, key)
			}
		}
	}
	if len(order) == 0 {
		return slides
	}

	out := make([]string, len(slides))
	for i, slide := range slides {
		out[i] = citeRe.ReplaceAllStringFunc(slide, func(match string) string {
			key := citeRe.FindStringSubmatch(match)[1]
			n, ok := numbers[key]
			if !ok {
				return match
			}
			return fmt.Sprintf("[[%d](#%s)]", n, key)
		})
	}
	out = append(out, referencesSlide(order, entries))
	return out
}

func referencesSlide(order []string, entries map[string]Entry) string {
	var b strings.Builder
	b.WriteString("## References\n\n")
	for i, key := range order {
		b.WriteString(FormatReference(i+1, entries[key]))
		b.WriteString("\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
